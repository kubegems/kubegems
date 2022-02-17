package orm

/*
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

*/
