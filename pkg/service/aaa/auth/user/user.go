package user

import "time"

type CommonUserIface interface {
	GetID() uint
	GetSystemRoleID() uint
	GetUsername() string
	GetUserKind() string
	GetEmail() string
	GetSource() string
	SetLastLogin(*time.Time)
	UnmarshalBinary(data []byte) error
	MarshalBinary() (data []byte, err error)
}
