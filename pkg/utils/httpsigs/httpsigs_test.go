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

package httpsigs

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestSigner_Validate(t *testing.T) {
	type fields struct {
		Token     string
		Duration  int64
		WhiteList []string
	}
	type args struct {
		req *http.Request
	}
	type testS struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	tests := []testS{}

	timeStr := strconv.FormatInt(time.Now().Unix(), 10)
	token := "123456"

	{
		name := "normal"
		path1 := "/test1"
		toSignStr1 := path1 + timeStr + token
		token1 := fmt.Sprintf("%x", md5.Sum([]byte(toSignStr1)))
		header1 := http.Header{}
		header1.Add(headerTime, timeStr)
		header1.Add(headerToken, token1)
		test1 := testS{
			name: name,
			fields: fields{
				Token:    "123456",
				Duration: 30,
			},
			args: args{
				&http.Request{
					Header: header1,
					URL: &url.URL{
						Path: path1,
					},
				},
			},
			wantErr: false,
		}
		tests = append(tests, test1)
	}
	{
		name := "error path"
		path2 := "/test2"
		toSignStr2 := "/" + timeStr + token
		token2 := fmt.Sprintf("%x", md5.Sum([]byte(toSignStr2)))
		header2 := http.Header{}
		header2.Add(headerTime, timeStr)
		header2.Add(headerToken, token2)
		test2 := testS{
			name: name,
			fields: fields{
				Token:    "123456",
				Duration: 10,
			},
			args: args{
				&http.Request{
					Header: header2,
					URL: &url.URL{
						Path: path2,
					},
				},
			},
			wantErr: true,
		}
		tests = append(tests, test2)
	}
	{
		name := "error time"
		path := "/test"
		toSignStr := "/" + timeStr + token
		token2 := fmt.Sprintf("%x", md5.Sum([]byte(toSignStr)))
		header2 := http.Header{}
		header2.Add(headerTime, timeStr+"x")
		header2.Add(headerToken, token2)
		test2 := testS{
			name: name,
			fields: fields{
				Token:    "123456",
				Duration: 10,
			},
			args: args{
				&http.Request{
					Header: header2,
					URL: &url.URL{
						Path: path,
					},
				},
			},
			wantErr: true,
		}
		tests = append(tests, test2)
	}
	{
		name := "timeout"
		path2 := "/test2"

		timeStr3 := strconv.FormatInt(time.Now().Add(time.Duration(-5)*time.Second).Unix(), 10)
		fmt.Println(timeStr3)
		toSignStr2 := path2 + timeStr3 + token
		token2 := fmt.Sprintf("%x", md5.Sum([]byte(toSignStr2)))
		header2 := http.Header{}
		header2.Add(headerTime, timeStr3)
		header2.Add(headerToken, token2)
		test2 := testS{
			name: name,
			fields: fields{
				Token:    "123456",
				Duration: 1,
			},
			args: args{
				&http.Request{
					Header: header2,
					URL: &url.URL{
						Path: path2,
					},
				},
			},
			wantErr: true,
		}
		tests = append(tests, test2)
	}
	{
		name := "whitelist"
		path4 := "/whitelist"
		toSignStr2 := path4 + timeStr + token
		token2 := fmt.Sprintf("%x", md5.Sum([]byte(toSignStr2)))
		header2 := http.Header{}
		header2.Add(headerTime, timeStr)
		header2.Add(headerToken, token2)
		test2 := testS{
			name: name,
			fields: fields{
				Token:    "123456",
				Duration: 1,
			},
			args: args{
				&http.Request{
					Header: header2,
					URL: &url.URL{
						Path: path4,
					},
				},
			},
			wantErr: false,
		}
		tests = append(tests, test2)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Signer{
				Token:     tt.fields.Token,
				Duration:  tt.fields.Duration,
				WhiteList: tt.fields.WhiteList,
			}
			if tt.name == "whitelist" {
				s.AddWhiteList("/whitelist")
				s.AddWhiteList("/whitelist")
			}
			if err := s.Validate(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("Signer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSigner_Sign(t *testing.T) {
	type fields struct {
		Token     string
		Duration  int64
		WhiteList []string
	}
	type args struct {
		req    *http.Request
		prefix string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "normal",
			fields: fields{
				Token:    "123456",
				Duration: 2,
			},
			args: args{
				req: &http.Request{
					URL: &url.URL{
						Path: "/test",
					},
					Header: http.Header{},
				},
				prefix: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Signer{
				Token:     tt.fields.Token,
				Duration:  tt.fields.Duration,
				WhiteList: tt.fields.WhiteList,
			}
			s.Sign(tt.args.req, tt.args.prefix)

			tt.args.req.Header.Get(headerToken)
			if len(tt.args.req.Header.Get(headerTime)) == 0 || len(tt.args.req.Header.Get(headerToken)) == 0 {
				t.Errorf("Signer.Sing() header token empty")
			}
		})
	}
}
