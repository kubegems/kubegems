package models

import "time"

const (
	ResVirtualDomain = "virtualDomain"
)

type VirtualDomain struct {
	ID                uint   `gorm:"primarykey"`
	VirtualDomainName string `gorm:"type:varchar(50);uniqueIndex"`

	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	IsActive  bool // 是否激活
	CreatedBy string
}
