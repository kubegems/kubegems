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
