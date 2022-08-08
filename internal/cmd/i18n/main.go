package main

import (
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

/*
gotext not support golang1.18's feature generic currently;
so use ast package to parse files instead;
*/

func main() {
	NewRootCmd().Execute()
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ki18n",
		Short: "kubegems i18n generator",
	}
	cmd.AddCommand(
		CollectCMD(),
		GeneraetCMD(),
	)
	return cmd
}

func CollectCMD() *cobra.Command {
	return &cobra.Command{
		Use:   "collect",
		Short: "collect all i18n variables",
		RunE: func(*cobra.Command, []string) error {
			return collectRawI18nDatas()
		},
	}
}

func GeneraetCMD() *cobra.Command {
	return &cobra.Command{
		Use:   "gen",
		Short: "generate i18n code",
		RunE: func(*cobra.Command, []string) error {
			return generateCodeViaData()
		},
	}
}

func collectRawI18nDatas() error {
	data := make(map[string]string)

	getI18nVariables := func(filename string) {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filename, string(content), 0)
		if err != nil {
			log.Fatal(err)
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pack, ok := fn.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pack.Name != "i18n" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}

			var expr ast.Expr
			if fn.Sel.Name == "Fprintf" {
				expr = call.Args[1]
			} else {
				expr = call.Args[0]
			}
			str, ok := expr.(*ast.BasicLit)
			if !ok {
				return true
			}
			data[str.Value] = str.Value
			return true
		})
	}

	walkfn := func(filepath string, dir fs.DirEntry, err error) error {
		if dir.Name() == "vendor" {
			return errors.New("skip vendor dircetory")
		}
		if !strings.HasSuffix(filepath, ".go") {
			return nil
		}
		getI18nVariables(filepath)
		return nil
	}
	filepath.WalkDir(".", walkfn)

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("translations/en_US/data.json", content, 0664)
	if err != nil {
		return err
	}
	log.Println("succeed")
	return nil
}

var (
	i18nTmpl = template.Must(template.New("i18n").Funcs(funcs).Parse(`package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

{{- range $k, $v := .Data }}
// {{ funcName $k }} will init {{ $k }} support.
func {{ funcName $k }}(tag language.Tag) {
	{{- range $k, $v := $v }}
	_ = message.SetString(tag, {{$k}}, {{$v}})
{{- end }}
}
{{- end }}

`))
	funcs = template.FuncMap{
		"funcName": func(lang string) string {
			lang = strings.ReplaceAll(lang, "_", "")
			lang = strings.ToUpper(lang[:1]) + lang[1:]
			return "init" + lang
		},
	}
)

const (
	translationPath   = "translations"
	generatedFilename = "pkg/i18n/generated.go"
)

func generateCodeViaData() error {
	fi, err := ioutil.ReadDir(translationPath)
	if err != nil {
		return err
	}
	generatedFile, err := os.Create(generatedFilename)
	if err != nil {
		return err
	}
	data := make(map[string]*map[string]string)
	for _, v := range fi {
		if !v.IsDir() {
			continue
		}
		dataFiles, err := ioutil.ReadDir(path.Join(translationPath, v.Name()))
		if err != nil {
			log.Fatal(err)
		}
		data[v.Name()] = new(map[string]string)
		for _, file := range dataFiles {
			content, err := ioutil.ReadFile(path.Join(translationPath, v.Name(), file.Name()))
			if err != nil {
				return err
			}
			err = json.Unmarshal(content, data[v.Name()])
			if err != nil {
				return err
			}
		}
	}
	err = i18nTmpl.Execute(generatedFile, struct {
		Data map[string]*map[string]string
	}{
		data,
	})
	if err != nil {
		return err
	}
	log.Println("succeed")
	return nil
}
