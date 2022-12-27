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

package database

import (
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
)

type Options struct {
	Addr      string `json:"addr" description:"mysql host addr"`
	Username  string `json:"username" description:"mysql username"`
	Password  string `json:"password" description:"mysql password"`
	Database  string `json:"database" description:"database to use"`
	Collation string `json:"collation" description:"collation to use"`
}

func NewDefaultOptions() *Options {
	return &Options{
		Addr:      "kubegems-mysql:3306",
		Username:  "root",
		Password:  "",
		Database:  "kubegems",
		Collation: "utf8mb4_unicode_ci",
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
	if err := db.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables())); err != nil {
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
		Collation:            opts.Collation,
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return cfg
}
