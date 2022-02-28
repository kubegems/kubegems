package database

import (
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
)

type Options struct {
	Addr     string `json:"addr" description:"mysql host addr"`
	Username string `json:"username" description:"mysql username"`
	Password string `json:"password" description:"mysql password"`
	Database string `json:"database" description:"database to use"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Addr:     "gems-mysql:3306",
		Username: "root",
		Password: "",
		Database: "gemcloud",
	}
}

type Database struct {
	db      *gorm.DB
	options *Options
	*DatabaseHelper
}

func (o *Database) DB() *gorm.DB {
	return o.db
}

func (o *Database) Options() *Options {
	return o.options
}

func NewDatabase(options *Options) (*Database, error) {
	db, err := gorm.Open(mysql.Open(options.ToDsn()), &gorm.Config{
		Logger: log.NewDefaultGormZapLogger(),
	})
	if err != nil {
		return nil, err
	}
	return &Database{
		db:             db,
		options:        options,
		DatabaseHelper: &DatabaseHelper{DB: db},
	}, nil
}

func (opts *Options) ToDsnWithOutDB() (string, string) {
	cfg := opts.ToDriverConfig()
	dbname := cfg.DBName
	cfg.DBName = ""
	return cfg.FormatDSN(), dbname
}

func (opts *Options) ToDsn() string {
	cfg := opts.ToDriverConfig()
	return cfg.FormatDSN()
}

func (opts *Options) ToDriverConfig() *driver.Config {
	cfg := &driver.Config{
		User:                 opts.Username,
		Passwd:               opts.Password,
		Net:                  "tcp",
		Addr:                 opts.Addr,
		DBName:               opts.Database,
		ParseTime:            true,
		Collation:            "utf8_general_ci",
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return cfg
}
