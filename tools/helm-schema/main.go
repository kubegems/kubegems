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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
	"github.com/mitchellh/copystructure"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

const DefaultFilePerm = 0o755

var ExtraPropsHandlers = map[string]ExtraPropsHandler{
	"schema": SchemaOptionHandler,
	"param":  NoopOptionHandler, // ignore @param options
	"hidden": HiddenOptionHandler,
	"order":  OrderOptionHandler,
	"title":  TitleOptionHandler,
}

type ExtraPropsHandler func(schema *spec.Schema, options any)

type Options struct {
	// parse all schema include not titled
	IncludeAll bool
}

func main() {
	help := false
	options := Options{
		IncludeAll: false,
	}
	flag.BoolVar(&help, "h", help, "show help")
	flag.BoolVar(&options.IncludeAll, "a", options.IncludeAll, "include all schema")
	flag.Usage = Usage
	flag.Parse()

	if len(os.Args) <= 1 || help {
		flag.Usage()
		return
	}

	for _, glob := range os.Args[1:] {
		matches, err := filepath.Glob(glob)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}
		for _, path := range matches {
			if err := GenerateWriteSchema(path, options); err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				os.Exit(1)
			}
		}
	}
}

func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), `Example:
	%s test/helm-chart/values.yaml
	%s test/helm-chart
	%s test/*

`, os.Args[0], os.Args[0], os.Args[0])
	fmt.Fprint(flag.CommandLine.Output(), "Args:\n")
	flag.PrintDefaults()
}

func GenerateWriteSchema(chartpath string, options Options) error {
	if filepath.Base(chartpath) == "values.yaml" {
		chartpath = filepath.Dir(chartpath)
	}
	valuesfile := filepath.Join(chartpath, "values.yaml")
	fmt.Printf("Reading %s\n", valuesfile)
	valuecontent, err := os.ReadFile(valuesfile)
	if err != nil {
		return err
	}

	schema, err := Generator{AllSchema: options.IncludeAll}.GenerateSchema(valuecontent)
	if err != nil {
		return err
	}
	if schema == nil {
		fmt.Printf("Empty schema of %s\n", valuesfile)
		return nil
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
	if schema == nil {
		return nil
	}
	ret := map[string]*spec.Schema{}
	for k, v := range schema.ExtraProps {
		strv, ok := v.(string)
		if !ok {
			continue
		}
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
		SetSchemaProp(ret[lang], basekey, strv)
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
			if ret[lang].Properties == nil {
				ret[lang].Properties = spec.SchemaProperties{}
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
	out, err := copystructure.Copy(in)
	if err != nil {
		panic(err)
	}
	// nolint: forcetypeassert
	return out.(*spec.Schema)
}

type Generator struct {
	AllSchema bool
	MaxDepth  int
}

func (g Generator) GenerateSchema(values []byte) (*spec.Schema, error) {
	node := &yaml.Node{}
	if err := yaml.Unmarshal(values, node); err != nil {
		return nil, err
	}
	return g.nodeSchema(node, "", 0), nil
}

// nolint: funlen
func (g Generator) nodeSchema(node *yaml.Node, keycomment string, depth int) *spec.Schema {
	if g.MaxDepth > 0 && g.MaxDepth < depth {
		return nil
	}
	schema := &spec.Schema{}
	switch node.Kind {
	case yaml.DocumentNode:
		rootschema := g.nodeSchema(node.Content[0], "", 0)
		if rootschema == nil {
			return nil
		}
		rootschema.Schema = "http://json-schema.org/schema#"
		return rootschema
	case yaml.MappingNode:
		schema.Type = spec.StringOrArray{"object"}
		if schema.Properties == nil {
			schema.Properties = spec.SchemaProperties{}
		}
		for i := 0; i < len(node.Content); i += 2 {
			key, keycomment := node.Content[i].Value, node.Content[i].HeadComment
			objectProperty := g.nodeSchema(node.Content[i+1], keycomment, depth+1)
			if objectProperty == nil {
				continue
			}
			schema.Properties[key] = *objectProperty
		}
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
			schema.Type = spec.StringOrArray{"null"}
		default:
			schema.Type = spec.StringOrArray{"object"}
		}
	case yaml.SequenceNode:
		schema.Type = spec.StringOrArray{"array"}
		var schemas []spec.Schema
		for _, itemnode := range node.Content {
			itemProperty := g.nodeSchema(itemnode, "", depth+1)
			if itemProperty == nil {
				continue
			}
			schemas = append(schemas, *itemProperty)
		}
		if len(schemas) == 1 {
			schema.Items = &spec.SchemaOrArray{Schema: &schemas[0]}
		} else {
			schema.Items = &spec.SchemaOrArray{Schemas: schemas}
		}
	}
	// update from comment
	completeFromComment(schema, keycomment)
	// depth > 0 in case of root schema has no title field.
	if depth > 0 && !g.AllSchema && schema.Title == "" {
		return nil
	}
	return schema
}

func completeFromComment(schema *spec.Schema, comment string) {
	annotaionOptions, leftcomment := parseComment(comment)
	_ = leftcomment
	for key, options := range annotaionOptions {
		if schema.ExtraProps == nil {
			schema.ExtraProps = map[string]interface{}{}
		}
		if handler, ok := ExtraPropsHandlers[key]; ok && handler != nil {
			handler(schema, options)
		} else {
			DefaultOptionHandler(schema, key, options)
		}
	}
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
	for k, v := range kvs {
		SetSchemaProp(schema, k, v)
	}
}

func DefaultOptionHandler(schema *spec.Schema, kind string, options any) {
	switch val := options.(type) {
	case string:
		SetSchemaProp(schema, kind, val)
	case map[string]string:
		kvs := map[string]any{}
		for k, v := range val {
			kvs[k] = formatYamlStr(v)
		}
		if schema.ExtraProps == nil {
			schema.ExtraProps = map[string]interface{}{}
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

type HiddenProps struct {
	Operator   HiddenPropsOperator `json:"operator,omitempty"`
	Conditions []HiddenCondition   `json:"conditions,omitempty"`
}

// Ref: https://github.com/kubegems/dashboard/blob/448f9c5767d4232adf4c86b711ae252f5a9e43de/src/views/appstore/components/DeployWizard/Param/index.vue#L160-L185
const (
	HiddenPropsOperatorOr  = "or"
	HiddenPropsOperatorAnd = "and"
	HiddenPropsOperatorNor = "nor"
	HiddenPropsOperatorNot = "not"
)

type HiddenPropsOperator string

type HiddenCondition struct {
	Path  string `json:"path"`
	Value any    `json:"value"`
}

func HiddenOptionHandler(schema *spec.Schema, options any) {
	kvs, ok := options.(map[string]string)
	if !ok {
		return
	}
	// convert map k=v to object type {path=jsonpath, value=value}
	operator := kvs["operator"]
	delete(kvs, "operator")
	if len(kvs) == 0 {
		return
	}
	if len(kvs) == 1 {
		// simple type
		for k, v := range kvs {
			// case : foo!=bar key=foo!
			if strings.HasSuffix(k, "!") {
				schema.ExtraProps["hidden"] = HiddenProps{
					Operator: HiddenPropsOperatorNot,
					Conditions: []HiddenCondition{
						{
							Path:  strings.TrimSuffix(k, "!"),
							Value: formatYamlStr(v),
						},
					},
				}
				return
			}
			schema.ExtraProps["hidden"] = HiddenCondition{Path: k, Value: formatYamlStr(v)}
			return
		}
	}
	if operator == "" {
		operator = HiddenPropsOperatorOr
	}
	props := HiddenProps{
		Operator: HiddenPropsOperator(operator),
	}
	for k, v := range kvs {
		props.Conditions = append(props.Conditions, HiddenCondition{Path: k, Value: formatYamlStr(v)})
	}
	schema.ExtraProps["hidden"] = props
}

// OrderOptionHandler handle properties order
// Ref: https://github.com/go-openapi/spec/blob/1005cfb91978aa416cfc5a1251b790126390788a/properties.go#L44
func OrderOptionHandler(schema *spec.Schema, options any) {
	schema.AddExtension("x-order", options)
}

func TitleOptionHandler(schema *spec.Schema, options any) {
	title, ok := options.(string)
	if !ok {
		return
	}
	DefaultOptionHandler(schema, "title", formatYamlStr(title))
	// automate add @form=true when @title exists
	if schema.Title != "" {
		SetSchemaProp(schema, "form", "true")
	}
}

// nolint: gomnd,funlen
func SetSchemaProp(schema *spec.Schema, k string, v string) {
	floatPointer := func(in string) *float64 {
		fval, err := strconv.ParseFloat(in, 32)
		if err != nil {
			return nil
		}
		return &fval
	}
	int64Pointer := func(in string) *int64 {
		ival, err := strconv.ParseInt(in, 10, 64)
		if err != nil {
			return nil
		}
		return &ival
	}
	switch k {
	case "min", "minmum":
		schema.Minimum = floatPointer(v)
	case "minLength", "minLen", "minlen":
		schema.MinLength = int64Pointer(v)
	case "maxLength", "maxLen", "maxlen":
		schema.MaxLength = int64Pointer(v)
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
	case "title":
		schema.Title = v
	case "enum":
		enums := []any{}
		for _, item := range strings.Split(v, ",") {
			enums = append(enums, formatYamlStr(item))
		}
		schema.Enum = enums
	case "description":
		schema.Description = v
	case "type":
		if !schema.Type.Contains(v) {
			schema.Type = append(schema.Type, v)
		}
	default:
		if schema.ExtraProps == nil {
			schema.ExtraProps = map[string]any{}
		}
		schema.ExtraProps[k] = formatYamlStr(v)
	}
}
