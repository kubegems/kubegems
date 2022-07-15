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
	"bytes"
	"fmt"
	"go/doc"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"io/ioutil"
	"regexp"
	"strings"
)

/*
Object 接口生成器
根据注释给所有的对象生成相应类型接口方法
*/

func GenerateOrms() {
	fset := token.NewFileSet()
	pkgs, first := parser.ParseDir(fset, "../orm", func(f fs.FileInfo) bool { return strings.HasPrefix(f.Name(), "m_") }, parser.ParseComments)
	if first != nil {
		panic(first)
	}
	var vars []string
	for _, f := range pkgs {
		p := doc.New(f, "./", 0)
		for _, t := range p.Types {
			parseDoc(t.Name, t.Doc, &vars)
		}
	}
	genCode(vars)
}

func parseDoc(typename, doc string, vars *[]string) {
	lines := strings.Split(doc, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+gen ") {
			content := strings.TrimPrefix(line, "+gen ")
			options := strings.Split(content, " ")
			optionMap := map[string]string{}
			for _, option := range options {
				option := strings.TrimSpace(option)
				if len(option) == 0 {
					continue
				}
				opts := strings.SplitN(option, ":", 2)
				if len(opts) != 2 {
					panic(fmt.Errorf("error format comment label for type %v: %v", typename, option))
				}
				key := opts[0]
				value := opts[1]
				optionMap[key] = value
			}
			*vars = append(*vars, gen(typename, optionMap)...)
		}
	}
}

func genCode(list []string) {
	code := `package orm `

	lines := append([]string{code}, list...)
	ret := strings.Join(lines, "\n")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", ret, 0)
	if err != nil {
		panic(err)
	}

	buffer := bytes.NewBufferString("")
	printer.Fprint(buffer, fset, f)
	rret, err := format.Source(buffer.Bytes())
	if err != nil {
		panic(err)
	}

	if e := ioutil.WriteFile("../orm/zz_generated.go", rret, fs.FileMode(0644)); e != nil {
		panic(e)
	}
}

func gen(typename string, optionMap map[string]string) []string {
	var ret []string
	for key, value := range optionMap {
		switch key {
		case "type":
			switch value {
			case "object":
				ret = append(ret, genVars(typename, optionMap))
				ret = append(ret, genObjectFunctions(typename, optionMap))
				ret = append(ret, genObjectList(typename))
				ret = append(ret, genObjectListFunctions(typename, optionMap))
			case "objectrel":
				ret = append(ret, genVars(typename, optionMap))
				ret = append(ret, genObjectFunctions(typename, optionMap))
				// ret = append(ret, genObjectRelFunctions(typename, optionMap))
				ret = append(ret, genObjectList(typename))
				ret = append(ret, genObjectListFunctions(typename, optionMap))
			}
		}
	}
	return ret
}
func genVars(typename string, optionMap map[string]string) string {
	var ret []string
	tpl := `var (
		%sKind = "%s"
		%sTable = "%s" 
		%sPrimaryKey = "%s"
		%sValidPreloads = []string{%s}
	)`
	var (
		pk      string
		preload []string
	)
	if volume, exist := optionMap["pk"]; exist {
		pk = volume
	} else {
		pk = "id"
	}

	if preloads, exist := optionMap["preloads"]; exist {
		preload = strings.Split(preloads, ",")
	}

	tmp := []string{}
	for _, p := range preload {
		tmp = append(tmp, "\""+p+"\"")
	}

	args := []interface{}{
		camelCase(typename), toSnakeCase(typename),
		camelCase(typename), tableName(typename),
		camelCase(typename), pk,
		camelCase(typename), strings.Join(tmp, ", "),
	}
	ret = append(ret, fmt.Sprintf(tpl, args...))
	return strings.Join(ret, "\n")
}

func genObjectFunctions(typename string, optionMap map[string]string) string {
	v := `func (obj *%s) TableName() *string {
	return &%sTable
}

func (obj *%s) GetKind() *string {
	return &%sKind
}

func (obj *%s) PrimaryKeyField() *string {
	return &%sPrimaryKey
}

func (obj *%s) PrimaryKeyValue() interface{} {
	return obj.%s
}

func (obj *%s) PreloadFields() *[]string {
	return &%sValidPreloads
}
`
	var pkField string
	if field, exist := optionMap["pkfield"]; exist {
		pkField = field
	} else {
		pkField = "ID"
	}
	args := []interface{}{
		typename, camelCase(typename),
		typename, kindName(typename),
		typename, camelCase(typename),
		typename, pkField,
		typename, camelCase(typename),
	}

	return fmt.Sprintf(v, args...)
}

func genObjectListFunctions(typename string, optionMap map[string]string) string {
	v := `func (objList *%sList) GetKind() *string {
	return &%sKind
}

func (obj *%sList) PrimaryKeyField() *string {
	return &%sPrimaryKey
}

func (objList *%sList) GetPageSize() (*int64, *int64) {
	return &objList.Page, &objList.Size
}

func (objList *%sList) SetPageSize(page, size int64) {
	objList.Page = page
	objList.Size = size
}

func (objList *%sList) GetTotal() *int64 {
	return &objList.Total
}

func (objList *%sList) SetTotal(total int64) {
	objList.Total = total
}

func (objList *%sList) DataPtr() interface{} {
	return &objList.Items
}
`
	args := []interface{}{
		typename, kindName(typename),
		typename, kindName(typename),
		typename,
		typename,
		typename,
		typename,
		typename,
	}

	return fmt.Sprintf(v, args...)
}

/*
func genObjectRelFunctions(typename string, optionMap map[string]string) string {
	var (
		leftObject  string
		rightObject string
	)
	if left, exist := optionMap["leftfield"]; exist {
		leftObject = left
	}
	if right, exist := optionMap["rightfield"]; exist {
		rightObject = right
	}
	v := `func (obj *%s) Left() client.Object {
	return obj.%s
}

func (obj *%s) Right() client.Object {
	return obj.%s
}
`
	args := []interface{}{
		typename, leftObject,
		typename, rightObject,
	}

	return fmt.Sprintf(v, args...)
}
*/

func genObjectList(typename string) string {
	v := `type %sList struct {
	Items []*%s
	BaseList
}`
	return fmt.Sprintf(v, typename, typename)

}

func camelCase(in string) string {
	inbytes := []byte(in)
	return string(append(bytes.ToLower(inbytes[:1]), inbytes[1:]...))
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func kindName(in string) string {
	var k string
	if strings.HasSuffix(in, "List") {
		k = strings.TrimSuffix(in, "List")
	} else {
		k = in
	}
	return camelCase(k)
}

func tableName(typename string) string {
	t := toSnakeCase(typename)
	if strings.HasSuffix(t, "s") {
		return t
	}
	return t + "s"
}
