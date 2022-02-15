package main

/*
DTO 工具
docs:
form 表单对象生成器(orm 类型, 非orm类型需要重新设计)

1. 表单对象需要实现接口
FromInterface{
	// 将表单对象转换成模型对象
	AsObject() model.Object
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
	SystemRoleCommon: {
		Object: SystemRole,
		Fields: {
			ID: uint,
			RoleCode: string,
			RoleName: string
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
	TenantDetail: {
		Object: Tenant,
		Fields: {
			ID: uint,
			TenantName: string,
			Users: []*UserCommon
		}
	}
}
*/

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	kind := flag.String("kind", "form", "which to generate")
	flag.Parse()
	switch *kind {
	case "form":
		fmt.Println("generate form")
		GenerateForms()
	case "orm":
		fmt.Println("generate orm")
		GenerateOrms()
	default:
		fmt.Printf("failed to generate; unknow kind %s ", *kind)
		os.Exit(1)
	}

}
