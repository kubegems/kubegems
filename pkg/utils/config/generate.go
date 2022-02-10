package config

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

func GenerateConfig(opt interface{}) {
	root := getYamlNode(opt)
	o, e := yaml.Marshal(root)
	if e != nil {
		panic(e)
	}
	fmt.Println(string(o))
}

func getYamlNode(v interface{}) *yaml.Node {
	node := &yaml.Node{}
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Ptr:
		vv = vv.Elem()
		node = getYamlNode(vv.Interface())
	case reflect.Map:
		node.Kind = yaml.MappingNode
		nodes := []*yaml.Node{}
		keys := vv.MapKeys()
		for _, k := range keys {
			nodes = append(nodes, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: k.String(),
			})
			nodes = append(nodes, getYamlNode(vv.MapIndex(k).Interface()))
		}
		node.Content = nodes
	case reflect.Array, reflect.Slice:
		nodes := []*yaml.Node{}
		for idx := 0; idx < vv.Len(); idx++ {
			nodes = append(nodes, getYamlNode(vv.Index(idx).Interface()))
		}
		node.Kind = yaml.SequenceNode
		node.Content = nodes
	case reflect.Struct:
		node.Kind = yaml.MappingNode
		nodes := []*yaml.Node{}
		t := reflect.TypeOf(v)
		for idx := 0; idx < t.NumField(); idx++ {
			field := t.FieldByIndex([]int{idx})
			fieldname := field.Tag.Get("yaml")
			if len(fieldname) == 0 {
				fieldname = strings.ToLower(t.FieldByIndex([]int{idx}).Name)
			}
			if !vv.FieldByIndex([]int{idx}).CanInterface() {
				continue
			}
			nodes = append(nodes, &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       fieldname,
				HeadComment: field.Tag.Get("head_comment"),
				LineComment: field.Tag.Get("line_comment"),
			})
			nodes = append(nodes, getYamlNode(vv.FieldByIndex([]int{idx}).Interface()))
		}
		node.Content = nodes
	default:
		node.Kind = yaml.ScalarNode
		node.Value = fmt.Sprintf("%v", vv.Interface())
	}
	return node
}
