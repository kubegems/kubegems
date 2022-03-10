package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
审计日志的描述格式为:
	{用户名} {时间} {操作}
操作:
	{动作,增删查改} {资源类型} {资源名字}
*/
// AuditLog 审计日志表
type AuditLog struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	// 操作用户
	Username string `gorm:"type:varchar(50)"`
	// 所属租户
	Tenant string `gorm:"type:varchar(50)"`
	// 操作模块 (资源类型，租户，项目，环境，报警规则等等)
	Module string `gorm:"type:varchar(512)"`
	// 模块名字
	Name string `gorm:"type:varchar(512)"`
	// 动作名字 (启用，禁用，开启，关闭，添加，删除，修改等)
	Action string `gorm:"type:varchar(255)"`
	// 是否成功 请求是否成功
	Success bool
	// 客户端ip 发起请求的客户端IP
	ClientIP string `gorm:"type:varchar(255)"`
	// 标签 记录一些额外的环境租户等数据信息
	Labels datatypes.JSON
	// 原始数据 记录的是request和response以及http_code
	RawData datatypes.JSON
}
