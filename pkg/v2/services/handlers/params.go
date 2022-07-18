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

package handlers

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConditionOperator string

var Eq, Gt, Lt, Neq, Gte, Lte, In, Like ConditionOperator = "=", ">", "<", "<>", ">=", "<=", "in", "like"

var (
	QueryPageNum  = restful.QueryParameter("page", "page number")
	QueryPageSize = restful.QueryParameter("size", "page size")
	QuerySearch   = restful.QueryParameter("search", "search condition")
	QueryOrder    = restful.QueryParameter("order", "order")
)

type Cond struct {
	Field string
	Op    ConditionOperator
	Value interface{}
}

func (cond *Cond) AsQuery() (string, interface{}) {
	return fmt.Sprintf("%s %s ?", cond.Field, cond.Op), cond.Value
}

func ListCommonQuery(rb *restful.RouteBuilder) *restful.RouteBuilder {
	return rb.Param(QueryPageNum).Param(QueryPageSize).Param(QuerySearch).Param(QueryOrder)
}

type RelationCondition struct {
	Key   string
	Value interface{}
	Table string
}

func ScopeTable(model interface{}) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table(tableName(tx, model))
	}
}

func ScopeCondition(conds []*Cond, model interface{}) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tname := tableName(tx, model)
		if len(conds) == 0 {
			return tx
		}
		qs := []string{}
		vs := []interface{}{}
		for idx := range conds {
			q, v := conds[idx].AsQuery()
			qs = append(qs, tname+"."+q)
			vs = append(vs, v)
		}
		return tx.Where(strings.Join(qs, " AND "), vs...)

	}
}

func ScopePreload(req *restful.Request, validPreloads []string) func(tx *gorm.DB) *gorm.DB {
	preloads := req.QueryParameters("preload")
	return func(tx *gorm.DB) *gorm.DB {
		tdb := tx
		for _, preloadField := range preloads {
			if validPreloads == nil {
				tdb = tdb.Preload(preloadField)
			} else {
				if contains(validPreloads, preloadField) {
					tdb = tdb.Preload(preloadField)
				}
			}
		}
		return tdb
	}
}

func ScopeOrder(req *restful.Request, valid []string) func(tx *gorm.DB) *gorm.DB {
	orders := req.QueryParameters("order")
	return func(tx *gorm.DB) *gorm.DB {
		tdb := tx
		for _, orderStr := range orders {
			tdb = tdb.Order(orderStr)
		}
		return tdb
	}
}

func ScopePageSize(req *restful.Request) func(tx *gorm.DB) *gorm.DB {
	pInt, err := strconv.Atoi(req.PathParameter("page"))
	if err != nil {
		pInt = 1
	}
	sInt, err := strconv.Atoi(req.PathParameter("page"))
	if err != nil {
		sInt = 10
	}
	if pInt <= 0 {
		pInt = 1
	}
	if sInt <= 0 {
		sInt = 10
	}
	nopager := req.PathParameter("_nopage") == "true"
	return func(tx *gorm.DB) *gorm.DB {
		if nopager {
			return tx
		}
		tdb := tx
		offset := sInt * (pInt - 1)
		tdb.Offset(offset).Limit(sInt)
		tdb.Set("page", pInt)
		tdb.Set("size", sInt)
		return tdb
	}
}

func ScopeFields(model interface{}, fields []string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if len(fields) == 0 {
			return tx
		}
		stmt1 := (&gorm.Statement{DB: tx})
		stmt1.Parse(model)
		tableFields := make([]string, len(fields))
		tableName := stmt1.Schema.Table
		for idx := range fields {
			tableFields[idx] = tableName + "." + fields[idx]
		}
		tdb := tx
		tdb.Select(strings.Join(tableFields, ","))
		return tdb
	}
}

func ScopeBelongViaField(model, selfmodel interface{}, cond *Cond, relfield string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tablename := tableName(tx, model)
		selftable := tableName(tx, selfmodel)
		tdb := tx.Joins(fmt.Sprintf("LEFT JOIN %s on %s.id = %s.%s", tablename, tablename, selftable, relfield))
		return tdb.Where(fmt.Sprintf("%s.%s %s ?", tablename, cond.Field, cond.Op), cond.Value)
	}
}

/*
m2m relations
eg:
	tables:
	1. table tenants
		id 			uint
		tenant_name string

	2. table users
		id 			uint
		username	string

	3. table tenant_user_rels
		tenant_id	int
		user_id 	int
		role		string

	query:
	1. all members of tenant{id: 1, name: "egTenant"}
		use id case, only once join:
		select * from users
			left join tenant_user_rels on tenant_user_rels.user_id = users.id
			where tenant_user_rels.tenant_id = 1
		use other filed case, more than one times join:
		select * from users
			left join tenant_user_rels on tenant_user_rels.user_id = users.id
			left join tenants on tenants.id = tenant_user_rels.tenant_id
			where tenants.tenant_name = "egTenant"

*/

func ScopeBelongM2M(ownerModel, selfModel, viaModel interface{}, cond *Cond, refSelfField, refOwnerField string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		oname := tableName(tx, ownerModel)
		sname := tableName(tx, selfModel)
		vname := tableName(tx, viaModel)

		tdb := tx.
			Joins(fmt.Sprintf("LEFT JOIN %s on %s.id = %s.%s", vname, sname, vname, refSelfField)).
			Joins(fmt.Sprintf("LEFT JOIN %s on %s.id = %s.%s", oname, oname, vname, refOwnerField))

		return tdb.Where(fmt.Sprintf("%s.%s %s ?", oname, cond.Field, cond.Op), cond.Value)
	}
}

func ScopeOmitAssociations(tx *gorm.DB) *gorm.DB {
	return tx.Omit(clause.Associations)
}

func ScopeSearch(req *restful.Request, model interface{}, fields []string) func(tx *gorm.DB) *gorm.DB {
	search := req.QueryParameter("search")
	return func(tx *gorm.DB) *gorm.DB {
		if search == "" {
			return tx
		}
		tdb := tx
		tname := tableName(tx, model)
		qs := []string{}
		for idx := range fields {
			qs = append(qs, fmt.Sprintf("%s %s ?", tname+"."+fields[idx], Like))
		}
		args := make([]interface{}, len(fields))
		for idx := range args {
			args[idx] = "%" + search + "%"
		}
		tdb.Where("("+strings.Join(qs, " OR ")+")", args...)
		return tdb
	}
}

func tableName(db *gorm.DB, model interface{}) string {
	stmt := (&gorm.Statement{DB: db})
	stmt.Parse(model)
	return stmt.Schema.Table
}

func Where(field string, op ConditionOperator, value interface{}) *Cond {
	return &Cond{Field: field, Op: op, Value: value}
}

func WhereEqual(field string, value interface{}) *Cond {
	return Where(field, Eq, value)

}

func WhereNameEqual(value interface{}) *Cond {
	return Where("name", Eq, value)
}

func isEmpty(v interface{}) bool {
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}

func contains(list []string, str string) bool {
	for idx := range list {
		if list[idx] == str {
			return true
		}
	}
	return false
}
