package pagination

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PageData struct {
	Total       int64
	List        interface{}
	CurrentPage int64
	CurrentSize int64
}

type listQuery struct {
	Page   int    `form:"page"`
	Size   int    `form:"size"`
	Search string `form:"search"`
	Sort   string `form:"sort"`
}

const defaultPageSize = 10

type SortAndSearchAble interface {
	GetName() string
	GetCreationTimestamp() metav1.Time
}

type noSortAndSearchAble struct {
	Data interface{}
}

func (no noSortAndSearchAble) MarshalJSON() ([]byte, error) {
	return json.Marshal(no.Data)
}

func (noSortAndSearchAble) GetName() string {
	return ""
}

func (noSortAndSearchAble) GetCreationTimestamp() metav1.Time {
	return metav1.Time{}
}

// NewPageDataFromContextReflect data 必须为一个实现  SortAndSearchAble 接口的list，其内部会自动进行 搜索 排序 分页
func NewPageDataFromContextReflect(c *gin.Context, list interface{}) PageData {
	v := reflect.ValueOf(list)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return PageData{}
	}
	return NewPageDataFromContext(c, func(i int) SortAndSearchAble {
		if ret, ok := v.Index(i).Interface().(SortAndSearchAble); ok {
			return ret
		}
		return noSortAndSearchAble{Data: v.Index(i).Interface()}
	}, v.Len(), list)
}

// NewPageDataFromContext 从context
// 读取 search 根据 search 对 resource.metadata.name 进行过滤
// 读取 sort 按照 resource.metadata. 中对应字段进行排序
// 读取 page size 对上述结果进行分页
func NewPageDataFromContext(c *gin.Context, metaAccessor func(i int) SortAndSearchAble, length int, _ interface{}) PageData {
	var q listQuery
	_ = c.BindQuery(&q)
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 {
		q.Size = defaultPageSize
	}

	tmpdatas := []SortAndSearchAble{}
	for tmp := 0; tmp < length; tmp++ {
		item := metaAccessor(tmp)
		if item == nil {
			continue
		}
		if (q.Search == "") || (q.Search != "" && strings.Contains(item.GetName(), q.Search)) {
			tmpdatas = append(tmpdatas, item)
		}
	}
	total := len(tmpdatas)
	SortByFunc(tmpdatas, q.Sort)

	startIdx := (q.Page - 1) * q.Size
	endIdx := startIdx + q.Size
	if startIdx > total {
		startIdx = 0
		endIdx = 0
	}
	if endIdx > total {
		endIdx = total
	}
	data := tmpdatas[startIdx:endIdx]

	return PageData{
		List:        data,
		Total:       int64(total),
		CurrentPage: int64(q.Page),
		CurrentSize: int64(q.Size),
	}
}

func SortByFunc(datas []SortAndSearchAble, by string) {
	switch by {
	case "createTimeAsc":
		sort.Slice(datas, func(i, j int) bool {
			return datas[i].GetCreationTimestamp().UnixNano() < datas[j].GetCreationTimestamp().UnixNano()
		})
	case "nameAsc", "name":
		sort.Slice(datas, func(i, j int) bool {
			return strings.Compare((datas[i].GetName()), (datas[j].GetName())) == -1
		})
	case "nameDesc":
		sort.Slice(datas, func(i, j int) bool {
			return strings.Compare((datas[i].GetName()), (datas[j].GetName())) == 1
		})
	case "createTimeDesc", "createTime", "time":
		sort.Slice(datas, func(i, j int) bool {
			return datas[i].GetCreationTimestamp().UnixNano() > datas[j].GetCreationTimestamp().UnixNano()
		})
	default:
		sort.Slice(datas, func(i, j int) bool {
			return datas[i].GetCreationTimestamp().UnixNano() > datas[j].GetCreationTimestamp().UnixNano()
		})
	}
}
