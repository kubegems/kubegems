package models

import "time"

const (
	ResVirtualDomain = "virtualDomain"
)

/*
ALTER TABLE virtual_domains RENAME COLUMN virtual_domain_name TO name;
*/

type VirtualDomain struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"type:varchar(50);uniqueIndex"`

	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	IsActive  bool // 是否激活
	CreatedBy string
}
