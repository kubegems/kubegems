package database

import (
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/spf13/pflag"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
)

type MySQLOptions struct {
	Addr     string `yaml:"addr" line_comment:"mysql host addr"`
	Username string `yaml:"username" line_comment:"mysql username"`
	Password string `yaml:"password" line_comment:"mysql password"`
	Database string `yaml:"databse" line_comment:"mysql database to use"`
	InitData bool   `yaml:"initdata" line_comment:"initdata switch"`
}

func (o *MySQLOptions) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, utils.JoinFlagName(prefix, "addr"), o.Addr, "mysql addr")
	fs.StringVar(&o.Username, utils.JoinFlagName(prefix, "username"), o.Username, "mysql username")
	fs.StringVar(&o.Password, utils.JoinFlagName(prefix, "password"), o.Password, "mysql password")
	fs.StringVar(&o.Database, utils.JoinFlagName(prefix, "database"), o.Database, "mysql database")
	fs.BoolVar(&o.InitData, utils.JoinFlagName(prefix, "initdata"), o.InitData, "mysql initdata")
}

func NewDefaultMySQLOptions() *MySQLOptions {
	return &MySQLOptions{
		Addr:     "gems-mysql:3306",
		Username: "root",
		Password: "",
		Database: "gemcloud",
	}
}

type Database struct {
	db      *gorm.DB
	options *MySQLOptions
	*DatabaseHelper
}

func (o *Database) DB() *gorm.DB {
	return o.db
}

func (o *Database) Options() *MySQLOptions {
	return o.options
}

func NewDatabase(options *MySQLOptions) (*Database, error) {
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

func (opts *MySQLOptions) ToDsnWithOutDB() (string, string) {
	cfg := opts.ToDriverConfig()
	dbname := cfg.DBName
	cfg.DBName = ""
	return cfg.FormatDSN(), dbname
}

func (opts *MySQLOptions) ToDsn() string {
	cfg := opts.ToDriverConfig()
	return cfg.FormatDSN()
}

func (opts *MySQLOptions) ToDriverConfig() *driver.Config {
	cfg := &driver.Config{
		User:                 opts.Username,
		Passwd:               opts.Password,
		Net:                  "tcp",
		Addr:                 opts.Addr,
		DBName:               opts.Database,
		ParseTime:            true,              // 不支持配置，否则出问题
		Collation:            "utf8_general_ci", // 不支持配置，否则出问题
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return cfg
}
