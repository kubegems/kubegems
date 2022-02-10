package workflow

import (
	"reflect"
)

func IdentityKeyOfFunction(fun interface{}) string {
	t := reflect.ValueOf(fun).Type()
	if t.Kind() != reflect.Func {
		return "__not_function__"
	}

	str := t.String()
	_ = str

	m := t.Method(0)

	_ = m

	name := t.Name()
	pkg := t.PkgPath()

	return pkg + name
}
