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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
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
	readfile := filepath.Join(chartpath, "values.yaml")
	fmt.Printf("Reading %s\n", readfile)
	valuecontent, err := os.ReadFile(readfile)
	if err != nil {
		return err
	}
	schema, err := GenerateSchema(valuecontent)
	if err != nil {
		return err
	}
	schemacontent, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	writefile := filepath.Join(chartpath, "values.schema.json")
	fmt.Printf("Writing %s\n", writefile)
	return os.WriteFile(writefile, schemacontent, DefaultFilePerm)
}

func GenerateSchema(values []byte) (*spec.Schema, error) {
	node := &yaml.Node{}
	if err := yaml.Unmarshal(values, node); err != nil {
		return nil, err
	}
	schema := nodeSchema(node, "")
	return &schema, nil
}

func nodeSchema(node *yaml.Node, keycomment string) spec.Schema {
	schema := spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"object"}}}
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
			properties[key] = nodeSchema(valueNode, keycomment)
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
			schema.Items = &spec.SchemaOrArray{Schema: &schema}
		} else {
			schemas := []spec.Schema{}
			for _, itemnode := range node.Content {
				schemas = append(schemas, nodeSchema(itemnode, ""))
			}
			schema.Items = &spec.SchemaOrArray{Schemas: schemas}
		}
		schema.Type = spec.StringOrArray{"array"}
	}
	// update from comment
	setExtraProperties(&schema, keycomment)
	return schema
}

// nolint: gomnd
func setExtraProperties(schema *spec.Schema, comment string) {
	if comment == "" {
		return
	}
	extras := map[string]any{}
	buf := bufio.NewReader(strings.NewReader(comment))
	for {
		line, isprefix, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		// no line more 4096 char
		_ = isprefix
		//    # @schema type=number,format=port,max=65535,min=1
		//    # @form true
		i := bytes.IndexRune(line, '@')
		if i < 0 || i == len(line) {
			continue
		}
		splits := strings.SplitN(string(line[i+1:]), " ", 2)
		if len(splits) == 1 {
			continue
		}

		key, opts := splits[0], splits[1]
		//  treat no '=',';' string as plain text
		if !strings.Contains(opts, "=") && !strings.Contains(opts, ";") {
			extras[string(key)] = opts
		} else {
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
			extras[string(key)] = options
		}
	}
	for kind, options := range extras {
		if handler, ok := ExtraPropsHandlers[kind]; ok && handler != nil {
			handler(schema, options)
		} else {
			DefaultOptionHandler(schema, kind, options)
		}
	}
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
