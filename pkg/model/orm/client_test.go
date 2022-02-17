package orm

import (
	"context"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/pkg/model/client"
)

var (
	gormdb *gorm.DB
	mock   sqlmock.Sqlmock
)

func setup() {
	db, _mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		panic(err)
	}
	_mock.ExpectQuery("SELECT VERSION()").WillReturnRows(sqlmock.NewRows([]string{"VERSION()"}).AddRow("5.7.33"))
	_db, err := gorm.Open(mysql.New(mysql.Config{
		Conn: db,
	}))
	if err != nil {
		panic(err)
	}
	gormdb = _db
	mock = _mock
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestClient_Exist(t *testing.T) {
	c := &Client{
		db:        gormdb,
		relations: map[string]*client.Relation{},
	}
	user1 := User{ID: 2}
	mock.ExpectQuery("SELECT count(*) FROM `users` WHERE `users`.`id` = ?").WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"cout(*)"}).AddRow(1))
	assert.True(t, c.Exist(context.Background(), &user1))

	user2 := User{ID: 33}
	mock.ExpectQuery("SELECT count(*) FROM `users` WHERE `users`.`id` = ?").WithArgs(33).WillReturnRows(sqlmock.NewRows([]string{"cout(*)"}).AddRow(0))
	assert.False(t, c.Exist(context.Background(), &user2))
}

func TestClient_Get(t *testing.T) {
	c := &Client{
		db:        gormdb,
		relations: map[string]*client.Relation{},
	}
	user1 := User{ID: 2}
	mock.ExpectQuery(
		"SELECT * FROM `users` WHERE `users`.`id` = ? ORDER BY `users`.`id` LIMIT 1",
	).WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(1, "test"))
	c.Get(context.Background(), &user1)
	assert.Equal(t, user1.Username, "test")
}

func TestClient_List(t *testing.T) {
	c := &Client{
		db:        gormdb,
		relations: map[string]*client.Relation{},
	}
	list := UserList{}
	mock.ExpectQuery(
		"SELECT * FROM `users`",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(1, "test"))
	c.List(context.Background(), &list)
	assert.Equal(t, list.Items[0].Username, "test")
}

func TestClient_Update(t *testing.T) {
}

func TestClient_Create(t *testing.T) {
}

func TestClient_Delete(t *testing.T) {}

func TestClient_Count(t *testing.T) {}
