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

package client

import (
	"fmt"
	"sort"
	"strings"
)

type RelationKind string

var (
	RelationM2M RelationKind = "m2m"
	RelationOwn RelationKind = "own"
)

type Relation struct {
	Key  string
	Kind RelationKind
	Via  ObjectTypeIface
}

type Cond struct {
	Field string
	Op    ConditionOperator
	Value interface{}
}

func (cond *Cond) AsQuery() (string, interface{}) {
	return fmt.Sprintf("%s %s ?", cond.Field, cond.Op), cond.Value
}

func RelationKey(obj1, obj2 ObjectTypeIface) string {
	keys := []string{*obj1.GetKind(), *obj2.GetKind()}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})
	return strings.Join(keys, ",")
}

func GetRelation(source, target, via ObjectTypeIface, relkind RelationKind) Relation {
	key := RelationKey(source, target)
	ret := Relation{
		Key:  key,
		Kind: relkind,
	}
	if relkind == RelationM2M && via != nil {
		ret.Via = via
	} else {
		ret.Via = nil
	}
	return ret
}

type RelationCondition struct {
	Key    string
	Value  interface{}
	Target Object
}

type Query struct {
	Page            int64
	Size            int64
	Search          string
	SearchFields    []string
	Orders          []string
	Preloads        []string
	Where           []*Cond
	Belong          []Object
	RelationOptions []RelationCondition
}
