package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"kubegems.io/kubegems/pkg/v2/model/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"kubegems.io/kubegems/pkg/v2/model/client"
)

type Client struct {
	db        *gorm.DB
	relations map[string]*client.Relation
	hooks     map[string]func(*gorm.DB, client.Object) error
}

func Init(opt *MySQLOptions) (*Client, error) {
	db, err := NewDatabaseInstance(opt)
	if err != nil {
		return nil, err
	}
	return NewOrmClient(db), nil
}

func NewOrmClient(db *gorm.DB) *Client {
	c := &Client{
		db:        db,
		relations: make(map[string]*client.Relation),
		hooks:     make(map[string]func(*gorm.DB, client.Object) error),
	}
	c.RegistHooks()
	return c
}

func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.Option) error {
	tx := c.db.WithContext(ctx).Begin()
	tdb := tx.Scopes(
		scopeTable(tableName(obj)),
		scopeOmitAssociations,
	)
	if err := c.executeHook(obj, client.BeforeCreate); err != nil {
		tx.Rollback()
		return err
	}
	if err := tdb.Create(obj).Error; err != nil {
		return err
	}
	if err := c.executeHook(obj, client.AfterCreate); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.Option) error {
	tx := c.db.WithContext(ctx).Begin()
	tdb := tx.Scopes(
		scopeOmitAssociations,
		scopeTable(tableName(obj)),
	)
	if err := c.executeHook(obj, client.BeforeUpdate); err != nil {
		tx.Rollback()
		return err
	}
	q := utils.GetQuery(opts...)
	tdb = tdb.Scopes(
		scopeCond(q.Where, tableName(obj)),
	)
	if err := tdb.Updates(obj).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := c.executeHook(obj, client.AfterUpdate); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.Option) error {
	tx := c.db.WithContext(ctx).Begin()
	tName := tableName(obj)
	q := utils.GetQuery(opts...)
	tdb := tx.Scopes(
		scopeTable(tName),
		scopeCond(q.Where, tName),
	)
	if err := c.executeHook(obj, client.BeforeDelete); err != nil {
		tx.Rollback()
		return err
	}
	if err := tdb.Delete(obj).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := c.executeHook(obj, client.AfterDelete); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (c *Client) CreateInBatches(ctx context.Context, obj client.ObjectListIface, opts ...client.Option) error {
	// NOTICE: no hook here, if need, just add it
	tdb := c.db.WithContext(ctx)
	tdb = tdb.Scopes(
		scopeTable(tableName(obj)),
		scopeOmitAssociations,
	)
	return tdb.CreateInBatches(obj.DataPtr(), 1000).Error
}

func (c *Client) Get(ctx context.Context, obj client.Object, opts ...client.Option) error {
	q := utils.GetQuery(opts...)
	tdb := c.db.WithContext(ctx)
	tdb = tdb.Scopes(
		scopeTable(tableName(obj)),
		scopePreload(q.Preloads, *obj.PreloadFields()),
		scopeCond(q.Where, tableName(obj)),
	)
	return tdb.First(obj).Error
}

func (c *Client) List(ctx context.Context, olist client.ObjectListIface, opts ...client.Option) error {
	tdb := c.db.WithContext(ctx)
	q := utils.GetQuery(opts...)

	tdb = tdb.Scopes(
		scopeTable(tableName(olist)),
		scopeBelong(q.Belong, tableName(olist)),
		scopeOrder(q.Orders),
		scopeRelation(q.RelationOptions, olist),
		scopePreload(q.Preloads, nil),
		scopeCond(q.Where, tableName(olist)),
		scopePageSize(q.Page, q.Size, olist),
	)

	var total int64
	c.Count(ctx, olist, &total, opts...)
	olist.SetTotal(total)

	return tdb.Find(olist.DataPtr()).Error
}

func (c *Client) Count(ctx context.Context, o client.ObjectTypeIface, t *int64, opts ...client.Option) error {
	q := utils.GetQuery(opts...)
	tdb := c.db.WithContext(ctx)
	tName := tableName(o)
	tdb = tdb.Scopes(
		scopeTable(tName),
		scopeCond(q.Where, tName),
	)
	return tdb.Count(t).Error
}

func scopeCond(conds []*client.Cond, mytable string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if len(conds) == 0 {
			return tx
		}
		qs := []string{}
		vs := []interface{}{}
		for idx := range conds {
			q, v := conds[idx].AsQuery()
			qs = append(qs, mytable+"."+q)
			vs = append(vs, v)
		}
		return tx.Where(strings.Join(qs, " AND "), vs...)

	}
}

func scopePreload(preloads, validPreloads []string) func(tx *gorm.DB) *gorm.DB {
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

func scopeOrder(orders []string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tdb := tx
		for _, orderStr := range orders {
			tdb = tdb.Order(orderStr)
		}
		return tdb
	}
}

func scopePageSize(page, size int64, ol client.ObjectListIface) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tdb := tx
		if size > 0 && page > 0 {
			offset := size * (page - 1)
			ol.SetPageSize(page, size)
			tdb.Offset(int(offset)).Limit(int(size))
		}
		return tdb
	}
}

func scopeBelong(belongs []client.Object, mytable string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if len(belongs) == 0 {
			return tx
		}
		tdb := tx
		for _, obj := range belongs {
			fieldName := fmt.Sprintf("%s_%s", *obj.GetKind(), *obj.PrimaryKeyField())
			if *obj.PrimaryKeyField() == "id" {
				q := fmt.Sprintf("%s = ?", fieldName)
				tdb = tx.Where(q, obj.PrimaryKeyValue())
			} else {
				rightTable := tableName(obj)
				joinQ := fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.%s", rightTable, rightTable, mytable, fieldName)
				tdb = tdb.Joins(joinQ)
				q := fmt.Sprintf("%s = ?", rightTable+"."+fieldName)
				tdb = tdb.Where(q, obj.PrimaryKeyValue())
			}
		}
		return tdb
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
func scopeRelation(rels []client.RelationCondition, ol client.ObjectTypeIface) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		tdb := tx
		if len(rels) == 0 {
			return tx
		}
		selfTable := tableName(ol)
		for _, rel := range rels {
			relTable := relTableName(rel.Target, ol)
			rightKey := *ol.GetKind() + "_id"
			joinQ := fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.id", relTable, relTable, rightKey, selfTable)
			tdb = tdb.Joins(joinQ)
			if rel.Key != "" {
				q := fmt.Sprintf("%s.%s = ?", relTable, rel.Key)
				tdb = tdb.Where(q, rel.Value)
			}
			if !isEmpty(rel.Target.PrimaryKeyValue()) {
				q := fmt.Sprintf("%s.%s", relTable, *rel.Target.GetKind()+"_id")
				tdb = tdb.Where(q, rel.Target.PrimaryKeyValue())
			}
		}
		return tdb
	}
}

func scopeTable(tablename string) func(tx *gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Table(tablename)
	}
}

func scopeOmitAssociations(tx *gorm.DB) *gorm.DB {
	return tx.Omit(clause.Associations)
}

func isEmpty(v interface{}) bool {
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}
