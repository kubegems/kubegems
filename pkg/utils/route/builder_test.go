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

package route

import (
	"encoding/json"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/go-openapi/spec"
)

func JsonStr(v interface{}) string {
	data, _ := json.MarshalIndent(v, " ", " ")
	return string(data)
}

func TestDefinitionBuilder_Build(t *testing.T) {
	type SimpleStruct struct {
		Name  string      `json:"name,omitempty"`
		Value interface{} `json:"value,omitempty"`
	}
	type Bar struct {
		Kind string
	}

	type Baz struct {
		Age int64 `json:"age,omitempty"`
	}

	type Foo struct {
		Bar        `json:",inline"`
		Duration   time.Duration
		Time       time.Time   `json:"time,omitempty"`
		Number     json.Number `json:"number,omitempty"`
		Ignored    string      `json:"-,omitempty"`
		unExported string
	}

	type MultiEmbedded struct {
		Bar
		Baz
		string    // should be ignored
		io.Reader // embedded interface
	}

	type AllSample struct {
		List []AllSample
	}

	tests := []struct {
		name           string
		data           interface{}
		wantDeinations map[string]spec.Schema
		wantSchema     *spec.Schema
	}{
		{
			name:       "string value",
			data:       "",
			wantSchema: spec.StringProperty(),
		},
		{
			name:       "ineterface{} nil value",
			data:       interface{}(nil),
			wantSchema: nil,
		},
		{
			name:       "simple struct",
			data:       SimpleStruct{},
			wantSchema: spec.RefSchema(DefinitionsRoot + "route.SimpleStruct"),
			wantDeinations: map[string]spec.Schema{
				"route.SimpleStruct": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"name":  *spec.StringProperty(),
							"value": *ObjectProperty(),
						},
					},
				},
			},
		},
		{
			name:       "all mixed",
			data:       AllSample{},
			wantSchema: spec.RefSchema(DefinitionsRoot + "route.AllSample"),
			wantDeinations: map[string]spec.Schema{
				"route.AllSample": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"List": *spec.ArrayProperty(spec.RefSchema(DefinitionsRoot + "route.AllSample")),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{}
			gotSchema := b.Build(tt.data)
			if !reflect.DeepEqual(gotSchema, tt.wantSchema) {
				t.Errorf("DefinitionBuilder.Build() = %v, want %v", JsonStr(gotSchema), JsonStr(tt.wantSchema))
			}
			if !reflect.DeepEqual(b.Definitions, tt.wantDeinations) {
				t.Errorf("DefinitionBuilder.Definations = %v, want %v", JsonStr(b.Definitions), JsonStr(tt.wantDeinations))
			}
		})
	}
}

func TestDefinitionBuilder_buildMap(t *testing.T) {
	type Foo struct {
		Value string
	}

	type fields struct {
		Definitions map[string]spec.Schema
	}
	tests := []struct {
		name   string
		fields fields
		data   interface{}
		want   *spec.Schema
	}{

		{
			name: "empty interface{} map",
			data: map[string]interface{}{},
			want: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					AdditionalProperties: &spec.SchemaOrBool{
						Allows: true,
						Schema: ObjectProperty(),
					},
				},
			},
		},
		{
			name: "empty struct map",
			data: map[string]Foo{},
			want: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					AdditionalProperties: &spec.SchemaOrBool{
						Allows: true,
						Schema: spec.RefSchema(DefinitionsRoot + "route.Foo"),
					},
				},
			},
		},

		{
			name: "fixed keys struct map",
			data: map[string]Foo{
				"must": {},
			},
			want: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Type: []string{"object"},
					Properties: map[string]spec.Schema{
						"must": *spec.RefSchema(DefinitionsRoot + "route.Foo"),
					},
					AdditionalProperties: &spec.SchemaOrBool{
						Allows: true,
						Schema: spec.RefSchema(DefinitionsRoot + "route.Foo"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				Definitions: tt.fields.Definitions,
			}
			v := reflect.ValueOf(tt.data)
			if got := b.buildMap(v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefinitionBuilder.buildMap() = %v, want %v", JsonStr(got), JsonStr(tt.want))
			}
		})
	}
}

func TestDefinitionBuilder_buildInterface(t *testing.T) {
	type fields struct {
		Definitions map[string]spec.Schema
		Options     InterfaceBuildOption
	}
	tests := []struct {
		name   string
		fields fields
		v      reflect.Value
		want   *spec.Schema
	}{
		{
			name: "no sample value interface{}",
			v: func() reflect.Value {
				type InnerInterface struct {
					Data interface{}
				}
				return reflect.ValueOf(InnerInterface{}).FieldByName("Data")
			}(),
			want: ObjectProperty(),
		},
		{
			name:   "valued interface{}",
			fields: fields{Options: InterfaceBuildOptionMerge},
			v: func() reflect.Value {
				type InnerInterface struct {
					Data interface{}
				}
				return reflect.ValueOf(InnerInterface{Data: ""}).FieldByName("Data")
			}(),
			want: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					AnyOf: []spec.Schema{
						*ObjectProperty(),
						*spec.StringProperty(),
					},
				},
			},
		},
		{
			name:   "replaced valued interface{}",
			fields: fields{Options: InterfaceBuildOptionOverride},
			v: func() reflect.Value {
				type InnerInterface struct {
					Data interface{}
				}
				return reflect.ValueOf(InnerInterface{Data: ""}).FieldByName("Data")
			}(),
			want: spec.StringProperty(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				Definitions:          tt.fields.Definitions,
				InterfaceBuildOption: tt.fields.Options,
			}
			if got := b.buildInterface(tt.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefinitionBuilder.buildInterface() = %v, want %v", JsonStr(got), JsonStr(tt.want))
			}
		})
	}
}

func TestDefinitionBuilder_buildStruct(t *testing.T) {
	type Bar struct {
		Kind string `json:"kind,omitempty"`
	}

	type Value struct {
		Value string `json:"value,omitempty"`
	}

	type Ignored struct {
		Ignored string `json:"-"`
		ignored string // unExported
		Value   string `json:"value,omitempty"`
	}

	type Embedded struct {
		Bar
		Value Value `json:",inline"` // json inlined
	}

	type InterfacedStruct struct {
		Name string      `json:"name,omitempty"`
		Data interface{} `json:"data,omitempty"`
	}

	type RecursiveStruct struct {
		Data *RecursiveStruct
	}

	type fields struct {
		Definitions map[string]spec.Schema
	}

	tests := []struct {
		name           string
		fields         fields
		data           interface{}
		want           *spec.Schema
		wantDeinations map[string]spec.Schema
	}{
		{
			name: "simple struct",
			data: Bar{},
			want: spec.RefSchema(DefinitionsRoot + "route.Bar"),
			wantDeinations: map[string]spec.Schema{
				"route.Bar": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"kind": *spec.StringProperty(),
						},
					},
				},
			},
		},
		{
			name: "struct with json ignored fields",
			data: Ignored{},
			want: spec.RefSchema(DefinitionsRoot + "route.Ignored"),
			wantDeinations: map[string]spec.Schema{
				"route.Ignored": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"value": *spec.StringProperty(),
						},
					},
				},
			},
		},
		{
			name: "embedded struct only",
			data: Embedded{},
			want: spec.RefSchema(DefinitionsRoot + "route.Embedded"),
			wantDeinations: map[string]spec.Schema{
				"route.Bar": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"kind": *spec.StringProperty(),
						},
					},
				},
				"route.Value": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"value": *spec.StringProperty(),
						},
					},
				},
				"route.Embedded": {
					SchemaProps: spec.SchemaProps{
						AllOf: []spec.Schema{
							*spec.RefSchema(DefinitionsRoot + "route.Bar"),
							*spec.RefSchema(DefinitionsRoot + "route.Value"),
						},
					},
				},
			},
		},
		{
			name: "interfaced struct",
			data: InterfacedStruct{},
			want: spec.RefSchema(DefinitionsRoot + "route.InterfacedStruct"),
			wantDeinations: map[string]spec.Schema{
				"route.InterfacedStruct": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"name": *spec.StringProperty(),
							"data": *ObjectProperty(),
						},
					},
				},
			},
		},
		{
			name: "valued interface struct",
			data: InterfacedStruct{
				Data: Value{},
			},
			want: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					AllOf: []spec.Schema{
						*spec.RefSchema(DefinitionsRoot + "route.InterfacedStruct"),
						{
							SchemaProps: spec.SchemaProps{
								Type: []string{"object"},
								Properties: map[string]spec.Schema{
									"data": *spec.RefSchema(DefinitionsRoot + "route.Value"),
								},
							},
						},
					},
				},
			},
			wantDeinations: map[string]spec.Schema{
				"route.Value": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"value": *spec.StringProperty(),
						},
					},
				},
				"route.InterfacedStruct": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"name": *spec.StringProperty(),
							"data": *ObjectProperty(),
						},
					},
				},
			},
		},
		{
			name: "recursive struct",
			data: RecursiveStruct{},
			want: spec.RefSchema(DefinitionsRoot + "route.RecursiveStruct"),
			wantDeinations: map[string]spec.Schema{
				"route.RecursiveStruct": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"Data": *spec.RefSchema(DefinitionsRoot + "route.RecursiveStruct"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				Definitions:          tt.fields.Definitions,
				InterfaceBuildOption: InterfaceBuildOptionOverride,
			}
			v := reflect.ValueOf(tt.data)
			if got := b.buildStruct(v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefinitionBuilder.buildStruct() = %v, want %v", JsonStr(got), JsonStr(tt.want))
			}
			if !reflect.DeepEqual(b.Definitions, tt.wantDeinations) {
				t.Errorf("DefinitionBuilder.Definations = %v, want %v", JsonStr(b.Definitions), JsonStr(tt.wantDeinations))
			}
		})
	}
}

func TestBuilder_buildSlice(t *testing.T) {
	type SliceItem struct {
		Value string `json:"value,omitempty"`
	}

	type fields struct {
		InterfaceBuildOption InterfaceBuildOption
		Definitions          map[string]spec.Schema
	}
	type args struct {
		v reflect.Value
	}
	tests := []struct {
		name   string
		fields fields
		data   interface{}
		want   *spec.Schema
	}{
		{
			name: "string slice",
			data: []string{},
			want: spec.ArrayProperty(spec.StringProperty()),
		},
		{
			name: "simple slice",
			data: []SliceItem{},
			want: spec.ArrayProperty(
				spec.RefSchema(DefinitionsRoot + "route.SliceItem"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{
				InterfaceBuildOption: tt.fields.InterfaceBuildOption,
				Definitions:          tt.fields.Definitions,
			}
			v := reflect.ValueOf(tt.data)
			if got := b.buildSlice(v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Builder.buildSlice() = %v, want %v", JsonStr(got), JsonStr(tt.want))
			}
		})
	}
}
