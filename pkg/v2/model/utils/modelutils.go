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

package utils

import (
	"fmt"

	"github.com/VividCortex/mysqlerr"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/v2/model/client"
)

func GetQuery(opts ...client.Option) *client.Query {
	q := &client.Query{}
	for _, opt := range opts {
		opt.Apply(q)
	}
	return q
}

func Contains(arr []string, t string) bool {
	for _, ar := range arr {
		if ar == t {
			return true
		}
	}
	return false
}

func GetErrMessage(err error) string {
	me, ok := err.(*driver.MySQLError)
	if !ok {
		return fmt.Sprintf("%v", err)
	}
	switch me.Number {
	case mysqlerr.ER_DUP_ENTRY:
		return fmt.Sprintf("存在重名对象(code=%v)", me.Number)
	case mysqlerr.ER_DATA_TOO_LONG:
		return fmt.Sprintf("数据超长(code=%v)", me.Number)
	case mysqlerr.ER_TRUNCATED_WRONG_VALUE:
		return fmt.Sprintf("日期格式错误(code=%v)", me.Number)
	case mysqlerr.ER_NO_REFERENCED_ROW_2:
		return fmt.Sprintf("系统错误(外键关联数据出错 code=%v)", me.Number)
	case mysqlerr.ER_ROW_IS_REFERENCED_2:
		return fmt.Sprintf("系统错误(外键关联数据错误 code=%v)", me.Number)
	default:
		return fmt.Sprintf("系统错误(code=%v, message=%v)!", me.Number, me.Message)
	}
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
