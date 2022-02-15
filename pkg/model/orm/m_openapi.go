package orm

// OpenAPP 第三方的app
// +gen type:object pkcolume:id pkfield:ID
type OpenAPP struct {
	Name      string `gorm:"unique"`
	ID        uint
	AppID     string
	AppSecret string
	// 系统权限范围,空则表示什么操作都不行,默认是ReadWorkload
	PermScopes string `sql:"DEFAULT:'ReadWorkload'"`
	// 可操作租户范围，通过id列表表示，逗号分隔，可以用通配符 *，表示所有, 默认*
	TenantScope string `sql:"DEFAULT:'*'"`
	// 访问频率限制，空则表示不限制,表示每分钟可以访问的次数，默认30
	RequestLimiter int `sql:"DEFAULT:30"`
}
