package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"kubegems.io/pkg/log"
)

var ErrCantRegister = errors.New("can't register flag")

func GenerateConfig(opt interface{}) {
	root := ToYamlNode(ParseStruct(opt))
	o, e := yaml.Marshal(root)
	if e != nil {
		panic(e)
	}
	fmt.Println(string(o))
}

type Node struct {
	Name     string
	Kind     reflect.Kind
	Tag      reflect.StructTag
	Value    reflect.Value
	Children []Node
}

func ToYamlNode(node Node) *yaml.Node {
	n := &yaml.Node{
		Content: make([]*yaml.Node, 0, len(node.Children)),
	}
	switch node.Kind {
	case reflect.Struct, reflect.Map:
		n.Kind = yaml.MappingNode
		n.Value = node.Name
		for _, v := range node.Children {
			comment := v.Tag.Get("help")
			if comment == "" {
				comment = v.Tag.Get("description")
			}
			in := &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       v.Name,
				LineComment: comment,
			}
			n.Content = append(n.Content, in)
			n.Content = append(n.Content, ToYamlNode(v))
		}
	case reflect.Array, reflect.Slice:
		n.Kind = yaml.SequenceNode
		for _, v := range node.Children {
			n.Content = append(n.Content, ToYamlNode(v))
		}
	default:
		n.Kind = yaml.ScalarNode
		n.Value = fmt.Sprintf("%v", node.Value.Interface())
	}
	return n
}

func prefixedKey(prefix, key string, splitor ...string) string {
	if len(prefix) == 0 {
		return strings.ToLower(key)
	}

	spl := "-"
	if len(splitor) > 0 {
		spl = string(splitor[0])
	}
	return strings.ToLower(prefix + spl + key)
}

func registerFlagSet(fs *pflag.FlagSet, prefix string, nodes []Node) {
	for _, node := range nodes {
		key := prefixedKey(prefix, node.Name)
		switch node.Kind {
		case reflect.Struct, reflect.Map:
			registerFlagSet(fs, key, node.Children)
		default:
			short := node.Tag.Get("short")
			description := node.Tag.Get("description")
			if !node.Value.CanAddr() {
				log.Error(ErrCantRegister, "key", key, "value", node.Value.Interface())
				continue
			}
			v := node.Value.Addr().Interface()
			switch value := v.(type) {
			case *string:
				fs.StringVarP(value, key, short, *value, description)
			case *bool:
				fs.BoolVarP(value, key, short, *value, description)
			case *int:
				fs.IntVarP(value, key, short, *value, description)
			case *int64:
				fs.Int64VarP(value, key, short, *value, description)
			case *uint16:
				fs.Uint16VarP(value, key, short, *value, description)
			case *[]bool:
				fs.BoolSliceVarP(value, key, short, *value, description)
			case *time.Duration:
				fs.DurationVarP(value, key, short, *value, description)
			case *float32:
				fs.Float32VarP(value, key, short, *value, description)
			case *float64:
				fs.Float64VarP(value, key, short, *value, description)
			case *[]string:
				fs.StringSliceVarP(value, key, short, *value, description)
			default:
				log.Error(ErrCantRegister, "unrecognized value type", "key", key, "kind", node.Kind)
			}
		}
	}
}

func ParseStruct(data interface{}) Node {
	v := reflect.Indirect(reflect.ValueOf(data))
	return complete(Node{}, v)
}

func ToJsonPathes(prefix string, nodes []Node) []KV {
	return toJsonPathes(prefix, nodes, []KV{})
}

type KV struct {
	Key   string
	Value interface{}
}

func toJsonPathes(prefix string, nodes []Node, kvs []KV) []KV {
	for _, node := range nodes {
		switch node.Kind {
		case reflect.Struct, reflect.Map:
			kvs = toJsonPathes(prefixedKey(prefix, node.Name, "."), node.Children, kvs)
		default:
			kvs = append(kvs, KV{
				Key:   prefixedKey(prefix, node.Name, "."),
				Value: node.Value.Interface(),
			})
		}
	}
	return kvs
}

func complete(node Node, v reflect.Value) Node {
	v = reflect.Indirect(v)

	node.Kind = v.Kind()
	node.Value = v

	var children []Node
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fi := v.Type().Field(i)

			// unexported
			if fi.PkgPath != "" {
				continue
			}

			opts := fi.Tag.Get("json")
			if opts == "" {
				opts = fi.Tag.Get("yaml")
			}

			jsonopts := strings.Split(opts, ",")

			if fi.Anonymous || (len(jsonopts) > 1 && jsonopts[1] == "inline") {
				children = append(children, complete(Node{}, v.Field(i)).Children...)
				continue
			}

			name := jsonopts[0]
			if name == "" {
				name = fi.Name
			}
			in := Node{
				Name: name,
				Tag:  fi.Tag,
			}
			children = append(children, complete(in, v.Field(i)))
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			in := Node{
				Name: k.String(),
			}
			children = append(children, complete(in, v.MapIndex(k)))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			in := Node{
				Name: strconv.Itoa(i),
			}
			children = append(children, complete(in, v.Index(i)))
		}
	}

	node.Children = children
	return node
}
