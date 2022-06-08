package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io/fs"
	"io/ioutil"
	"sort"
	"strings"
	"text/template"
)

type TypeDefine struct {
	Name   string
	Object string
	Fields map[string]string
}

func GenerateForms() {
	formfset := token.NewFileSet()
	formpkgs, first := parser.ParseDir(formfset, "../forms", func(f fs.FileInfo) bool { return strings.HasPrefix(f.Name(), "f_") }, parser.ParseComments)
	if first != nil {
		panic(first)
	}
	objectMapping := getObjectMapping(formpkgs)

	ormfset := token.NewFileSet()
	ormpkgs, first := parser.ParseDir(ormfset, "../orm", func(f fs.FileInfo) bool { return strings.HasPrefix(f.Name(), "m_") }, parser.ParseComments)
	if first != nil {
		panic(first)
	}

	forms := parseFields(formpkgs, objectMapping)
	orms := parseFields(ormpkgs, nil)

	genFormCode(forms, orms)
}

func parseFields(pkgs map[string]*ast.Package, objectMapping map[string]string) map[string]TypeDefine {
	structFields := map[string]TypeDefine{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch v := decl.(type) {
				case *ast.GenDecl:
					for _, spec := range v.Specs {
						switch typeSpec := spec.(type) {
						case *ast.TypeSpec:
							var ormObj string
							if objectMapping != nil {
								ormObj = objectMapping[typeSpec.Name.String()]
							}
							def := TypeDefine{
								Name:   typeSpec.Name.String(),
								Object: ormObj,
								Fields: make(map[string]string),
							}
							switch typeSpec.Type.(type) {
							case *ast.StructType:
								structType := typeSpec.Type.(*ast.StructType)
								for _, field := range structType.Fields.List {
									if i, ok := field.Type.(*ast.Ident); ok {
										fieldType := i.Name
										for _, name := range field.Names {
											def.Fields[name.Name] = fieldType
										}
									} else {
										for _, name := range field.Names {
											def.Fields[name.Name] = types.ExprString(field.Type)
										}
									}
								}
							}
							structFields[typeSpec.Name.String()] = def
						}
					}
				}
			}
		}
	}
	return structFields
}

func getObjectMapping(pkgs map[string]*ast.Package) map[string]string {
	ret := make(map[string]string)
	for _, pkg := range pkgs {
		p := doc.New(pkg, "./", 0)

		for _, t := range p.Types {
			lines := strings.Split(t.Doc, "\n")
			var genline string
			for _, line := range lines {
				if strings.HasPrefix(line, "+genform") {
					genline = strings.TrimPrefix(line, "+genform ")
				}
			}
			if genline == "" {
				continue
			}
			seps := strings.Split(genline, " ")
			for _, sep := range seps {
				sseps := strings.Split(sep, ":")
				if len(sseps) != 2 {
					continue
				}
				if sseps[0] == "object" {
					ret[t.Name] = sseps[1]
				}
			}
		}
	}
	return ret
}

func writeCode(code, dest string) {

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", code, 0)
	if err != nil {
		panic(err)
	}

	buffer := bytes.NewBufferString("")
	printer.Fprint(buffer, fset, f)
	rret, err := format.Source(buffer.Bytes())
	if err != nil {
		panic(err)
	}

	if e := ioutil.WriteFile(dest, rret, fs.FileMode(0644)); e != nil {
		panic(e)
	}
}

func genFormCode(forms, orms map[string]TypeDefine) {
	head := `package forms
import (
	"kubegems.io/kubegems/pkg/model/orm"
	"kubegems.io/kubegems/pkg/model/client"
)`
	r := []string{head}
	keys := []string{}
	for k, v := range forms {
		if v.Object != "" {
			keys = append(keys, k)
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	for _, k := range keys {
		v := forms[k]

		orm, exist := orms[v.Object]
		if !exist {
			continue
		}

		r = append(r, gen_FORM_LIST(k))
		r = append(r, gen_FORM_LIST_FUNCS(k, v.Object))
		r = append(r, gen_ToObject(k, v.Object))
		r = append(r, gen_Data(k, v.Object))
		r = append(r, gen_Convert_FORM_ORM(k, v.Object, v.Fields, orm.Fields, forms))
		r = append(r, gen_Convert_ORM_FORM(k, v.Object, v.Fields, orm.Fields, forms))
		r = append(r, gen_Convert_FORM_ORM_slice(k, v.Object, v.Fields, orm.Fields, forms))
		r = append(r, gen_Convert_ORM_FORM_slice(k, v.Object, v.Fields, orm.Fields, forms))
	}
	code := strings.Join(r, "\n")
	writeCode(code, "../forms/zz_generated.go")

}

func gen_ToObject(formtype, ormtype string) string {
	tpl := `func (r *%s) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_%s_%s(r) 
	}
	return r.object
}

`
	return fmt.Sprintf(tpl, formtype, formtype, ormtype)
}

func gen_Data(formtype, ormtype string) string {
	tpl := `func (u *{{ .FormType }}) Data() *{{ .FormType }} {
	if u.data != nil {
		return u.data.(*{{ .FormType }})
	}
	tmp := Convert_{{ .OrmType }}_{{ .FormType }}(u.object.(*orm.{{ .OrmType }}))
	u.data = tmp
	return tmp
	}

	func (u *{{ .FormType }}) DataPtr() interface{} {
		return u.Data()
	}
`
	return Render(tpl, map[string]string{"FormType": formtype, "OrmType": ormtype})
}

func gen_Convert_FORM_ORM(formtype, ormtype string, fields, ofields map[string]string, forms map[string]TypeDefine) string {
	fs := []string{}
	sortedFields := sortedFields(fields)
	for _, field := range sortedFields {
		ftype := fields[field]
		if _, exist := ofields[field]; !exist {
			continue
		}
		rftype, isarr := getRealType((ftype))
		if define, exist := forms[rftype]; exist {
			if isarr {
				fs = append(fs, fmt.Sprintf("r.%s = Convert_%s_%s_slice(f.%s)", field, rftype, define.Object, field))
			} else {
				fs = append(fs, fmt.Sprintf("r.%s = Convert_%s_%s(f.%s)", field, rftype, define.Object, field))
			}
		} else {
			fs = append(fs, fmt.Sprintf("r.%s = f.%s", field, field))
		}
	}
	fss := strings.Join(fs, "\n\t")

	tpl := `func Convert_{{ .FormType }}_{{ .OrmType }}(f *{{ .FormType }}) *orm.{{ .OrmType }} {
	r := &orm.{{ .OrmType }}{}
	if f == nil {
		return nil
	}
	f.object = r
	{{ .FSS }}
	return r
}`
	ctx := map[string]string{"OrmType": ormtype, "FormType": formtype, "FSS": fss}
	return Render(tpl, ctx)
}

func gen_Convert_ORM_FORM(formtype, ormtype string, fields, ofields map[string]string, forms map[string]TypeDefine) string {
	fs := []string{}
	sortedFields := sortedFields(fields)
	for _, field := range sortedFields {
		ftype := fields[field]
		if _, exist := ofields[field]; !exist {
			continue
		}
		rftype, isarr := getRealType((ftype))
		if define, exist := forms[rftype]; exist {
			if isarr {
				fs = append(fs, fmt.Sprintf("r.%s = Convert_%s_%s_slice(f.%s)", field, define.Object, rftype, field))
			} else {
				fs = append(fs, fmt.Sprintf("r.%s = Convert_%s_%s(f.%s)", field, define.Object, rftype, field))
			}
		} else {
			fs = append(fs, fmt.Sprintf("r.%s = f.%s", field, field))
		}
	}
	fss := strings.Join(fs, "\n\t")

	tpl := `func Convert_{{ .OrmType }}_{{ .FormType }}(f *orm.{{ .OrmType }}) *{{ .FormType }} {
	if f == nil {
		return nil
	}
	var r {{ .FormType }}
	{{ .FSS }}
	return &r
}`
	ctx := map[string]string{"OrmType": ormtype, "FormType": formtype, "FSS": fss}
	return Render(tpl, ctx)
}

func gen_Convert_ORM_FORM_slice(formtype, ormtype string, fields, ofileds map[string]string, forms map[string]TypeDefine) string {
	tpl := `func Convert_{{ .OrmType }}_{{ .FormType }}_slice(arr []*orm.{{ .OrmType }}) []*{{ .FormType }} {
	r := []*{{ .FormType }}{}
	for _, u := range arr {
		r = append(r, Convert_{{ .OrmType }}_{{ .FormType }}(u))
	}
	return r
}
	`
	ctx := map[string]string{"FormType": formtype, "OrmType": ormtype}
	return Render(tpl, ctx)
}

func gen_Convert_FORM_ORM_slice(formtype, ormtype string, fields, ofileds map[string]string, forms map[string]TypeDefine) string {
	tpl := `func Convert_{{ .FormType }}_{{ .OrmType }}_slice(arr []*{{ .FormType }}) []*orm.{{ .OrmType }} {
	r := []*orm.{{ .OrmType }}{}
	for _, u := range arr {
		r = append(r, Convert_{{ .FormType }}_{{ .OrmType }}(u))
	}
	return r
}
`
	ctx := map[string]string{"FormType": formtype, "OrmType": ormtype}
	return Render(tpl, ctx)
}

func gen_FORM_LIST(formtype string) string {
	tpl := `type {{ .FormType }}List struct {
	BaseListForm
	Items []*{{ .FormType }}
}
	`
	ctx := map[string]string{"FormType": formtype}
	return Render(tpl, ctx)
}

func gen_FORM_LIST_FUNCS(formtype, ormtype string) string {
	tpl := `func (ul *{{ .FormType }}List) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.{{ .OrmType }}List{}
	return ul.objectlist
}

func (ul *{{ .FormType }}List) Data() []*{{ .FormType }} {
	if ul.data != nil {
		return ul.data.([]*{{ .FormType }})
	}
	us := ul.objectlist.(*orm.{{ .OrmType }}List)
	tmp := Convert_{{ .OrmType }}_{{ .FormType }}_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *{{ .FormType }}List) DataPtr() interface{} {
	return ul.Data()
}
`
	ctx := map[string]string{"FormType": formtype, "OrmType": ormtype}
	return Render(tpl, ctx)
}

func Render(tpl string, ctx map[string]string) string {
	tp, e := template.New("").Parse(tpl)
	if e != nil {
		panic(e)
	}
	buf := bytes.NewBufferString("")
	if e := tp.Execute(buf, ctx); e != nil {
		panic(e)
	}
	return buf.String()
}

func getRealType(s string) (string, bool) {
	var (
		r     string
		isarr bool
	)
	if strings.HasPrefix(s, "[]") {
		isarr = true
	}
	r = strings.TrimLeft(s, "[]* ")

	return r, isarr
}

func sortedFields(m map[string]string) []string {
	ret := []string{}
	for field := range m {
		ret = append(ret, field)
	}
	sort.Strings(ret)
	return ret
}
