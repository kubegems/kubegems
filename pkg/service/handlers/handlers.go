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
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"kubegems.io/kubegems/pkg/i18n"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/models/validate"
	"kubegems.io/library/rest/response"
)

var namer = schema.NamingStrategy{}

const (
	MessageOK           = "ok"
	MessageNotFound     = "not found"
	MessageForbidden    = "forbidden"
	MessageUnauthorized = "unauthorized"
)

func Page[T any](total int64, list []T, page, size int64) *response.Page[T] {
	return &response.Page[T]{Total: total, List: list, Page: page, Size: size}
}

func OK(c *gin.Context, data interface{}) {
	Response(c, http.StatusOK, data, nil)
}

func Created(c *gin.Context, data interface{}) {
	Response(c, http.StatusCreated, data, nil)
}

func NoContent(c *gin.Context, data interface{}) {
	Response(c, http.StatusNoContent, data, nil)
}

func BadRequest(c *gin.Context, err error) {
	Error(c, http.StatusBadRequest, err)
}

func Forbidden(c *gin.Context, err error) {
	Error(c, http.StatusForbidden, err)
}

func Unauthorized(c *gin.Context, err error) {
	Error(c, http.StatusUnauthorized, err)
}

func Error(c *gin.Context, code int, err error) {
	Response(c, code, nil, err)
}

func Response(c *gin.Context, code int, data interface{}, err error) {
	if err != nil {
		if code == 0 {
			code = http.StatusBadRequest
		}
		c.JSON(code, response.Response{Message: err.Error(), Error: err})
		return
	}
	if code == 0 {
		code = http.StatusOK
	}
	c.JSON(code, response.Response{Data: data})
}

func NotOK(c *gin.Context, err error) {
	log.Error(err, "not ok")
	defer func() {
		c.Errors = append(c.Errors, &gin.Error{Err: err, Type: gin.ErrorTypeAny})
	}()
	// validation error
	if errs, ok := err.(validator.ValidationErrors); ok {
		verrors := []string{}
		for _, e := range errs {
			// TODO: get trans from context
			verrors = append(verrors, e.Translate(validate.Get().Translator))
		}
		BadRequest(c, errors.New(strings.Join(verrors, ";")))
		return
	}
	// k8s apiserver error
	// always as badrequest,do not return 401,403
	if errors.Is(err, &apierrors.StatusError{}) {
		BadRequest(c, err)
		return
	}
	// grpc error
	if rpcerr, ok := status.FromError(err); ok {
		if rpcerr.Code() == codes.NotFound {
			Error(c, http.StatusNotFound, err)
			return
		}
	}
	// gorm error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		msg := i18n.Errorf(c.Request.Context(), "the object or the parent object is not found")
		Error(c, http.StatusNotFound, msg)
		return
	}

	// mysql error
	me := &mysql.MySQLError{}
	if errors.As(err, &me) {
		BadRequest(c, models.FormatMysqlError(me))
		return
	}

	// default error
	BadRequest(c, err)
}

func NewPageDataFromContext[T any](c *gin.Context, list []T, namefunc func(item T) string, timefunc func(item T) time.Time) response.Page[T] {
	return response.PageFromRequest(c.Request, list, namefunc, timefunc)
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
		return i18n.Errorf(context.TODO(), "invalid page number query parameter")
	}
	q.size, err = strconv.ParseInt(q.Size, 10, 64)
	if err != nil {
		return i18n.Errorf(context.TODO(), "invalid page size query parameter")
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

// 网络隔离用到的数据结构
type ClusterIsolatedSwitch struct {
	Isolate   bool `json:"isolate"`
	ClusterID uint `json:"cluster_id" binding:"required"`
}

type IsolatedSwitch struct {
	Isolate bool `json:"isolate"`
}
