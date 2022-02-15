package orm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"kubegems.io/pkg/model/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"kubegems.io/pkg/model/client"
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
	c.RegistRelation(&User{}, &TenantList{}, &TenantUserRel{}, client.RelationM2M)
	c.RegistHooks()
	return c
}

func (c *Client) RegistRelation(source, target, via client.ObjectTypeIfe, kind client.RelationKind) {
	rel := client.GetRelation(source, target, via, kind)
	c.relations[rel.Key] = &rel
}

func (c *Client) getRelation(s, t client.ObjectTypeIfe) *client.Relation {
	key := client.RelationKey(s, t)
	return c.relations[key]
}

func (c *Client) Exist(obj client.Object, opts ...client.Option) bool {
	var total int64
	if err := c.db.Table(tableName(obj)).Where(obj).Count(&total).Error; err != nil {
		// TODO： log
		return false
	}
	return total > 0
}

func (c *Client) Get(obj client.Object, opts ...client.Option) error {
	tdb := c.db.Table(tableName(obj))
	q := utils.GetQuery(opts...)
	validPreloads := obj.ValidPreloads()
	for _, preloadField := range q.Preloads {
		if contains(*validPreloads, preloadField) {
			tdb = tdb.Preload(preloadField)
		}
	}
	for _, where := range q.Where {
		tdb.Where(where.AsQuery())
	}
	return tdb.First(obj).Error
}

func (c *Client) Delete(obj client.Object, opts ...client.Option) error {
	tx := c.db.Begin()
	tdb := tx.Table(tableName(obj))
	q := utils.GetQuery(opts...)
	for _, where := range q.Where {
		tdb.Where(where.AsQuery())
	}
	if err := c.executeHook(obj, client.BeforeDelete, tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := tdb.Delete(obj).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := c.executeHook(obj, client.AfterDelete, tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (c *Client) Update(obj client.Object, opts ...client.Option) error {
	tdb := c.db.Table(tableName(obj)).Omit(clause.Associations)
	q := utils.GetQuery(opts...)
	for _, where := range q.Where {
		tdb.Where(where.AsQuery())
	}
	return tdb.Updates(obj).Error
}

func (c *Client) Create(obj client.Object, opts ...client.Option) error {
	tdb := c.db.Table(tableName(obj)).Omit(clause.Associations)
	// q := GetQuery(opts...)
	return tdb.Create(obj).Error
}

func (c *Client) CreateInBatches(obj client.ObjectListIfe, opts ...client.Option) error {
	tdb := c.db.Table(tableName(obj)).Omit(clause.Associations)
	return tdb.CreateInBatches(obj.DataPtr(), 1000).Error
}

func (c *Client) List(olist client.ObjectListIfe, opts ...client.Option) error {
	tdb := c.db.Table(tableName(olist))
	q := utils.GetQuery(opts...)
	for _, preload := range q.Preloads {
		tdb = tdb.Preload(preload)
	}
	for _, where := range q.Where {
		tdb = tdb.Where(where.AsQuery())
	}
	for _, orderStr := range q.Orders {
		tdb = tdb.Order(orderStr)
	}
	if q.Size > 0 && q.Page > 0 {
		offset := q.Size * (q.Page - 1)
		olist.SetPageSize(q.Page, q.Size)
		tdb.Offset(int(offset)).Limit(int(q.Size))
		var total int64
		if e := c.Count(olist, q, &total); e != nil {
			return e
		}
		olist.SetTotal(total)
	}
	return tdb.Find(olist.DataPtr()).Error
}

func (c *Client) Count(o client.ObjectTypeIfe, q *client.Query, t *int64) error {
	tdb := c.db.Table(tableName(o))
	for _, where := range q.Where {
		tdb.Where(where.AsQuery())
	}
	return tdb.Count(t).Error
}

func (c *Client) CountSubResource(obj client.Object, olist client.ObjectListIfe, relation client.Relation, q *client.Query, t *int64) error {
	tdb := c.db.Table(tableName(olist))
	if relation.Kind == client.RelationM2M {
		joinTable := tableName(relation.Via)
		joinstr := fmt.Sprintf("JOIN %s on %s.%s_%s = %s.%s", joinTable, joinTable, *olist.GetKind(), *olist.GetPKField(), tableName(olist), *olist.GetPKField())
		tdb = tdb.Joins(joinstr)
	}
	for _, where := range q.Where {
		tdb.Where(where.AsQuery())
	}
	return tdb.Count(t).Error
}

func (c *Client) ListSubResource(obj client.Object, olist client.ObjectListIfe, opts ...client.Option) error {
	tdb := c.db.Table(tableName(olist))
	relation := c.getRelation(obj, olist)
	if relation == nil {
		return fmt.Errorf("failed to load relation between %s and %s", *obj.GetKind(), *olist.GetKind())
	}
	if relation.Kind == client.RelationM2M {
		joinTable := tableName(relation.Via)
		joinstr := fmt.Sprintf("LEFT JOIN %s on %s.%s_%s = %s.%s", joinTable, joinTable, *olist.GetKind(), *olist.GetPKField(), tableName(olist), *olist.GetPKField())
		tdb = tdb.Joins(joinstr)
		opts = append(opts, client.Where(fmt.Sprintf("%s.%s_%s", joinTable, *obj.GetKind(), *obj.GetPKField()), client.Eq, obj.GetPKValue()))
	} else {
		opts = append(opts, client.Where(objIDFiled(obj), client.Eq, obj.GetPKValue()))
	}
	q := utils.GetQuery(opts...)
	if relation.Kind == client.RelationM2M && len(q.RelationFields) > 0 {
		joinTable := tableName(relation.Via)
		rfields := []string{}
		for _, field := range q.RelationFields {
			rfields = append(rfields, fmt.Sprintf("%s.%s", joinTable, field))
		}
		tdb = tdb.Select(fmt.Sprintf("%s.*, %s", tableName(olist), strings.Join(rfields, ",")))
	}

	for _, preload := range q.Preloads {
		tdb = tdb.Preload(preload)
	}
	for _, where := range q.Where {
		tdb.Where(where.AsQuery())
	}
	for _, orderStr := range q.Orders {
		tdb = tdb.Order(orderStr)
	}
	if q.Size > 0 && q.Page > 0 {
		offset := q.Size * (q.Page - 1)
		olist.SetPageSize(q.Page, q.Size)
		tdb.Offset(int(offset)).Limit(int(q.Size))
		var total int64
		if e := c.CountSubResource(obj, olist, *relation, q, &total); e != nil {
			return e
		}
		olist.SetTotal(total)
	}
	return tdb.Find(olist.DataPtr()).Error
}

func (c *Client) DelM2MRelation(relation client.RelationShip) error {
	q := fmt.Sprintf("%s = ? and %s = ?", objIDFiled(relation.Left()), objIDFiled(relation.Right()))
	return c.db.Table(tableName(relation)).Where(q, relation.Left().GetPKValue(), relation.Right().GetPKValue()).Delete(nil).Error
}

func (c *Client) ExistM2MRelation(relation client.RelationShip) bool {
	q := fmt.Sprintf("%s = ? and %s = ?", objIDFiled(relation.Left()), objIDFiled(relation.Right()))

	var t int64
	if e := c.db.Table(tableName(relation)).Where(q, relation.Left().GetPKValue(), relation.Right().GetPKValue()).Count(&t).Error; e != nil {
		return false
	}
	return t > 0
}

/*
ListCluster
GetByName
GetManagerCluster
实现ClusterGetter接口
*/

func (c *Client) ListCluster() []string {
	var clusters []string
	c.db.Table(tableName(&Cluster{})).Pluck("cluster_name", &clusters)
	return clusters
}

func (c *Client) GetByName(name string) (agentAddr, mode string, agentcert, agentkey, agentca, kubeconfig []byte, err error) {
	var cluster Cluster
	if err = c.db.First(&cluster, "cluster_name = ?", name).Error; err != nil {
		return
	}
	agentcert = []byte(cluster.AgentCert)
	agentkey = []byte(cluster.AgentKey)
	agentca = []byte(cluster.AgentCA)
	agentAddr = cluster.AgentAddr
	mode = cluster.Mode
	kubeconfig = []byte(cluster.KubeConfig)
	return
}

func (c *Client) GetManagerCluster(ctx context.Context) (string, error) {
	ret := []string{}
	cluster := &Cluster{Primary: true}
	if err := c.db.Where(cluster).Model(cluster).Pluck("cluster_name", &ret).Error; err != nil {
		return "", err
	}
	if len(ret) == 0 {
		return "", errors.New("no manager cluster found")
	}
	return ret[0], nil
}
