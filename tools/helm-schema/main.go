// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

const DefaultFilePerm = 0o755

var ExtraPropsHandlers = map[string]ExtraPropsHandler{
	"schema": SchemaOptionHandler,
	"param":  NoopOptionHandler, // ignore @param options
}

type ExtraPropsHandler func(schema *spec.Schema, options any)

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf(`  %s [path]...
  Usage:
    Generate value shema from values.yaml for kubegems helm chart or plugin.

  Example:
    %s path/to/helm-chart
`, os.Args[0], os.Args[0])
		os.Exit(1)
	}
	for _, glob := range os.Args[1:] {
		matches, err := filepath.Glob(glob)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}
		for _, path := range matches {
			if err := GenerateWriteSchema(path); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				os.Exit(1)
			}
		}
	}
}

func GenerateWriteSchema(chartpath string) error {
	if filepath.Base(chartpath) == "values.yaml" {
		chartpath = filepath.Dir(chartpath)
	}
	valuesfile := filepath.Join(chartpath, "values.yaml")
	fmt.Printf("Reading %s\n", valuesfile)
	valuecontent, err := os.ReadFile(valuesfile)
	if err != nil {
		return err
	}
	schema, err := GenerateSchema(valuecontent)
	if err != nil {
		return err
	}
	for lang, langschema := range SplitSchemaI18n(schema) {
		if err := writeJson(filepath.Join(chartpath, "i18n", fmt.Sprintf("values.schema.%s.json", lang)), langschema); err != nil {
			return err
		}
	}
	return writeJson(filepath.Join(chartpath, "values.schema.json"), schema)
}

func writeJson(filename string, data any) error {
	schemacontent, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), DefaultFilePerm); err != nil {
		return err
	}
	fmt.Printf("Writing %s\n", filename)
	return os.WriteFile(filename, schemacontent, DefaultFilePerm)
}

// SplitSchemaI18n
// nolint: gocognit
func SplitSchemaI18n(schema *spec.Schema) map[string]*spec.Schema {
	ret := map[string]*spec.Schema{}
	for k, v := range schema.ExtraProps {
		i := strings.IndexRune(k, '.')
		if i < 0 {
			continue
		}
		// this is a dot key,remove from parent
		delete(schema.ExtraProps, k)
		basekey, lang := k[:i], k[i+1:]
		if _, ok := ret[lang]; !ok {
			copyschema := DeepCopySchema(schema)
			removeDotKey(copyschema.ExtraProps)
			ret[lang] = copyschema
		}
		ret[lang].ExtraProps[basekey] = v
	}
	if schema.Items != nil {
		if itemschema := schema.Items.Schema; itemschema != nil {
			for lang, langschema := range SplitSchemaI18n(itemschema) {
				if _, ok := ret[lang]; !ok {
					ret[lang].Items.Schema = DeepCopySchema(schema)
				}
				ret[lang].Items.Schema = langschema
			}
		}
		for i, itemschema := range schema.Items.Schemas {
			for lang, langschema := range SplitSchemaI18n(&itemschema) {
				if _, ok := ret[lang]; !ok {
					ret[lang].Items.Schemas = slices.Clone(schema.Items.Schemas)
				}
				ret[lang].Items.Schemas[i] = *langschema
			}
		}
	}
	for name, val := range schema.Properties {
		for lang, itemlangschema := range SplitSchemaI18n(&val) {
			if _, ok := ret[lang]; !ok {
				ret[lang] = DeepCopySchema(schema)
			}
			ret[lang].Properties[name] = *itemlangschema
		}
	}
	return ret
}

func removeDotKey(kvs map[string]any) {
	maps.DeleteFunc(kvs, func(k string, _ any) bool {
		return strings.ContainsRune(k, '.')
	})
}

func DeepCopySchema(in *spec.Schema) *spec.Schema {
	out := &spec.Schema{}
	raw := bytes.NewBuffer(nil)
	// do not use json as encoder
	gob.NewEncoder(raw).Encode(in)
	gob.NewDecoder(raw).Decode(out)
	return out
}

func GenerateSchema(values []byte) (*spec.Schema, error) {
	node := &yaml.Node{}
	if err := yaml.Unmarshal(values, node); err != nil {
		return nil, err
	}
	return nodeSchema(node, ""), nil
}

func nodeSchema(node *yaml.Node, keycomment string) *spec.Schema {
	schema := &spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"object"}}}
	switch node.Kind {
	case yaml.DocumentNode:
		schema := nodeSchema(node.Content[0], "")
		schema.Schema = "http://json-schema.org/schema#"
		return schema
	case yaml.MappingNode:
		properties := spec.SchemaProperties{}
		for i := 0; i < len(node.Content); i += 2 {
			key, keycomment := node.Content[i].Value, node.Content[i].HeadComment
			valueNode := node.Content[i+1]
			properties[key] = *nodeSchema(valueNode, keycomment)
		}
		schema.Type = spec.StringOrArray{"object"}
		schema.Properties = properties
	case yaml.ScalarNode:
		schema.Default = formatYamlStr(node.Value)
		switch node.Tag {
		case "!!str", "!binary":
			schema.Type = spec.StringOrArray{"string"}
		case "!!int":
			schema.Type = spec.StringOrArray{"integer"}
		case "!!float":
			schema.Type = spec.StringOrArray{"number"}
		case "!!bool":
			schema.Type = spec.StringOrArray{"boolean"}
		case "!!timestamp":
			schema.Type = spec.StringOrArray{"string"}
			schema.Format = "data-time"
		case "!!null":
			schema.Nullable = true
		}
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			schema.Items = nil
		} else if len(node.Content) == 1 {
			schema := nodeSchema(node.Content[0], "")
			schema.Items = &spec.SchemaOrArray{Schema: schema}
		} else {
			schemas := []spec.Schema{}
			for _, itemnode := range node.Content {
				schemas = append(schemas, *nodeSchema(itemnode, ""))
			}
			schema.Items = &spec.SchemaOrArray{Schemas: schemas}
		}
		schema.Type = spec.StringOrArray{"array"}
	}
	// update from comment
	completeFromComment(schema, keycomment)
	return schema
}

func completeFromComment(schema *spec.Schema, comment string) {
	annotaionOptions, leftcomment := parseComment(comment)
	_ = leftcomment
	for key, options := range annotaionOptions {
		switch key {
		// case "title":
		// 	schema.Title = annotationOptionsToString(options)
		// case "description":
		// 	schema.Description = annotationOptionsToString(options)
		default:
			if handler, ok := ExtraPropsHandlers[key]; ok && handler != nil {
				handler(schema, options)
			} else {
				DefaultOptionHandler(schema, key, options)
			}
		}
	}
}

func annotationOptionsToString(val any) string {
	strval, ok := val.(string)
	if !ok {
		return ""
	}
	if formated, ok := formatYamlStr(strval).(string); ok {
		return formated
	}
	return strval
}

// nolint: gomnd
func parseComment(comment string) (map[string]any, string) {
	if comment == "" {
		return nil, ""
	}
	var othercomments []string
	annotations := map[string]any{}

	buf := bufio.NewReader(strings.NewReader(comment))
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		if len(line) == 0 {
			continue
		}
		// example: # @schema type=number,format=port,max=65535,min=1
		// trim prefix '#' and remove leading space
		for i, b := range line[1:] {
			if b != ' ' {
				line = line[i+1:]
				break
			}
		}
		if len(line) == 0 {
			continue
		}
		// start with '@'
		if line[0] != '@' {
			othercomments = append(othercomments, string(line))
			continue
		}
		splits := strings.SplitN(string(line[1:]), " ", 2)
		if len(splits) == 1 {
			continue
		}
		key, opts := splits[0], splits[1]
		//  treat no '=',';' string as plain text
		if !strings.Contains(opts, "=") && !strings.Contains(opts, ";") {
			annotations[key] = opts
			continue
		}
		options := map[string]string{}
		for _, opt := range strings.Split(opts, ";") {
			if opt == "" {
				continue
			}
			if optsplits := strings.SplitN(opt, "=", 2); len(optsplits) == 1 {
				options[optsplits[0]] = ""
			} else {
				options[optsplits[0]] = optsplits[1]
			}
		}
		annotations[key] = options
	}
	return annotations, strings.Join(othercomments, "\n")
}

// nolint: gomnd
func SchemaOptionHandler(schema *spec.Schema, options any) {
	kvs, ok := options.(map[string]string)
	if !ok {
		return
	}
	floatPointer := func(in string) *float64 {
		fval, err := strconv.ParseFloat(in, 32)
		if err != nil {
			return nil
		}
		return &fval
	}
	// performance: direct get val from map key
	for k, v := range kvs {
		switch k {
		case "min", "minmum":
			schema.Minimum = floatPointer(v)
		case "max", "maxmum":
			schema.Maximum = floatPointer(v)
		case "format":
			schema.Format = v
		case "pattern":
			schema.Pattern = v
		case "required":
			schema.Required = strings.Split(v, ",")
		case "default":
			schema.Default = formatYamlStr(v)
		case "nullable":
			if v == "" {
				schema.Nullable = true
			} else {
				schema.Nullable, _ = strconv.ParseBool(v)
			}
		case "example":
			schema.Example = v
		case "enum":
			enums := []any{}
			for _, item := range strings.Split(v, ",") {
				enums = append(enums, formatYamlStr(item))
			}
			schema.Enum = enums
		}
	}
}

func DefaultOptionHandler(schema *spec.Schema, kind string, options any) {
	if schema.ExtraProps == nil {
		schema.ExtraProps = map[string]interface{}{}
	}
	switch val := options.(type) {
	case string:
		schema.ExtraProps[kind] = formatYamlStr(val)
	case map[string]string:
		kvs := map[string]any{}
		for k, v := range val {
			kvs[k] = formatYamlStr(v)
		}
		schema.ExtraProps[kind] = kvs
	}
}

// formatYamlStr convert "true" to bool(true), "123" => int(123)
func formatYamlStr(str string) any {
	into := map[string]any{}
	if err := yaml.Unmarshal([]byte("key: "+str), &into); err != nil {
		return str
	}
	return into["key"]
}

func NoopOptionHandler(schema *spec.Schema, options any) {
}
