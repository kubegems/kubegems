package main

// go:generate go run . form,orm
/*
code generator;

1. 表单对象需要实现接口
FromInterface{
	// 将表单对象转换成模型对象
	Object() model.Object
	// 将模型对象转换成表单对象
	Data() FormInterface
}

需要对两个对象(FORM, ORM)实现
func Convert_FORM_ORM(f *FORM) *ORM {
	o := &ORM{}
	o.xxx = f.xxx
	...
	o.NESTED = Convert_NESTED_NESTEDORM(f.NESTED)
	return o
}

func Convert_ORM_FORM(o *ORM, f *FORM) {
	f.xxx = o.xxx
	...
}

func Convert_FORM_ORM_arr(fs []*FORM) []*ORM {
	os := []*ORM{}
	for _, f := range fs {
		os = append(os, Convert_FORM_ORM(f))
	}
	return os
}

Convert_ORM_FORM_arr(os []*ORM) []*FORM {
	ret := []*FORM{}
	for _, o := range os {
		var tmp  FORM
		Convert_ORM_FORM(o, &FORM)
		ret = append(ret, &tmp)
	}
	return ret
}

----
Form - Object 关系映射
{
	UserCommon: {
		Object: User,
		Fields: {
			ID: uint,
			Username: string,
			Email: string
		}
	},
	UserDetail: {
		Object: User,
		Fields: {
			ID: uint,
			Username: string,
			Email: string
			SystemRole: *SystemRoleCommon,
			Tenants: []*TenantCommon
		}
	},
	SystemRoleDetail: {
		Object: SystemRole,
		Fields: {
			ID: uint,
			RoleCode: string,
			RoleName: string
			Users: []*UserCommon
		}
	},
	TenantCommon: {
		Object: Tenant,
		Fields: {
			ID: uint,
			TenantName: string,
		}
	},
	...
}
*/

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	flag.Parse()
	arg := flag.Arg(0)
	if arg == "" {
		log.Println("options required; eg: orm")
		os.Exit(1)
	}
	kinds := strings.Split(arg, ",")
	var (
		genForm, genOrm, hasErr bool
	)
	for idx := range kinds {
		switch kinds[idx] {
		case "form":
			log.Println("generate form")
			genForm = true
		case "orm":
			log.Println("generate orm")
			genOrm = true
		default:
			log.Println(fmt.Sprintf("not support option %s", kinds[idx]))
			hasErr = true
		}
	}
	if hasErr {
		os.Exit(1)
	}
	if genForm {
		GenerateForms()
	}
	if genOrm {
		GenerateOrms()
	}
}
