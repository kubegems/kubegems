package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/VividCortex/mysqlerr"
	driver "github.com/go-sql-driver/mysql"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/prometheus"
	"github.com/kubegems/gems/pkg/utils/redis"
	"gorm.io/gorm"
)

type MySQLOptions = database.MySQLOptions

var NewDefaultMySQLOptions = database.NewDefaultMySQLOptions

func createDatabaseIfNotExists(dsn, dbname string) error {
	tmpdb, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer tmpdb.Close()
	sqlStr := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4;", dbname)
	if _, err := tmpdb.Exec(sqlStr); err != nil {
		return err
	}
	if _, err := tmpdb.Exec(sqlStr); err != nil {
		return err
	}
	return nil
}

func MigrateDatabaseAndInitData(opts *MySQLOptions, redisopts *redis.Options) error {
	rediscli, err := redis.NewClient(redisopts)
	if err != nil {
		return err
	}
	// hook 中需要redis
	if err := InitRedis(rediscli); err != nil {
		return err
	}

	// init database schema
	if err := createDatabaseIfNotExists(opts.ToDsnWithOutDB()); err != nil {
		return err
	}

	db, err := database.NewDatabase(opts)
	if err != nil {
		return err
	}

	if err := migrateModels(db.DB()); err != nil {
		return err
	}

	if opts.InitData {
		if err := initBaseData(db.DB()); err != nil {
			return err
		}
	}
	return nil
}

// 初始化系统角色和系统管理员
func initBaseData(db *gorm.DB) error {
	active := true
	sysadmin := SystemRole{ID: 1, RoleName: "系统管理员", RoleCode: "sysadmin"}
	normal := SystemRole{ID: 2, RoleName: "普通用户", RoleCode: "normal"}
	admin := User{
		ID:           1,
		Username:     "admin",
		IsActive:     &active,
		SystemRoleID: 1,
		// 默认密码 demo!@#admin 生成 htpasswd -bnBC 10 "" 'demo!@#admin'| tr -d ':\n'
		Password: "$2y$10$n3GZNQIB8jTMJS//1DY04eoRC7dQiPVp8MbFP/vPcaNJU96/MmPci",
	}
	if e := db.FirstOrCreate(&sysadmin, sysadmin.ID).Error; e != nil {
		return e
	}
	if e := db.FirstOrCreate(&normal, normal.ID).Error; e != nil {
		return e
	}
	if e := db.FirstOrCreate(&admin, admin.ID).Error; e != nil {
		return e
	}

	metricCfg := Config{
		ID:      1,
		Name:    MetricConfig,
		Content: prometheus.DefaultMetricConfigContent(),
	}
	if err := db.FirstOrCreate(&metricCfg, metricCfg.ID).Error; err != nil {
		return err
	}
	return nil
}

func migrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		// 审计表
		&AuditLog{},
		// 用户表
		&User{},
		// 系统角色表
		&SystemRole{},
		// 租户表
		&Tenant{},
		// 租户成员关系表
		&TenantUserRels{},
		// 租户集群资源表
		&TenantResourceQuota{},
		// 项目表
		&Project{},
		// 项目成员关系表
		&ProjectUserRels{},
		// 环境表
		&Environment{},
		// 环境成员关系表
		&EnvironmentUserRels{},
		// 应用表
		&Application{},
		// 集群表
		&Cluster{},
		// 镜像仓库表
		&Registry{},
		// 日志查询历史表
		&LogQueryHistory{},
		// 日志查询快照表
		&LogQuerySnapshot{},
		// workload 资源建议表
		&Workload{},
		// 容器资源建议表
		&Container{},
		// 租户集群资源申请表
		&TenantResourceQuotaApply{},
		// ??
		&EnvironmentResource{},
		// 消息表
		&Message{},
		// 用户消息表
		&UserMessageStatus{},
		// helmChart仓库
		&ChartRepo{},
		// 虚拟空间表
		&VirtualSpace{},
		// 虚拟空间用户表
		&VirtualSpaceUserRels{},
		// 虚拟域名表
		&VirtualDomain{},
		// 告警信息表
		&AlertInfo{},
		// 告警消息表
		&AlertMessage{},
		// 监控面板表
		&MetricDashborad{},
		// 配置
		&Config{},
	)
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func GetErrMessage(err error) string {
	me := &driver.MySQLError{}
	if !errors.As(err, &me) {
		return err.Error()
	}
	switch me.Number {
	case mysqlerr.ER_DUP_ENTRY:
		return fmt.Sprintf("存在重名对象(code=%v)", me.Number)
	case mysqlerr.ER_DATA_TOO_LONG:
		return fmt.Sprintf("数据超长(code=%v)", me.Number)
	case mysqlerr.ER_TRUNCATED_WRONG_VALUE:
		return fmt.Sprintf("日期格式错误(code=%v)", me.Number)
	case mysqlerr.ER_NO_REFERENCED_ROW_2:
		return fmt.Sprintf("系统错误(外键关联数据出错 code=%v)", me.Number)
	case mysqlerr.ER_ROW_IS_REFERENCED_2:
		return fmt.Sprintf("系统错误(外键关联数据错误 code=%v)", me.Number)
	default:
		return fmt.Sprintf("系统错误(code=%v, message=%v)!", me.Number, me.Message)
	}
}
