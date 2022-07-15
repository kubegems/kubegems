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
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"k8s.io/apimachinery/pkg/api/errors"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/models/validate"
)

var namer = schema.NamingStrategy{}

const (
	defaultPageSize = 10
)

const (
	MessageOK           = "ok"
	MessageNotFound     = "not found"
	MessageError        = "err"
	MessageForbidden    = "forbidden"
	MessageUnauthorized = "unauthorized"

	MethodList = "LIST"
)

type ResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}

type PageData struct {
	Total       int64
	List        interface{}
	CurrentPage int64
	CurrentSize int64
}

func Page(total int64, list interface{}, page, size int64) *PageData {
	return &PageData{
		Total:       total,
		List:        list,
		CurrentPage: page,
		CurrentSize: size,
	}
}

func Response(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(code, ResponseStruct{Message: msg, Data: data})
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ResponseStruct{Message: MessageOK, Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, ResponseStruct{Message: MessageOK, Data: data})
}

func NoContent(c *gin.Context, data interface{}) {
	c.JSON(http.StatusNoContent, ResponseStruct{Message: MessageOK, Data: data})
}

func Forbidden(c *gin.Context, data interface{}) {
	c.JSON(http.StatusForbidden, ResponseStruct{Message: MessageForbidden, Data: data})
}

func Unauthorized(c *gin.Context, data interface{}) {
	c.JSON(http.StatusUnauthorized, ResponseStruct{Message: MessageUnauthorized, Data: data})
}

func errResponse(errData interface{}) ResponseStruct {
	return ResponseStruct{Message: MessageError, ErrorData: errData}
}

func NotOK(c *gin.Context, err error) {
	defer func() {
		c.Errors = append(c.Errors, &gin.Error{
			Err:  err,
			Type: gin.ErrorTypeAny,
		})
	}()
	if errs, ok := err.(validator.ValidationErrors); ok {
		verrors := []string{}
		for _, e := range errs {
			verrors = append(verrors, e.Translate(validate.Get().Translator))
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, errResponse(strings.Join(verrors, ";")))
		return
	}
	if err, ok := err.(*errors.StatusError); ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, errResponse(err.Error()))
		return
	}

	// grpc error
	if rpcerr, ok := status.FromError(err); ok {
		if rpcerr.Code() == codes.NotFound {
			c.AbortWithStatusJSON(http.StatusNotFound, errResponse(err.Error()))
			return
		}
	}

	if models.IsNotFound(err) {
		msg := "没有找到该对象 或者 该对象的上级关联对象(404)"
		c.AbortWithStatusJSON(http.StatusNotFound, errResponse(msg))
		return
	}
	msg := models.GetErrMessage(err)
	c.AbortWithStatusJSON(http.StatusBadRequest, errResponse(msg))
}

func NewPageDataFromContext(c *gin.Context, fulllist interface{}, pick PageFilterFunc, sortfn PageSortFunc) PageData {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("size"))
	if pick == nil {
		pick = NoopPageFilterFunc
	}
	return NewPageData(fulllist, page, size, pick, sortfn)
}

type (
	PageFilterFunc func(i int) bool
	PageSortFunc   func(i, j int) bool
)

var (
	NoopPageFilterFunc = func(i int) bool { return true }
	NoopPageSortFunc   = func(i, j int) bool { return false }
)

func NewPageData(list interface{}, page, size int, filterfn PageFilterFunc, sortfn PageSortFunc) PageData {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}
	// sort
	if sortfn != nil {
		sort.Slice(list, sortfn)
	}

	v := reflect.ValueOf(list)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return PageData{}
	}

	// filter
	if filterfn != nil {
		ret := reflect.MakeSlice(v.Type(), 0, 10)
		for i := 0; i < v.Len(); i++ {
			if filterfn(i) {
				ret = reflect.Append(ret, v.Index(i))
			}
		}
		v = ret
	}

	// page
	total := v.Len()
	start := (page - 1) * size
	end := page * size
	if end > total {
		end = total
	}
	v = v.Slice(start, end)

	return PageData{
		List:        v.Interface(),
		Total:       int64(total),
		CurrentPage: int64(page),
		CurrentSize: int64(size),
	}
}

func contains(arr []string, s string) bool {
	for _, ar := range arr {
		if ar == s {
			return true
		}
	}
	return false
}

func tableName(model string) string {
	return namer.TableName(model)
}

func columnName(model, field string) string {
	return namer.ColumnName(tableName(model), field)
}

type URLQuery struct {
	Page    string `form:"page"`
	Size    string `form:"size"`
	Order   string `form:"order"`
	Search  string `form:"search"`
	Preload string `form:"preload"`

	sortFunc func(interface{}, string)

	preloads []string
	page     int64
	size     int64
	startPos int64
	endPos   int64
}

func NewURLQuery(sortfn func(interface{}, string)) *URLQuery {
	return &URLQuery{
		sortFunc: sortfn,
		page:     1,
		size:     10,
		endPos:   10,
		Page:     "1",
		Size:     "10",
		preloads: []string{},
	}
}

func GetQuery(c *gin.Context, sortFunc func(interface{}, string)) (*URLQuery, error) {
	q := NewURLQuery(sortFunc)
	if err := c.BindQuery(&q); err != nil {
		return nil, err
	}
	if err := q.convert(); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *URLQuery) convert() error {
	var err error
	q.page, err = strconv.ParseInt(q.Page, 10, 64)
	if err != nil {
		return fmt.Errorf("页错误")
	}
	q.size, err = strconv.ParseInt(q.Size, 10, 64)
	if err != nil {
		return fmt.Errorf("页size错误")
	}
	if q.page <= 0 {
		q.page = 1
	}
	if q.size <= 0 {
		q.size = 10
	}
	q.startPos = (q.page - 1) * q.size
	q.endPos = q.startPos + q.size
	preSeps := strings.Split(q.Preload, ",")
	var preloads []string
	for idx := range preSeps {
		if len(preSeps[idx]) > 0 {
			preloads = append(preloads, preSeps[idx])
		}
	}
	if len(preloads) > 0 {
		q.preloads = preloads
	}
	return nil
}

const (
	orderDESC = "DESC"
	orderASC  = "ASC"
)

type QArgs struct {
	Query interface{}
	Args  []interface{}
}

func Args(q interface{}, args ...interface{}) *QArgs {
	return &QArgs{
		Query: q,
		Args:  args,
	}
}

type PageQueryCond struct {
	Model                  string
	SearchFields           []string
	SortFields             []string
	PreloadFields          []string
	PreloadSensitiveFields map[string]string
	Select                 *QArgs
	Join                   *QArgs
	Where                  []*QArgs
}

func (q *URLQuery) PageQuery(db *gorm.DB, cond *PageQueryCond) *gorm.DB {
	tmpdb := db.Offset(int(q.startPos)).Limit(int(q.size))
	if len(q.Search) > 0 {
		qs := []string{}
		for _, field := range cond.SearchFields {
			qs = append(qs, fmt.Sprintf("%s like ?", columnName(cond.Model, field)))
		}
		if len(qs) > 0 {
			tmpq := strings.Join(qs, " or ")
			tmpqs := make([]interface{}, len(qs))
			for idx := range tmpqs {
				tmpqs[idx] = fmt.Sprintf("%%%s%%", q.Search)
			}
			tmpdb = tmpdb.Where(tmpq, tmpqs...)
		}
	}
	if len(q.Order) > 0 {
		for _, field := range cond.SearchFields {
			if strings.EqualFold(q.Order, field+orderDESC) {
				columnName := columnName(cond.Model, field)
				tmpdb = tmpdb.Order(fmt.Sprintf("%s DESC", columnName))
				break
			}
			if strings.EqualFold(q.Order, field) || strings.EqualFold(q.Order, field+orderASC) {
				columnName := columnName(cond.Model, field)
				tmpdb = tmpdb.Order(fmt.Sprintf("%s ASC", columnName))
				break
			}
		}
	}
	for _, preload := range q.preloads {
		if contains(cond.PreloadFields, preload) {
			selectFields, exist := cond.PreloadSensitiveFields[preload]
			if !exist {
				tmpdb = tmpdb.Preload(preload)
			} else {
				tmpdb = tmpdb.Preload(preload, func(tx *gorm.DB) *gorm.DB { return tx.Select(selectFields) })
			}
		}
	}

	if cond.Join != nil {
		tmpdb = tmpdb.Joins(cond.Join.Query.(string), cond.Join.Args...)
	}

	if cond.Select != nil {
		tmpdb = tmpdb.Select(cond.Select.Query.(string), cond.Select.Args...)
	}

	for _, where := range cond.Where {
		tmpdb = tmpdb.Where(where.Query, where.Args...)
	}
	return tmpdb
}

func (q *URLQuery) Count(db *gorm.DB, cond *PageQueryCond) (total int64, err error) {
	table := tableName(cond.Model)
	countdb := db.Table(table)
	if len(q.Search) > 0 {
		qs := []string{}
		for _, field := range cond.SearchFields {
			qs = append(qs, fmt.Sprintf("%s like ?", columnName(cond.Model, field)))
		}
		if len(qs) > 0 {
			tmpq := strings.Join(qs, " or ")
			tmpqs := make([]interface{}, len(qs))
			for idx := range tmpqs {
				tmpqs[idx] = fmt.Sprintf("%%%s%%", q.Search)
			}
			countdb = countdb.Where(tmpq, tmpqs...)
		}
	}
	if cond.Join != nil {
		countdb = countdb.Joins(cond.Join.Query.(string), cond.Join.Args...)
	}
	if len(cond.Where) > 0 {
		for _, where := range cond.Where {
			countdb = countdb.Where(where.Query, where.Args...)
		}
	}
	if err = countdb.Count(&total).Error; err != nil {
		return
	}
	return
}

func (q *URLQuery) PageList(db *gorm.DB, cond *PageQueryCond, dest interface{}) (total int64, page, size int64, err error) {
	originClause := make(map[string]clause.Clause)
	for k, v := range db.Statement.Clauses {
		originClause[k] = v
	}
	total, err = q.Count(db, cond)
	if err != nil {
		return
	}

	db.Statement.Clauses = originClause
	querydb := q.PageQuery(db, cond)
	if err = querydb.Find(dest).Error; err != nil {
		return
	}
	page = q.page
	size = q.size
	return
}

func (q *URLQuery) MustPreload(mustpreloads []string) *URLQuery {
	q.preloads = append(q.preloads, mustpreloads...)
	return q
}

func (q *URLQuery) PageResponse(data interface{}) interface{} {
	if q.sortFunc != nil {
		q.sortFunc(data, q.Order)
	}
	kdata := reflect.TypeOf(data)
	if kdata.Kind() != reflect.Array && kdata.Kind() != reflect.Slice {
		return ResponseStruct{Message: MessageError, ErrorData: fmt.Sprintf("error data to paginate, kind is %v", kdata.Kind())}
	}
	vdata := reflect.ValueOf(data)
	total := int64(vdata.Len())
	if q.endPos >= total {
		q.endPos = total
	}
	if q.startPos >= q.endPos {
		return ResponseStruct{
			Message: MessageOK,
			Data:    Page(int64(total), []interface{}{}, int64(q.page), int64(q.size)),
		}
	}
	return ResponseStruct{
		Message: MessageOK,
		Data:    Page(int64(total), vdata.Slice(int(q.startPos), int(q.endPos)).Interface(), int64(q.page), int64(q.size)),
	}
}

// 网络隔离用到的数据结构
type ClusterIsolatedSwitch struct {
	Isolate   bool `json:"isolate"`
	ClusterID uint `json:"cluster_id" binding:"required"`
}

type IsolatedSwitch struct {
	Isolate bool `json:"isolate"`
}
