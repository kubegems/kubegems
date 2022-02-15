package client

import "time"

type SystemRole struct {
	ID       uint
	RoleName string
	RoleCode string
}

type CommonUser struct {
	ID           uint
	Username     string
	Email        string
	Phone        string
	Password     string
	IsActive     *bool
	Kind         string
	Source       string
	CreatedAt    *time.Time
	LastLoginAt  *time.Time
	SystemRole   *SystemRole
	SystemRoleID uint
}

func (u *CommonUser) GetID() uint {
	return u.ID
}

func (u *CommonUser) GetSystemRoleID() uint {
	return u.SystemRoleID
}

func (u *CommonUser) GetUsername() string {
	return u.Username
}

func (u *CommonUser) GetKind() string {
	return u.Kind
}

func (u *CommonUser) GetEmail() string {
	return u.Email
}

func (u *CommonUser) GetSource() string {
	return u.Source
}
