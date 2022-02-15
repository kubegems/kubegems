package forms

import "time"

// +genform object:Registry
type RegistryCommon struct {
	BaseForm
	ID              uint
	RegistryName    string
	RegistryAddress string
	UpdateTime      *time.Time
	Creator         *UserCommon
	CreatorID       uint
	Project         *ProjectCommon
	ProjectID       uint
	IsDefault       bool
}

// +genform object:Registry
type RegistryDetail struct {
	BaseForm
	ID              uint
	RegistryName    string
	RegistryAddress string
	Username        string
	Password        string
	Creator         *UserCommon
	UpdateTime      *time.Time
	CreatorID       uint
	Project         *ProjectCommon
	ProjectID       uint
	IsDefault       bool
}
