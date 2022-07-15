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

package orm

import (
	"database/sql"
	"fmt"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
)

type MySQLOptions struct {
	Addr     string `yaml:"addr" default:"127.0.0.1" help:"mysql host"`
	Username string `yaml:"username" default:"root" help:"mysql username"`
	Password string `yaml:"password" default:"root_password" help:"mysql password"`
	Database string `yaml:"database" default:"localdb" help:"mysql database"`
}

func NewDatabaseInstance(opts *MySQLOptions) (*gorm.DB, error) {
	cfg := &driver.Config{
		User:                 opts.Username,
		Passwd:               opts.Password,
		Addr:                 opts.Addr,
		DBName:               opts.Database,
		Net:                  "tcp",
		ParseTime:            true,
		Collation:            "utf8_general_ci",
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger: log.NewDefaultGormZapLogger(),
	})
}

func ExecuteMigrate(opts *MySQLOptions) error {
	cfg := &driver.Config{
		User:                 opts.Username,
		Passwd:               opts.Password,
		Addr:                 opts.Addr,
		Net:                  "tcp",
		ParseTime:            true,
		Collation:            "utf8_general_ci",
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	tmpdb, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	sqlStr := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4;", opts.Database)
	if _, err := tmpdb.Exec(sqlStr); err != nil {
		return err
	}
	defer tmpdb.Close()

	cfg.DBName = opts.Database
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger: log.NewDefaultGormZapLogger(),
	})
	if err != nil {
		return err
	}
	return Migrate(db)
}
