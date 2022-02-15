package forms

import (
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/orm"
)

type WorkloadCommonList struct {
	BaseListForm
	Items []*WorkloadCommon
}

func (ul *WorkloadCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.WorkloadList{}
	return ul.objectlist
}

func (ul *WorkloadCommonList) AsListData() []*WorkloadCommon {
	us := ul.objectlist.(*orm.WorkloadList)
	return Convert_Workload_WorkloadCommon_arr(us.Items)
}

func (r *WorkloadCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_WorkloadCommon_Workload(r)
	}
	return r.object
}

func (u *WorkloadCommon) Data() *WorkloadCommon {
	return Convert_Workload_WorkloadCommon(u.object.(*orm.Workload))
}

func Convert_WorkloadCommon_Workload(f *WorkloadCommon) *orm.Workload {
	r := &orm.Workload{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.ClusterName = f.ClusterName
	r.Namespace = f.Namespace
	r.Containers = Convert_ContainerCommon_Container_arr(f.Containers)
	r.ID = f.ID
	r.Type = f.Type
	r.Name = f.Name
	r.CPULimitStdvar = f.CPULimitStdvar
	r.MemoryLimitStdvar = f.MemoryLimitStdvar
	return r
}
func Convert_Workload_WorkloadCommon(f *orm.Workload) *WorkloadCommon {
	if f == nil {
		return nil
	}
	var r WorkloadCommon
	r.CreatedAt = f.CreatedAt
	r.ClusterName = f.ClusterName
	r.Namespace = f.Namespace
	r.Containers = Convert_Container_ContainerCommon_arr(f.Containers)
	r.ID = f.ID
	r.Type = f.Type
	r.Name = f.Name
	r.CPULimitStdvar = f.CPULimitStdvar
	r.MemoryLimitStdvar = f.MemoryLimitStdvar
	return &r
}
func Convert_WorkloadCommon_Workload_arr(arr []*WorkloadCommon) []*orm.Workload {
	r := []*orm.Workload{}
	for _, u := range arr {
		r = append(r, Convert_WorkloadCommon_Workload(u))
	}
	return r
}

func Convert_Workload_WorkloadCommon_arr(arr []*orm.Workload) []*WorkloadCommon {
	r := []*WorkloadCommon{}
	for _, u := range arr {
		r = append(r, Convert_Workload_WorkloadCommon(u))
	}
	return r
}

type VirtualSpaceUserRelCommonList struct {
	BaseListForm
	Items []*VirtualSpaceUserRelCommon
}

func (ul *VirtualSpaceUserRelCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.VirtualSpaceUserRelList{}
	return ul.objectlist
}

func (ul *VirtualSpaceUserRelCommonList) AsListData() []*VirtualSpaceUserRelCommon {
	us := ul.objectlist.(*orm.VirtualSpaceUserRelList)
	return Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon_arr(us.Items)
}

func (r *VirtualSpaceUserRelCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel(r)
	}
	return r.object
}

func (u *VirtualSpaceUserRelCommon) Data() *VirtualSpaceUserRelCommon {
	return Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon(u.object.(*orm.VirtualSpaceUserRel))
}

func Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel(f *VirtualSpaceUserRelCommon) *orm.VirtualSpaceUserRel {
	r := &orm.VirtualSpaceUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.UserID = f.UserID
	r.User = Convert_UserCommon_User(f.User)
	r.Role = f.Role
	r.ID = f.ID
	r.VirtualSpaceID = f.VirtualSpaceID
	r.VirtualSpace = Convert_VirtualSpaceCommon_VirtualSpace(f.VirtualSpace)
	return r
}
func Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon(f *orm.VirtualSpaceUserRel) *VirtualSpaceUserRelCommon {
	if f == nil {
		return nil
	}
	var r VirtualSpaceUserRelCommon
	r.User = Convert_User_UserCommon(f.User)
	r.Role = f.Role
	r.ID = f.ID
	r.VirtualSpaceID = f.VirtualSpaceID
	r.VirtualSpace = Convert_VirtualSpace_VirtualSpaceCommon(f.VirtualSpace)
	r.UserID = f.UserID
	return &r
}
func Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel_arr(arr []*VirtualSpaceUserRelCommon) []*orm.VirtualSpaceUserRel {
	r := []*orm.VirtualSpaceUserRel{}
	for _, u := range arr {
		r = append(r, Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel(u))
	}
	return r
}

func Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon_arr(arr []*orm.VirtualSpaceUserRel) []*VirtualSpaceUserRelCommon {
	r := []*VirtualSpaceUserRelCommon{}
	for _, u := range arr {
		r = append(r, Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon(u))
	}
	return r
}

type VirtualSpaceCommonList struct {
	BaseListForm
	Items []*VirtualSpaceCommon
}

func (ul *VirtualSpaceCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.VirtualSpaceList{}
	return ul.objectlist
}

func (ul *VirtualSpaceCommonList) AsListData() []*VirtualSpaceCommon {
	us := ul.objectlist.(*orm.VirtualSpaceList)
	return Convert_VirtualSpace_VirtualSpaceCommon_arr(us.Items)
}

func (r *VirtualSpaceCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_VirtualSpaceCommon_VirtualSpace(r)
	}
	return r.object
}

func (u *VirtualSpaceCommon) Data() *VirtualSpaceCommon {
	return Convert_VirtualSpace_VirtualSpaceCommon(u.object.(*orm.VirtualSpace))
}

func Convert_VirtualSpaceCommon_VirtualSpace(f *VirtualSpaceCommon) *orm.VirtualSpace {
	r := &orm.VirtualSpace{}
	if f == nil {
		return nil
	}
	f.object = r
	r.VirtualSpaceName = f.VirtualSpaceName
	r.ID = f.ID
	return r
}
func Convert_VirtualSpace_VirtualSpaceCommon(f *orm.VirtualSpace) *VirtualSpaceCommon {
	if f == nil {
		return nil
	}
	var r VirtualSpaceCommon
	r.ID = f.ID
	r.VirtualSpaceName = f.VirtualSpaceName
	return &r
}
func Convert_VirtualSpaceCommon_VirtualSpace_arr(arr []*VirtualSpaceCommon) []*orm.VirtualSpace {
	r := []*orm.VirtualSpace{}
	for _, u := range arr {
		r = append(r, Convert_VirtualSpaceCommon_VirtualSpace(u))
	}
	return r
}

func Convert_VirtualSpace_VirtualSpaceCommon_arr(arr []*orm.VirtualSpace) []*VirtualSpaceCommon {
	r := []*VirtualSpaceCommon{}
	for _, u := range arr {
		r = append(r, Convert_VirtualSpace_VirtualSpaceCommon(u))
	}
	return r
}

type VirtualDomainCommonList struct {
	BaseListForm
	Items []*VirtualDomainCommon
}

func (ul *VirtualDomainCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.VirtualDomainList{}
	return ul.objectlist
}

func (ul *VirtualDomainCommonList) AsListData() []*VirtualDomainCommon {
	us := ul.objectlist.(*orm.VirtualDomainList)
	return Convert_VirtualDomain_VirtualDomainCommon_arr(us.Items)
}

func (r *VirtualDomainCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_VirtualDomainCommon_VirtualDomain(r)
	}
	return r.object
}

func (u *VirtualDomainCommon) Data() *VirtualDomainCommon {
	return Convert_VirtualDomain_VirtualDomainCommon(u.object.(*orm.VirtualDomain))
}

func Convert_VirtualDomainCommon_VirtualDomain(f *VirtualDomainCommon) *orm.VirtualDomain {
	r := &orm.VirtualDomain{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.VirtualDomainName = f.VirtualDomainName
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	r.IsActive = f.IsActive
	r.CreatedBy = f.CreatedBy
	return r
}
func Convert_VirtualDomain_VirtualDomainCommon(f *orm.VirtualDomain) *VirtualDomainCommon {
	if f == nil {
		return nil
	}
	var r VirtualDomainCommon
	r.ID = f.ID
	r.VirtualDomainName = f.VirtualDomainName
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	r.IsActive = f.IsActive
	r.CreatedBy = f.CreatedBy
	return &r
}
func Convert_VirtualDomainCommon_VirtualDomain_arr(arr []*VirtualDomainCommon) []*orm.VirtualDomain {
	r := []*orm.VirtualDomain{}
	for _, u := range arr {
		r = append(r, Convert_VirtualDomainCommon_VirtualDomain(u))
	}
	return r
}

func Convert_VirtualDomain_VirtualDomainCommon_arr(arr []*orm.VirtualDomain) []*VirtualDomainCommon {
	r := []*VirtualDomainCommon{}
	for _, u := range arr {
		r = append(r, Convert_VirtualDomain_VirtualDomainCommon(u))
	}
	return r
}

type UserSettingList struct {
	BaseListForm
	Items []*UserSetting
}

func (ul *UserSettingList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserSettingList) AsListData() []*UserSetting {
	us := ul.objectlist.(*orm.UserList)
	return Convert_User_UserSetting_arr(us.Items)
}

func (r *UserSetting) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserSetting_User(r)
	}
	return r.object
}

func (u *UserSetting) Data() *UserSetting {
	return Convert_User_UserSetting(u.object.(*orm.User))
}

func Convert_UserSetting_User(f *UserSetting) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Phone = f.Phone
	r.Email = f.Email
	r.Password = f.Password
	r.IsActive = f.IsActive
	r.SystemRole = Convert_SystemRoleCommon_SystemRole(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	r.ID = f.ID
	r.Username = f.Username
	return r
}
func Convert_User_UserSetting(f *orm.User) *UserSetting {
	if f == nil {
		return nil
	}
	var r UserSetting
	r.ID = f.ID
	r.Username = f.Username
	r.Phone = f.Phone
	r.Email = f.Email
	r.Password = f.Password
	r.IsActive = f.IsActive
	r.SystemRole = Convert_SystemRole_SystemRoleCommon(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return &r
}
func Convert_UserSetting_User_arr(arr []*UserSetting) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserSetting_User(u))
	}
	return r
}

func Convert_User_UserSetting_arr(arr []*orm.User) []*UserSetting {
	r := []*UserSetting{}
	for _, u := range arr {
		r = append(r, Convert_User_UserSetting(u))
	}
	return r
}

type UserMessageStatusCommonList struct {
	BaseListForm
	Items []*UserMessageStatusCommon
}

func (ul *UserMessageStatusCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.UserMessageStatusList{}
	return ul.objectlist
}

func (ul *UserMessageStatusCommonList) AsListData() []*UserMessageStatusCommon {
	us := ul.objectlist.(*orm.UserMessageStatusList)
	return Convert_UserMessageStatus_UserMessageStatusCommon_arr(us.Items)
}

func (r *UserMessageStatusCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserMessageStatusCommon_UserMessageStatus(r)
	}
	return r.object
}

func (u *UserMessageStatusCommon) Data() *UserMessageStatusCommon {
	return Convert_UserMessageStatus_UserMessageStatusCommon(u.object.(*orm.UserMessageStatus))
}

func Convert_UserMessageStatusCommon_UserMessageStatus(f *UserMessageStatusCommon) *orm.UserMessageStatus {
	r := &orm.UserMessageStatus{}
	if f == nil {
		return nil
	}
	f.object = r
	r.IsRead = f.IsRead
	r.ID = f.ID
	r.UserID = f.UserID
	r.User = Convert_UserCommon_User(f.User)
	r.MessageID = f.MessageID
	r.Message = Convert_MessageCommon_Message(f.Message)
	return r
}
func Convert_UserMessageStatus_UserMessageStatusCommon(f *orm.UserMessageStatus) *UserMessageStatusCommon {
	if f == nil {
		return nil
	}
	var r UserMessageStatusCommon
	r.ID = f.ID
	r.UserID = f.UserID
	r.User = Convert_User_UserCommon(f.User)
	r.MessageID = f.MessageID
	r.Message = Convert_Message_MessageCommon(f.Message)
	r.IsRead = f.IsRead
	return &r
}
func Convert_UserMessageStatusCommon_UserMessageStatus_arr(arr []*UserMessageStatusCommon) []*orm.UserMessageStatus {
	r := []*orm.UserMessageStatus{}
	for _, u := range arr {
		r = append(r, Convert_UserMessageStatusCommon_UserMessageStatus(u))
	}
	return r
}

func Convert_UserMessageStatus_UserMessageStatusCommon_arr(arr []*orm.UserMessageStatus) []*UserMessageStatusCommon {
	r := []*UserMessageStatusCommon{}
	for _, u := range arr {
		r = append(r, Convert_UserMessageStatus_UserMessageStatusCommon(u))
	}
	return r
}

type UserInternalList struct {
	BaseListForm
	Items []*UserInternal
}

func (ul *UserInternalList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserInternalList) AsListData() []*UserInternal {
	us := ul.objectlist.(*orm.UserList)
	return Convert_User_UserInternal_arr(us.Items)
}

func (r *UserInternal) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserInternal_User(r)
	}
	return r.object
}

func (u *UserInternal) Data() *UserInternal {
	return Convert_User_UserInternal(u.object.(*orm.User))
}

func Convert_UserInternal_User(f *UserInternal) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Username = f.Username
	r.Password = f.Password
	r.CreatedAt = f.CreatedAt
	r.LastLoginAt = f.LastLoginAt
	r.SystemRoleID = f.SystemRoleID
	r.ID = f.ID
	r.Email = f.Email
	r.Role = f.Role
	r.Phone = f.Phone
	r.Source = f.Source
	r.IsActive = f.IsActive
	r.SystemRole = Convert_SystemRoleCommon_SystemRole(f.SystemRole)
	return r
}
func Convert_User_UserInternal(f *orm.User) *UserInternal {
	if f == nil {
		return nil
	}
	var r UserInternal
	r.SystemRoleID = f.SystemRoleID
	r.Username = f.Username
	r.Password = f.Password
	r.CreatedAt = f.CreatedAt
	r.LastLoginAt = f.LastLoginAt
	r.Source = f.Source
	r.IsActive = f.IsActive
	r.SystemRole = Convert_SystemRole_SystemRoleCommon(f.SystemRole)
	r.ID = f.ID
	r.Email = f.Email
	r.Role = f.Role
	r.Phone = f.Phone
	return &r
}
func Convert_UserInternal_User_arr(arr []*UserInternal) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserInternal_User(u))
	}
	return r
}

func Convert_User_UserInternal_arr(arr []*orm.User) []*UserInternal {
	r := []*UserInternal{}
	for _, u := range arr {
		r = append(r, Convert_User_UserInternal(u))
	}
	return r
}

type UserDetailList struct {
	BaseListForm
	Items []*UserDetail
}

func (ul *UserDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserDetailList) AsListData() []*UserDetail {
	us := ul.objectlist.(*orm.UserList)
	return Convert_User_UserDetail_arr(us.Items)
}

func (r *UserDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserDetail_User(r)
	}
	return r.object
}

func (u *UserDetail) Data() *UserDetail {
	return Convert_User_UserDetail(u.object.(*orm.User))
}

func Convert_UserDetail_User(f *UserDetail) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Username = f.Username
	r.Email = f.Email
	r.LastLoginAt = f.LastLoginAt
	r.SystemRoleID = f.SystemRoleID
	r.Role = f.Role
	r.ID = f.ID
	r.Phone = f.Phone
	r.Source = f.Source
	r.IsActive = f.IsActive
	r.CreatedAt = f.CreatedAt
	r.SystemRole = Convert_SystemRoleCommon_SystemRole(f.SystemRole)
	return r
}
func Convert_User_UserDetail(f *orm.User) *UserDetail {
	if f == nil {
		return nil
	}
	var r UserDetail
	r.Role = f.Role
	r.Username = f.Username
	r.Email = f.Email
	r.LastLoginAt = f.LastLoginAt
	r.SystemRoleID = f.SystemRoleID
	r.CreatedAt = f.CreatedAt
	r.SystemRole = Convert_SystemRole_SystemRoleCommon(f.SystemRole)
	r.ID = f.ID
	r.Phone = f.Phone
	r.Source = f.Source
	r.IsActive = f.IsActive
	return &r
}
func Convert_UserDetail_User_arr(arr []*UserDetail) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserDetail_User(u))
	}
	return r
}

func Convert_User_UserDetail_arr(arr []*orm.User) []*UserDetail {
	r := []*UserDetail{}
	for _, u := range arr {
		r = append(r, Convert_User_UserDetail(u))
	}
	return r
}

type UserCommonList struct {
	BaseListForm
	Items []*UserCommon
}

func (ul *UserCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserCommonList) AsListData() []*UserCommon {
	us := ul.objectlist.(*orm.UserList)
	return Convert_User_UserCommon_arr(us.Items)
}

func (r *UserCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserCommon_User(r)
	}
	return r.object
}

func (u *UserCommon) Data() *UserCommon {
	return Convert_User_UserCommon(u.object.(*orm.User))
}

func Convert_UserCommon_User(f *UserCommon) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Username = f.Username
	r.Email = f.Email
	r.Role = f.Role
	r.ID = f.ID
	return r
}
func Convert_User_UserCommon(f *orm.User) *UserCommon {
	if f == nil {
		return nil
	}
	var r UserCommon
	r.ID = f.ID
	r.Username = f.Username
	r.Email = f.Email
	r.Role = f.Role
	return &r
}
func Convert_UserCommon_User_arr(arr []*UserCommon) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserCommon_User(u))
	}
	return r
}

func Convert_User_UserCommon_arr(arr []*orm.User) []*UserCommon {
	r := []*UserCommon{}
	for _, u := range arr {
		r = append(r, Convert_User_UserCommon(u))
	}
	return r
}

type TenantUserRelCommonList struct {
	BaseListForm
	Items []*TenantUserRelCommon
}

func (ul *TenantUserRelCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.TenantUserRelList{}
	return ul.objectlist
}

func (ul *TenantUserRelCommonList) AsListData() []*TenantUserRelCommon {
	us := ul.objectlist.(*orm.TenantUserRelList)
	return Convert_TenantUserRel_TenantUserRelCommon_arr(us.Items)
}

func (r *TenantUserRelCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantUserRelCommon_TenantUserRel(r)
	}
	return r.object
}

func (u *TenantUserRelCommon) Data() *TenantUserRelCommon {
	return Convert_TenantUserRel_TenantUserRelCommon(u.object.(*orm.TenantUserRel))
}

func Convert_TenantUserRelCommon_TenantUserRel(f *TenantUserRelCommon) *orm.TenantUserRel {
	r := &orm.TenantUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.TenantID = f.TenantID
	r.User = Convert_UserCommon_User(f.User)
	r.UserID = f.UserID
	r.Role = f.Role
	return r
}
func Convert_TenantUserRel_TenantUserRelCommon(f *orm.TenantUserRel) *TenantUserRelCommon {
	if f == nil {
		return nil
	}
	var r TenantUserRelCommon
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.TenantID = f.TenantID
	r.User = Convert_User_UserCommon(f.User)
	r.UserID = f.UserID
	r.Role = f.Role
	r.ID = f.ID
	return &r
}
func Convert_TenantUserRelCommon_TenantUserRel_arr(arr []*TenantUserRelCommon) []*orm.TenantUserRel {
	r := []*orm.TenantUserRel{}
	for _, u := range arr {
		r = append(r, Convert_TenantUserRelCommon_TenantUserRel(u))
	}
	return r
}

func Convert_TenantUserRel_TenantUserRelCommon_arr(arr []*orm.TenantUserRel) []*TenantUserRelCommon {
	r := []*TenantUserRelCommon{}
	for _, u := range arr {
		r = append(r, Convert_TenantUserRel_TenantUserRelCommon(u))
	}
	return r
}

type TenantResourceQuotaCommonList struct {
	BaseListForm
	Items []*TenantResourceQuotaCommon
}

func (ul *TenantResourceQuotaCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.TenantResourceQuotaList{}
	return ul.objectlist
}

func (ul *TenantResourceQuotaCommonList) AsListData() []*TenantResourceQuotaCommon {
	us := ul.objectlist.(*orm.TenantResourceQuotaList)
	return Convert_TenantResourceQuota_TenantResourceQuotaCommon_arr(us.Items)
}

func (r *TenantResourceQuotaCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantResourceQuotaCommon_TenantResourceQuota(r)
	}
	return r.object
}

func (u *TenantResourceQuotaCommon) Data() *TenantResourceQuotaCommon {
	return Convert_TenantResourceQuota_TenantResourceQuotaCommon(u.object.(*orm.TenantResourceQuota))
}

func Convert_TenantResourceQuotaCommon_TenantResourceQuota(f *TenantResourceQuotaCommon) *orm.TenantResourceQuota {
	r := &orm.TenantResourceQuota{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Content = f.Content
	r.TenantID = f.TenantID
	r.ClusterID = f.ClusterID
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.TenantResourceQuotaApply = Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(f.TenantResourceQuotaApply)
	r.TenantResourceQuotaApplyID = f.TenantResourceQuotaApplyID
	return r
}
func Convert_TenantResourceQuota_TenantResourceQuotaCommon(f *orm.TenantResourceQuota) *TenantResourceQuotaCommon {
	if f == nil {
		return nil
	}
	var r TenantResourceQuotaCommon
	r.TenantResourceQuotaApplyID = f.TenantResourceQuotaApplyID
	r.ID = f.ID
	r.Content = f.Content
	r.TenantID = f.TenantID
	r.ClusterID = f.ClusterID
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.TenantResourceQuotaApply = Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon(f.TenantResourceQuotaApply)
	return &r
}
func Convert_TenantResourceQuotaCommon_TenantResourceQuota_arr(arr []*TenantResourceQuotaCommon) []*orm.TenantResourceQuota {
	r := []*orm.TenantResourceQuota{}
	for _, u := range arr {
		r = append(r, Convert_TenantResourceQuotaCommon_TenantResourceQuota(u))
	}
	return r
}

func Convert_TenantResourceQuota_TenantResourceQuotaCommon_arr(arr []*orm.TenantResourceQuota) []*TenantResourceQuotaCommon {
	r := []*TenantResourceQuotaCommon{}
	for _, u := range arr {
		r = append(r, Convert_TenantResourceQuota_TenantResourceQuotaCommon(u))
	}
	return r
}

type TenantResourceQuotaApplyCommonList struct {
	BaseListForm
	Items []*TenantResourceQuotaApplyCommon
}

func (ul *TenantResourceQuotaApplyCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.TenantResourceQuotaApplyList{}
	return ul.objectlist
}

func (ul *TenantResourceQuotaApplyCommonList) AsListData() []*TenantResourceQuotaApplyCommon {
	us := ul.objectlist.(*orm.TenantResourceQuotaApplyList)
	return Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon_arr(us.Items)
}

func (r *TenantResourceQuotaApplyCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(r)
	}
	return r.object
}

func (u *TenantResourceQuotaApplyCommon) Data() *TenantResourceQuotaApplyCommon {
	return Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon(u.object.(*orm.TenantResourceQuotaApply))
}

func Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(f *TenantResourceQuotaApplyCommon) *orm.TenantResourceQuotaApply {
	r := &orm.TenantResourceQuotaApply{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Username = f.Username
	r.UpdatedAt = f.UpdatedAt
	r.ID = f.ID
	r.Content = f.Content
	r.Status = f.Status
	return r
}
func Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon(f *orm.TenantResourceQuotaApply) *TenantResourceQuotaApplyCommon {
	if f == nil {
		return nil
	}
	var r TenantResourceQuotaApplyCommon
	r.ID = f.ID
	r.Content = f.Content
	r.Status = f.Status
	r.Username = f.Username
	r.UpdatedAt = f.UpdatedAt
	return &r
}
func Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply_arr(arr []*TenantResourceQuotaApplyCommon) []*orm.TenantResourceQuotaApply {
	r := []*orm.TenantResourceQuotaApply{}
	for _, u := range arr {
		r = append(r, Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(u))
	}
	return r
}

func Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon_arr(arr []*orm.TenantResourceQuotaApply) []*TenantResourceQuotaApplyCommon {
	r := []*TenantResourceQuotaApplyCommon{}
	for _, u := range arr {
		r = append(r, Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon(u))
	}
	return r
}

type TenantDetailList struct {
	BaseListForm
	Items []*TenantDetail
}

func (ul *TenantDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.TenantList{}
	return ul.objectlist
}

func (ul *TenantDetailList) AsListData() []*TenantDetail {
	us := ul.objectlist.(*orm.TenantList)
	return Convert_Tenant_TenantDetail_arr(us.Items)
}

func (r *TenantDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantDetail_Tenant(r)
	}
	return r.object
}

func (u *TenantDetail) Data() *TenantDetail {
	return Convert_Tenant_TenantDetail(u.object.(*orm.Tenant))
}

func Convert_TenantDetail_Tenant(f *TenantDetail) *orm.Tenant {
	r := &orm.Tenant{}
	if f == nil {
		return nil
	}
	f.object = r
	r.IsActive = f.IsActive
	r.Users = Convert_UserCommon_User_arr(f.Users)
	r.ID = f.ID
	r.TenantName = f.TenantName
	r.Remark = f.Remark
	return r
}
func Convert_Tenant_TenantDetail(f *orm.Tenant) *TenantDetail {
	if f == nil {
		return nil
	}
	var r TenantDetail
	r.ID = f.ID
	r.TenantName = f.TenantName
	r.Remark = f.Remark
	r.IsActive = f.IsActive
	r.Users = Convert_User_UserCommon_arr(f.Users)
	return &r
}
func Convert_TenantDetail_Tenant_arr(arr []*TenantDetail) []*orm.Tenant {
	r := []*orm.Tenant{}
	for _, u := range arr {
		r = append(r, Convert_TenantDetail_Tenant(u))
	}
	return r
}

func Convert_Tenant_TenantDetail_arr(arr []*orm.Tenant) []*TenantDetail {
	r := []*TenantDetail{}
	for _, u := range arr {
		r = append(r, Convert_Tenant_TenantDetail(u))
	}
	return r
}

type TenantCommonList struct {
	BaseListForm
	Items []*TenantCommon
}

func (ul *TenantCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.TenantList{}
	return ul.objectlist
}

func (ul *TenantCommonList) AsListData() []*TenantCommon {
	us := ul.objectlist.(*orm.TenantList)
	return Convert_Tenant_TenantCommon_arr(us.Items)
}

func (r *TenantCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantCommon_Tenant(r)
	}
	return r.object
}

func (u *TenantCommon) Data() *TenantCommon {
	return Convert_Tenant_TenantCommon(u.object.(*orm.Tenant))
}

func Convert_TenantCommon_Tenant(f *TenantCommon) *orm.Tenant {
	r := &orm.Tenant{}
	if f == nil {
		return nil
	}
	f.object = r
	r.TenantName = f.TenantName
	r.ID = f.ID
	return r
}
func Convert_Tenant_TenantCommon(f *orm.Tenant) *TenantCommon {
	if f == nil {
		return nil
	}
	var r TenantCommon
	r.ID = f.ID
	r.TenantName = f.TenantName
	return &r
}
func Convert_TenantCommon_Tenant_arr(arr []*TenantCommon) []*orm.Tenant {
	r := []*orm.Tenant{}
	for _, u := range arr {
		r = append(r, Convert_TenantCommon_Tenant(u))
	}
	return r
}

func Convert_Tenant_TenantCommon_arr(arr []*orm.Tenant) []*TenantCommon {
	r := []*TenantCommon{}
	for _, u := range arr {
		r = append(r, Convert_Tenant_TenantCommon(u))
	}
	return r
}

type SystemRoleDetailList struct {
	BaseListForm
	Items []*SystemRoleDetail
}

func (ul *SystemRoleDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.SystemRoleList{}
	return ul.objectlist
}

func (ul *SystemRoleDetailList) AsListData() []*SystemRoleDetail {
	us := ul.objectlist.(*orm.SystemRoleList)
	return Convert_SystemRole_SystemRoleDetail_arr(us.Items)
}

func (r *SystemRoleDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_SystemRoleDetail_SystemRole(r)
	}
	return r.object
}

func (u *SystemRoleDetail) Data() *SystemRoleDetail {
	return Convert_SystemRole_SystemRoleDetail(u.object.(*orm.SystemRole))
}

func Convert_SystemRoleDetail_SystemRole(f *SystemRoleDetail) *orm.SystemRole {
	r := &orm.SystemRole{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.RoleName = f.RoleName
	r.RoleCode = f.RoleCode
	r.Users = Convert_UserCommon_User_arr(f.Users)
	return r
}
func Convert_SystemRole_SystemRoleDetail(f *orm.SystemRole) *SystemRoleDetail {
	if f == nil {
		return nil
	}
	var r SystemRoleDetail
	r.ID = f.ID
	r.RoleName = f.RoleName
	r.RoleCode = f.RoleCode
	r.Users = Convert_User_UserCommon_arr(f.Users)
	return &r
}
func Convert_SystemRoleDetail_SystemRole_arr(arr []*SystemRoleDetail) []*orm.SystemRole {
	r := []*orm.SystemRole{}
	for _, u := range arr {
		r = append(r, Convert_SystemRoleDetail_SystemRole(u))
	}
	return r
}

func Convert_SystemRole_SystemRoleDetail_arr(arr []*orm.SystemRole) []*SystemRoleDetail {
	r := []*SystemRoleDetail{}
	for _, u := range arr {
		r = append(r, Convert_SystemRole_SystemRoleDetail(u))
	}
	return r
}

type SystemRoleCommonList struct {
	BaseListForm
	Items []*SystemRoleCommon
}

func (ul *SystemRoleCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.SystemRoleList{}
	return ul.objectlist
}

func (ul *SystemRoleCommonList) AsListData() []*SystemRoleCommon {
	us := ul.objectlist.(*orm.SystemRoleList)
	return Convert_SystemRole_SystemRoleCommon_arr(us.Items)
}

func (r *SystemRoleCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_SystemRoleCommon_SystemRole(r)
	}
	return r.object
}

func (u *SystemRoleCommon) Data() *SystemRoleCommon {
	return Convert_SystemRole_SystemRoleCommon(u.object.(*orm.SystemRole))
}

func Convert_SystemRoleCommon_SystemRole(f *SystemRoleCommon) *orm.SystemRole {
	r := &orm.SystemRole{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.RoleName = f.RoleName
	r.RoleCode = f.RoleCode
	return r
}
func Convert_SystemRole_SystemRoleCommon(f *orm.SystemRole) *SystemRoleCommon {
	if f == nil {
		return nil
	}
	var r SystemRoleCommon
	r.ID = f.ID
	r.RoleName = f.RoleName
	r.RoleCode = f.RoleCode
	return &r
}
func Convert_SystemRoleCommon_SystemRole_arr(arr []*SystemRoleCommon) []*orm.SystemRole {
	r := []*orm.SystemRole{}
	for _, u := range arr {
		r = append(r, Convert_SystemRoleCommon_SystemRole(u))
	}
	return r
}

func Convert_SystemRole_SystemRoleCommon_arr(arr []*orm.SystemRole) []*SystemRoleCommon {
	r := []*SystemRoleCommon{}
	for _, u := range arr {
		r = append(r, Convert_SystemRole_SystemRoleCommon(u))
	}
	return r
}

type RegistryDetailList struct {
	BaseListForm
	Items []*RegistryDetail
}

func (ul *RegistryDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.RegistryList{}
	return ul.objectlist
}

func (ul *RegistryDetailList) AsListData() []*RegistryDetail {
	us := ul.objectlist.(*orm.RegistryList)
	return Convert_Registry_RegistryDetail_arr(us.Items)
}

func (r *RegistryDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_RegistryDetail_Registry(r)
	}
	return r.object
}

func (u *RegistryDetail) Data() *RegistryDetail {
	return Convert_Registry_RegistryDetail(u.object.(*orm.Registry))
}

func Convert_RegistryDetail_Registry(f *RegistryDetail) *orm.Registry {
	r := &orm.Registry{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.IsDefault = f.IsDefault
	r.RegistryAddress = f.RegistryAddress
	r.Username = f.Username
	r.CreatorID = f.CreatorID
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.UpdateTime = f.UpdateTime
	r.ID = f.ID
	r.RegistryName = f.RegistryName
	r.Password = f.Password
	return r
}
func Convert_Registry_RegistryDetail(f *orm.Registry) *RegistryDetail {
	if f == nil {
		return nil
	}
	var r RegistryDetail
	r.RegistryAddress = f.RegistryAddress
	r.Username = f.Username
	r.CreatorID = f.CreatorID
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	r.IsDefault = f.IsDefault
	r.ID = f.ID
	r.RegistryName = f.RegistryName
	r.Password = f.Password
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.UpdateTime = f.UpdateTime
	return &r
}
func Convert_RegistryDetail_Registry_arr(arr []*RegistryDetail) []*orm.Registry {
	r := []*orm.Registry{}
	for _, u := range arr {
		r = append(r, Convert_RegistryDetail_Registry(u))
	}
	return r
}

func Convert_Registry_RegistryDetail_arr(arr []*orm.Registry) []*RegistryDetail {
	r := []*RegistryDetail{}
	for _, u := range arr {
		r = append(r, Convert_Registry_RegistryDetail(u))
	}
	return r
}

type RegistryCommonList struct {
	BaseListForm
	Items []*RegistryCommon
}

func (ul *RegistryCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.RegistryList{}
	return ul.objectlist
}

func (ul *RegistryCommonList) AsListData() []*RegistryCommon {
	us := ul.objectlist.(*orm.RegistryList)
	return Convert_Registry_RegistryCommon_arr(us.Items)
}

func (r *RegistryCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_RegistryCommon_Registry(r)
	}
	return r.object
}

func (u *RegistryCommon) Data() *RegistryCommon {
	return Convert_Registry_RegistryCommon(u.object.(*orm.Registry))
}

func Convert_RegistryCommon_Registry(f *RegistryCommon) *orm.Registry {
	r := &orm.Registry{}
	if f == nil {
		return nil
	}
	f.object = r
	r.RegistryName = f.RegistryName
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.ID = f.ID
	r.RegistryAddress = f.RegistryAddress
	r.UpdateTime = f.UpdateTime
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.IsDefault = f.IsDefault
	return r
}
func Convert_Registry_RegistryCommon(f *orm.Registry) *RegistryCommon {
	if f == nil {
		return nil
	}
	var r RegistryCommon
	r.ID = f.ID
	r.RegistryAddress = f.RegistryAddress
	r.UpdateTime = f.UpdateTime
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.IsDefault = f.IsDefault
	r.RegistryName = f.RegistryName
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	return &r
}
func Convert_RegistryCommon_Registry_arr(arr []*RegistryCommon) []*orm.Registry {
	r := []*orm.Registry{}
	for _, u := range arr {
		r = append(r, Convert_RegistryCommon_Registry(u))
	}
	return r
}

func Convert_Registry_RegistryCommon_arr(arr []*orm.Registry) []*RegistryCommon {
	r := []*RegistryCommon{}
	for _, u := range arr {
		r = append(r, Convert_Registry_RegistryCommon(u))
	}
	return r
}

type ProjectUserRelCommonList struct {
	BaseListForm
	Items []*ProjectUserRelCommon
}

func (ul *ProjectUserRelCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ProjectUserRelList{}
	return ul.objectlist
}

func (ul *ProjectUserRelCommonList) AsListData() []*ProjectUserRelCommon {
	us := ul.objectlist.(*orm.ProjectUserRelList)
	return Convert_ProjectUserRel_ProjectUserRelCommon_arr(us.Items)
}

func (r *ProjectUserRelCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ProjectUserRelCommon_ProjectUserRel(r)
	}
	return r.object
}

func (u *ProjectUserRelCommon) Data() *ProjectUserRelCommon {
	return Convert_ProjectUserRel_ProjectUserRelCommon(u.object.(*orm.ProjectUserRel))
}

func Convert_ProjectUserRelCommon_ProjectUserRel(f *ProjectUserRelCommon) *orm.ProjectUserRel {
	r := &orm.ProjectUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.UserID = f.UserID
	r.ProjectID = f.ProjectID
	r.Role = f.Role
	r.ID = f.ID
	r.User = Convert_UserCommon_User(f.User)
	return r
}
func Convert_ProjectUserRel_ProjectUserRelCommon(f *orm.ProjectUserRel) *ProjectUserRelCommon {
	if f == nil {
		return nil
	}
	var r ProjectUserRelCommon
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.UserID = f.UserID
	r.ProjectID = f.ProjectID
	r.Role = f.Role
	r.ID = f.ID
	r.User = Convert_User_UserCommon(f.User)
	return &r
}
func Convert_ProjectUserRelCommon_ProjectUserRel_arr(arr []*ProjectUserRelCommon) []*orm.ProjectUserRel {
	r := []*orm.ProjectUserRel{}
	for _, u := range arr {
		r = append(r, Convert_ProjectUserRelCommon_ProjectUserRel(u))
	}
	return r
}

func Convert_ProjectUserRel_ProjectUserRelCommon_arr(arr []*orm.ProjectUserRel) []*ProjectUserRelCommon {
	r := []*ProjectUserRelCommon{}
	for _, u := range arr {
		r = append(r, Convert_ProjectUserRel_ProjectUserRelCommon(u))
	}
	return r
}

type ProjectDetailList struct {
	BaseListForm
	Items []*ProjectDetail
}

func (ul *ProjectDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ProjectList{}
	return ul.objectlist
}

func (ul *ProjectDetailList) AsListData() []*ProjectDetail {
	us := ul.objectlist.(*orm.ProjectList)
	return Convert_Project_ProjectDetail_arr(us.Items)
}

func (r *ProjectDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ProjectDetail_Project(r)
	}
	return r.object
}

func (u *ProjectDetail) Data() *ProjectDetail {
	return Convert_Project_ProjectDetail(u.object.(*orm.Project))
}

func Convert_ProjectDetail_Project(f *ProjectDetail) *orm.Project {
	r := &orm.Project{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Applications = Convert_ApplicationCommon_Application_arr(f.Applications)
	r.Environments = Convert_EnvironmentCommon_Environment_arr(f.Environments)
	r.Users = Convert_UserCommon_User_arr(f.Users)
	r.ID = f.ID
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.TenantID = f.TenantID
	r.CreatedAt = f.CreatedAt
	r.ProjectName = f.ProjectName
	r.ResourceQuota = f.ResourceQuota
	return r
}
func Convert_Project_ProjectDetail(f *orm.Project) *ProjectDetail {
	if f == nil {
		return nil
	}
	var r ProjectDetail
	r.ResourceQuota = f.ResourceQuota
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.TenantID = f.TenantID
	r.CreatedAt = f.CreatedAt
	r.ProjectName = f.ProjectName
	r.Remark = f.Remark
	r.Applications = Convert_Application_ApplicationCommon_arr(f.Applications)
	r.Environments = Convert_Environment_EnvironmentCommon_arr(f.Environments)
	r.Users = Convert_User_UserCommon_arr(f.Users)
	r.ID = f.ID
	r.ProjectAlias = f.ProjectAlias
	return &r
}
func Convert_ProjectDetail_Project_arr(arr []*ProjectDetail) []*orm.Project {
	r := []*orm.Project{}
	for _, u := range arr {
		r = append(r, Convert_ProjectDetail_Project(u))
	}
	return r
}

func Convert_Project_ProjectDetail_arr(arr []*orm.Project) []*ProjectDetail {
	r := []*ProjectDetail{}
	for _, u := range arr {
		r = append(r, Convert_Project_ProjectDetail(u))
	}
	return r
}

type ProjectCommonList struct {
	BaseListForm
	Items []*ProjectCommon
}

func (ul *ProjectCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ProjectList{}
	return ul.objectlist
}

func (ul *ProjectCommonList) AsListData() []*ProjectCommon {
	us := ul.objectlist.(*orm.ProjectList)
	return Convert_Project_ProjectCommon_arr(us.Items)
}

func (r *ProjectCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ProjectCommon_Project(r)
	}
	return r.object
}

func (u *ProjectCommon) Data() *ProjectCommon {
	return Convert_Project_ProjectCommon(u.object.(*orm.Project))
}

func Convert_ProjectCommon_Project(f *ProjectCommon) *orm.Project {
	r := &orm.Project{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.ProjectName = f.ProjectName
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	r.ID = f.ID
	return r
}
func Convert_Project_ProjectCommon(f *orm.Project) *ProjectCommon {
	if f == nil {
		return nil
	}
	var r ProjectCommon
	r.ID = f.ID
	r.CreatedAt = f.CreatedAt
	r.ProjectName = f.ProjectName
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	return &r
}
func Convert_ProjectCommon_Project_arr(arr []*ProjectCommon) []*orm.Project {
	r := []*orm.Project{}
	for _, u := range arr {
		r = append(r, Convert_ProjectCommon_Project(u))
	}
	return r
}

func Convert_Project_ProjectCommon_arr(arr []*orm.Project) []*ProjectCommon {
	r := []*ProjectCommon{}
	for _, u := range arr {
		r = append(r, Convert_Project_ProjectCommon(u))
	}
	return r
}

type OpenAPPDetailList struct {
	BaseListForm
	Items []*OpenAPPDetail
}

func (ul *OpenAPPDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.OpenAPPList{}
	return ul.objectlist
}

func (ul *OpenAPPDetailList) AsListData() []*OpenAPPDetail {
	us := ul.objectlist.(*orm.OpenAPPList)
	return Convert_OpenAPP_OpenAPPDetail_arr(us.Items)
}

func (r *OpenAPPDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_OpenAPPDetail_OpenAPP(r)
	}
	return r.object
}

func (u *OpenAPPDetail) Data() *OpenAPPDetail {
	return Convert_OpenAPP_OpenAPPDetail(u.object.(*orm.OpenAPP))
}

func Convert_OpenAPPDetail_OpenAPP(f *OpenAPPDetail) *orm.OpenAPP {
	r := &orm.OpenAPP{}
	if f == nil {
		return nil
	}
	f.object = r
	r.PermScopes = f.PermScopes
	r.TenantScope = f.TenantScope
	r.RequestLimiter = f.RequestLimiter
	r.Name = f.Name
	r.ID = f.ID
	r.AppID = f.AppID
	r.AppSecret = f.AppSecret
	return r
}
func Convert_OpenAPP_OpenAPPDetail(f *orm.OpenAPP) *OpenAPPDetail {
	if f == nil {
		return nil
	}
	var r OpenAPPDetail
	r.PermScopes = f.PermScopes
	r.TenantScope = f.TenantScope
	r.RequestLimiter = f.RequestLimiter
	r.Name = f.Name
	r.ID = f.ID
	r.AppID = f.AppID
	r.AppSecret = f.AppSecret
	return &r
}
func Convert_OpenAPPDetail_OpenAPP_arr(arr []*OpenAPPDetail) []*orm.OpenAPP {
	r := []*orm.OpenAPP{}
	for _, u := range arr {
		r = append(r, Convert_OpenAPPDetail_OpenAPP(u))
	}
	return r
}

func Convert_OpenAPP_OpenAPPDetail_arr(arr []*orm.OpenAPP) []*OpenAPPDetail {
	r := []*OpenAPPDetail{}
	for _, u := range arr {
		r = append(r, Convert_OpenAPP_OpenAPPDetail(u))
	}
	return r
}

type OpenAPPCommonList struct {
	BaseListForm
	Items []*OpenAPPCommon
}

func (ul *OpenAPPCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.OpenAPPList{}
	return ul.objectlist
}

func (ul *OpenAPPCommonList) AsListData() []*OpenAPPCommon {
	us := ul.objectlist.(*orm.OpenAPPList)
	return Convert_OpenAPP_OpenAPPCommon_arr(us.Items)
}

func (r *OpenAPPCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_OpenAPPCommon_OpenAPP(r)
	}
	return r.object
}

func (u *OpenAPPCommon) Data() *OpenAPPCommon {
	return Convert_OpenAPP_OpenAPPCommon(u.object.(*orm.OpenAPP))
}

func Convert_OpenAPPCommon_OpenAPP(f *OpenAPPCommon) *orm.OpenAPP {
	r := &orm.OpenAPP{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Name = f.Name
	r.ID = f.ID
	r.AppID = f.AppID
	r.PermScopes = f.PermScopes
	r.TenantScope = f.TenantScope
	r.RequestLimiter = f.RequestLimiter
	return r
}
func Convert_OpenAPP_OpenAPPCommon(f *orm.OpenAPP) *OpenAPPCommon {
	if f == nil {
		return nil
	}
	var r OpenAPPCommon
	r.ID = f.ID
	r.AppID = f.AppID
	r.PermScopes = f.PermScopes
	r.TenantScope = f.TenantScope
	r.RequestLimiter = f.RequestLimiter
	r.Name = f.Name
	return &r
}
func Convert_OpenAPPCommon_OpenAPP_arr(arr []*OpenAPPCommon) []*orm.OpenAPP {
	r := []*orm.OpenAPP{}
	for _, u := range arr {
		r = append(r, Convert_OpenAPPCommon_OpenAPP(u))
	}
	return r
}

func Convert_OpenAPP_OpenAPPCommon_arr(arr []*orm.OpenAPP) []*OpenAPPCommon {
	r := []*OpenAPPCommon{}
	for _, u := range arr {
		r = append(r, Convert_OpenAPP_OpenAPPCommon(u))
	}
	return r
}

type MessageCommonList struct {
	BaseListForm
	Items []*MessageCommon
}

func (ul *MessageCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.MessageList{}
	return ul.objectlist
}

func (ul *MessageCommonList) AsListData() []*MessageCommon {
	us := ul.objectlist.(*orm.MessageList)
	return Convert_Message_MessageCommon_arr(us.Items)
}

func (r *MessageCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_MessageCommon_Message(r)
	}
	return r.object
}

func (u *MessageCommon) Data() *MessageCommon {
	return Convert_Message_MessageCommon(u.object.(*orm.Message))
}

func Convert_MessageCommon_Message(f *MessageCommon) *orm.Message {
	r := &orm.Message{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Title = f.Title
	r.Content = f.Content
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.MessageType = f.MessageType
	return r
}
func Convert_Message_MessageCommon(f *orm.Message) *MessageCommon {
	if f == nil {
		return nil
	}
	var r MessageCommon
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.MessageType = f.MessageType
	r.Title = f.Title
	r.Content = f.Content
	return &r
}
func Convert_MessageCommon_Message_arr(arr []*MessageCommon) []*orm.Message {
	r := []*orm.Message{}
	for _, u := range arr {
		r = append(r, Convert_MessageCommon_Message(u))
	}
	return r
}

func Convert_Message_MessageCommon_arr(arr []*orm.Message) []*MessageCommon {
	r := []*MessageCommon{}
	for _, u := range arr {
		r = append(r, Convert_Message_MessageCommon(u))
	}
	return r
}

type LogQuerySnapshotCommonList struct {
	BaseListForm
	Items []*LogQuerySnapshotCommon
}

func (ul *LogQuerySnapshotCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.LogQuerySnapshotList{}
	return ul.objectlist
}

func (ul *LogQuerySnapshotCommonList) AsListData() []*LogQuerySnapshotCommon {
	us := ul.objectlist.(*orm.LogQuerySnapshotList)
	return Convert_LogQuerySnapshot_LogQuerySnapshotCommon_arr(us.Items)
}

func (r *LogQuerySnapshotCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_LogQuerySnapshotCommon_LogQuerySnapshot(r)
	}
	return r.object
}

func (u *LogQuerySnapshotCommon) Data() *LogQuerySnapshotCommon {
	return Convert_LogQuerySnapshot_LogQuerySnapshotCommon(u.object.(*orm.LogQuerySnapshot))
}

func Convert_LogQuerySnapshotCommon_LogQuerySnapshot(f *LogQuerySnapshotCommon) *orm.LogQuerySnapshot {
	r := &orm.LogQuerySnapshot{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.SourceFile = f.SourceFile
	r.SnapshotCount = f.SnapshotCount
	r.DownloadURL = f.DownloadURL
	r.StartTime = f.StartTime
	r.EndTime = f.EndTime
	r.CreateAt = f.CreateAt
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.SnapshotName = f.SnapshotName
	return r
}
func Convert_LogQuerySnapshot_LogQuerySnapshotCommon(f *orm.LogQuerySnapshot) *LogQuerySnapshotCommon {
	if f == nil {
		return nil
	}
	var r LogQuerySnapshotCommon
	r.CreateAt = f.CreateAt
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.SnapshotName = f.SnapshotName
	r.SourceFile = f.SourceFile
	r.SnapshotCount = f.SnapshotCount
	r.DownloadURL = f.DownloadURL
	r.StartTime = f.StartTime
	r.EndTime = f.EndTime
	r.ID = f.ID
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.CreatorID = f.CreatorID
	return &r
}
func Convert_LogQuerySnapshotCommon_LogQuerySnapshot_arr(arr []*LogQuerySnapshotCommon) []*orm.LogQuerySnapshot {
	r := []*orm.LogQuerySnapshot{}
	for _, u := range arr {
		r = append(r, Convert_LogQuerySnapshotCommon_LogQuerySnapshot(u))
	}
	return r
}

func Convert_LogQuerySnapshot_LogQuerySnapshotCommon_arr(arr []*orm.LogQuerySnapshot) []*LogQuerySnapshotCommon {
	r := []*LogQuerySnapshotCommon{}
	for _, u := range arr {
		r = append(r, Convert_LogQuerySnapshot_LogQuerySnapshotCommon(u))
	}
	return r
}

type LogQueryHistoryCommonList struct {
	BaseListForm
	Items []*LogQueryHistoryCommon
}

func (ul *LogQueryHistoryCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.LogQueryHistoryList{}
	return ul.objectlist
}

func (ul *LogQueryHistoryCommonList) AsListData() []*LogQueryHistoryCommon {
	us := ul.objectlist.(*orm.LogQueryHistoryList)
	return Convert_LogQueryHistory_LogQueryHistoryCommon_arr(us.Items)
}

func (r *LogQueryHistoryCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_LogQueryHistoryCommon_LogQueryHistory(r)
	}
	return r.object
}

func (u *LogQueryHistoryCommon) Data() *LogQueryHistoryCommon {
	return Convert_LogQueryHistory_LogQueryHistoryCommon(u.object.(*orm.LogQueryHistory))
}

func Convert_LogQueryHistoryCommon_LogQueryHistory(f *LogQueryHistoryCommon) *orm.LogQueryHistory {
	r := &orm.LogQueryHistory{}
	if f == nil {
		return nil
	}
	f.object = r
	r.FilterJSON = f.FilterJSON
	r.LogQL = f.LogQL
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.ID = f.ID
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.LabelJSON = f.LabelJSON
	r.CreateAt = f.CreateAt
	r.CreatorID = f.CreatorID
	return r
}
func Convert_LogQueryHistory_LogQueryHistoryCommon(f *orm.LogQueryHistory) *LogQueryHistoryCommon {
	if f == nil {
		return nil
	}
	var r LogQueryHistoryCommon
	r.ID = f.ID
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.LabelJSON = f.LabelJSON
	r.CreateAt = f.CreateAt
	r.CreatorID = f.CreatorID
	r.FilterJSON = f.FilterJSON
	r.LogQL = f.LogQL
	r.Creator = Convert_User_UserCommon(f.Creator)
	return &r
}
func Convert_LogQueryHistoryCommon_LogQueryHistory_arr(arr []*LogQueryHistoryCommon) []*orm.LogQueryHistory {
	r := []*orm.LogQueryHistory{}
	for _, u := range arr {
		r = append(r, Convert_LogQueryHistoryCommon_LogQueryHistory(u))
	}
	return r
}

func Convert_LogQueryHistory_LogQueryHistoryCommon_arr(arr []*orm.LogQueryHistory) []*LogQueryHistoryCommon {
	r := []*LogQueryHistoryCommon{}
	for _, u := range arr {
		r = append(r, Convert_LogQueryHistory_LogQueryHistoryCommon(u))
	}
	return r
}

type EnvironmentUserRelCommonList struct {
	BaseListForm
	Items []*EnvironmentUserRelCommon
}

func (ul *EnvironmentUserRelCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.EnvironmentUserRelList{}
	return ul.objectlist
}

func (ul *EnvironmentUserRelCommonList) AsListData() []*EnvironmentUserRelCommon {
	us := ul.objectlist.(*orm.EnvironmentUserRelList)
	return Convert_EnvironmentUserRel_EnvironmentUserRelCommon_arr(us.Items)
}

func (r *EnvironmentUserRelCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentUserRelCommon_EnvironmentUserRel(r)
	}
	return r.object
}

func (u *EnvironmentUserRelCommon) Data() *EnvironmentUserRelCommon {
	return Convert_EnvironmentUserRel_EnvironmentUserRelCommon(u.object.(*orm.EnvironmentUserRel))
}

func Convert_EnvironmentUserRelCommon_EnvironmentUserRel(f *EnvironmentUserRelCommon) *orm.EnvironmentUserRel {
	r := &orm.EnvironmentUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Environment = Convert_EnvironmentCommon_Environment(f.Environment)
	r.UserID = f.UserID
	r.EnvironmentID = f.EnvironmentID
	r.Role = f.Role
	r.ID = f.ID
	r.User = Convert_UserCommon_User(f.User)
	return r
}
func Convert_EnvironmentUserRel_EnvironmentUserRelCommon(f *orm.EnvironmentUserRel) *EnvironmentUserRelCommon {
	if f == nil {
		return nil
	}
	var r EnvironmentUserRelCommon
	r.User = Convert_User_UserCommon(f.User)
	r.Environment = Convert_Environment_EnvironmentCommon(f.Environment)
	r.UserID = f.UserID
	r.EnvironmentID = f.EnvironmentID
	r.Role = f.Role
	r.ID = f.ID
	return &r
}
func Convert_EnvironmentUserRelCommon_EnvironmentUserRel_arr(arr []*EnvironmentUserRelCommon) []*orm.EnvironmentUserRel {
	r := []*orm.EnvironmentUserRel{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentUserRelCommon_EnvironmentUserRel(u))
	}
	return r
}

func Convert_EnvironmentUserRel_EnvironmentUserRelCommon_arr(arr []*orm.EnvironmentUserRel) []*EnvironmentUserRelCommon {
	r := []*EnvironmentUserRelCommon{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentUserRel_EnvironmentUserRelCommon(u))
	}
	return r
}

type EnvironmentResourceCommonList struct {
	BaseListForm
	Items []*EnvironmentResourceCommon
}

func (ul *EnvironmentResourceCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.EnvironmentResourceList{}
	return ul.objectlist
}

func (ul *EnvironmentResourceCommonList) AsListData() []*EnvironmentResourceCommon {
	us := ul.objectlist.(*orm.EnvironmentResourceList)
	return Convert_EnvironmentResource_EnvironmentResourceCommon_arr(us.Items)
}

func (r *EnvironmentResourceCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentResourceCommon_EnvironmentResource(r)
	}
	return r.object
}

func (u *EnvironmentResourceCommon) Data() *EnvironmentResourceCommon {
	return Convert_EnvironmentResource_EnvironmentResourceCommon(u.object.(*orm.EnvironmentResource))
}

func Convert_EnvironmentResourceCommon_EnvironmentResource(f *EnvironmentResourceCommon) *orm.EnvironmentResource {
	r := &orm.EnvironmentResource{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ProjectName = f.ProjectName
	r.EnvironmentName = f.EnvironmentName
	r.MaxCPUUsageCore = f.MaxCPUUsageCore
	r.MinPVCUsageByte = f.MinPVCUsageByte
	r.ClusterName = f.ClusterName
	r.TenantName = f.TenantName
	r.MinCPUUsageCore = f.MinCPUUsageCore
	r.AvgMemoryUsageByte = f.AvgMemoryUsageByte
	r.NetworkReceiveByte = f.NetworkReceiveByte
	r.NetworkSendByte = f.NetworkSendByte
	r.AvgPVCUsageByte = f.AvgPVCUsageByte
	r.ID = f.ID
	r.CreatedAt = f.CreatedAt
	r.MaxMemoryUsageByte = f.MaxMemoryUsageByte
	r.AvgCPUUsageCore = f.AvgCPUUsageCore
	r.MinMemoryUsageByte = f.MinMemoryUsageByte
	r.MaxPVCUsageByte = f.MaxPVCUsageByte
	return r
}
func Convert_EnvironmentResource_EnvironmentResourceCommon(f *orm.EnvironmentResource) *EnvironmentResourceCommon {
	if f == nil {
		return nil
	}
	var r EnvironmentResourceCommon
	r.AvgCPUUsageCore = f.AvgCPUUsageCore
	r.NetworkReceiveByte = f.NetworkReceiveByte
	r.NetworkSendByte = f.NetworkSendByte
	r.AvgPVCUsageByte = f.AvgPVCUsageByte
	r.ID = f.ID
	r.CreatedAt = f.CreatedAt
	r.MaxMemoryUsageByte = f.MaxMemoryUsageByte
	r.MinMemoryUsageByte = f.MinMemoryUsageByte
	r.MaxPVCUsageByte = f.MaxPVCUsageByte
	r.ProjectName = f.ProjectName
	r.EnvironmentName = f.EnvironmentName
	r.MaxCPUUsageCore = f.MaxCPUUsageCore
	r.AvgMemoryUsageByte = f.AvgMemoryUsageByte
	r.MinPVCUsageByte = f.MinPVCUsageByte
	r.ClusterName = f.ClusterName
	r.TenantName = f.TenantName
	r.MinCPUUsageCore = f.MinCPUUsageCore
	return &r
}
func Convert_EnvironmentResourceCommon_EnvironmentResource_arr(arr []*EnvironmentResourceCommon) []*orm.EnvironmentResource {
	r := []*orm.EnvironmentResource{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentResourceCommon_EnvironmentResource(u))
	}
	return r
}

func Convert_EnvironmentResource_EnvironmentResourceCommon_arr(arr []*orm.EnvironmentResource) []*EnvironmentResourceCommon {
	r := []*EnvironmentResourceCommon{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentResource_EnvironmentResourceCommon(u))
	}
	return r
}

type EnvironmentDetailList struct {
	BaseListForm
	Items []*EnvironmentDetail
}

func (ul *EnvironmentDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.EnvironmentList{}
	return ul.objectlist
}

func (ul *EnvironmentDetailList) AsListData() []*EnvironmentDetail {
	us := ul.objectlist.(*orm.EnvironmentList)
	return Convert_Environment_EnvironmentDetail_arr(us.Items)
}

func (r *EnvironmentDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentDetail_Environment(r)
	}
	return r.object
}

func (u *EnvironmentDetail) Data() *EnvironmentDetail {
	return Convert_Environment_EnvironmentDetail(u.object.(*orm.Environment))
}

func Convert_EnvironmentDetail_Environment(f *EnvironmentDetail) *orm.Environment {
	r := &orm.Environment{}
	if f == nil {
		return nil
	}
	f.object = r
	r.DeletePolicy = f.DeletePolicy
	r.ClusterID = f.ClusterID
	r.VirtualSpaceID = f.VirtualSpaceID
	r.CreatorID = f.CreatorID
	r.VirtualSpace = Convert_VirtualSpaceCommon_VirtualSpace(f.VirtualSpace)
	r.EnvironmentName = f.EnvironmentName
	r.Remark = f.Remark
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.Namespace = f.Namespace
	r.MetaType = f.MetaType
	r.LimitRange = f.LimitRange
	r.ProjectID = f.ProjectID
	r.Users = Convert_UserCommon_User_arr(f.Users)
	r.ID = f.ID
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.ResourceQuota = f.ResourceQuota
	r.Applications = Convert_ApplicationCommon_Application_arr(f.Applications)
	return r
}
func Convert_Environment_EnvironmentDetail(f *orm.Environment) *EnvironmentDetail {
	if f == nil {
		return nil
	}
	var r EnvironmentDetail
	r.Users = Convert_User_UserCommon_arr(f.Users)
	r.ID = f.ID
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.ResourceQuota = f.ResourceQuota
	r.Applications = Convert_Application_ApplicationCommon_arr(f.Applications)
	r.DeletePolicy = f.DeletePolicy
	r.ClusterID = f.ClusterID
	r.VirtualSpaceID = f.VirtualSpaceID
	r.CreatorID = f.CreatorID
	r.VirtualSpace = Convert_VirtualSpace_VirtualSpaceCommon(f.VirtualSpace)
	r.EnvironmentName = f.EnvironmentName
	r.Remark = f.Remark
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.Namespace = f.Namespace
	r.MetaType = f.MetaType
	r.LimitRange = f.LimitRange
	r.ProjectID = f.ProjectID
	return &r
}
func Convert_EnvironmentDetail_Environment_arr(arr []*EnvironmentDetail) []*orm.Environment {
	r := []*orm.Environment{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentDetail_Environment(u))
	}
	return r
}

func Convert_Environment_EnvironmentDetail_arr(arr []*orm.Environment) []*EnvironmentDetail {
	r := []*EnvironmentDetail{}
	for _, u := range arr {
		r = append(r, Convert_Environment_EnvironmentDetail(u))
	}
	return r
}

type EnvironmentCommonList struct {
	BaseListForm
	Items []*EnvironmentCommon
}

func (ul *EnvironmentCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.EnvironmentList{}
	return ul.objectlist
}

func (ul *EnvironmentCommonList) AsListData() []*EnvironmentCommon {
	us := ul.objectlist.(*orm.EnvironmentList)
	return Convert_Environment_EnvironmentCommon_arr(us.Items)
}

func (r *EnvironmentCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentCommon_Environment(r)
	}
	return r.object
}

func (u *EnvironmentCommon) Data() *EnvironmentCommon {
	return Convert_Environment_EnvironmentCommon(u.object.(*orm.Environment))
}

func Convert_EnvironmentCommon_Environment(f *EnvironmentCommon) *orm.Environment {
	r := &orm.Environment{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Namespace = f.Namespace
	r.Remark = f.Remark
	r.MetaType = f.MetaType
	r.DeletePolicy = f.DeletePolicy
	r.ID = f.ID
	r.EnvironmentName = f.EnvironmentName
	return r
}
func Convert_Environment_EnvironmentCommon(f *orm.Environment) *EnvironmentCommon {
	if f == nil {
		return nil
	}
	var r EnvironmentCommon
	r.MetaType = f.MetaType
	r.DeletePolicy = f.DeletePolicy
	r.ID = f.ID
	r.EnvironmentName = f.EnvironmentName
	r.Namespace = f.Namespace
	r.Remark = f.Remark
	return &r
}
func Convert_EnvironmentCommon_Environment_arr(arr []*EnvironmentCommon) []*orm.Environment {
	r := []*orm.Environment{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentCommon_Environment(u))
	}
	return r
}

func Convert_Environment_EnvironmentCommon_arr(arr []*orm.Environment) []*EnvironmentCommon {
	r := []*EnvironmentCommon{}
	for _, u := range arr {
		r = append(r, Convert_Environment_EnvironmentCommon(u))
	}
	return r
}

type ContainerCommonList struct {
	BaseListForm
	Items []*ContainerCommon
}

func (ul *ContainerCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ContainerList{}
	return ul.objectlist
}

func (ul *ContainerCommonList) AsListData() []*ContainerCommon {
	us := ul.objectlist.(*orm.ContainerList)
	return Convert_Container_ContainerCommon_arr(us.Items)
}

func (r *ContainerCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ContainerCommon_Container(r)
	}
	return r.object
}

func (u *ContainerCommon) Data() *ContainerCommon {
	return Convert_Container_ContainerCommon(u.object.(*orm.Container))
}

func Convert_ContainerCommon_Container(f *ContainerCommon) *orm.Container {
	r := &orm.Container{}
	if f == nil {
		return nil
	}
	f.object = r
	r.PodName = f.PodName
	r.CPULimitCore = f.CPULimitCore
	r.MemoryLimitBytes = f.MemoryLimitBytes
	r.CPUPercent = f.CPUPercent
	r.MemoryUsageBytes = f.MemoryUsageBytes
	r.MemoryPercent = f.MemoryPercent
	r.ID = f.ID
	r.Name = f.Name
	r.WorkloadID = f.WorkloadID
	r.CPUUsageCore = f.CPUUsageCore
	r.Workload = Convert_WorkloadCommon_Workload(f.Workload)
	return r
}
func Convert_Container_ContainerCommon(f *orm.Container) *ContainerCommon {
	if f == nil {
		return nil
	}
	var r ContainerCommon
	r.CPUUsageCore = f.CPUUsageCore
	r.Workload = Convert_Workload_WorkloadCommon(f.Workload)
	r.CPUPercent = f.CPUPercent
	r.MemoryUsageBytes = f.MemoryUsageBytes
	r.MemoryPercent = f.MemoryPercent
	r.ID = f.ID
	r.Name = f.Name
	r.PodName = f.PodName
	r.CPULimitCore = f.CPULimitCore
	r.MemoryLimitBytes = f.MemoryLimitBytes
	r.WorkloadID = f.WorkloadID
	return &r
}
func Convert_ContainerCommon_Container_arr(arr []*ContainerCommon) []*orm.Container {
	r := []*orm.Container{}
	for _, u := range arr {
		r = append(r, Convert_ContainerCommon_Container(u))
	}
	return r
}

func Convert_Container_ContainerCommon_arr(arr []*orm.Container) []*ContainerCommon {
	r := []*ContainerCommon{}
	for _, u := range arr {
		r = append(r, Convert_Container_ContainerCommon(u))
	}
	return r
}

type ClusterDetailList struct {
	BaseListForm
	Items []*ClusterDetail
}

func (ul *ClusterDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ClusterList{}
	return ul.objectlist
}

func (ul *ClusterDetailList) AsListData() []*ClusterDetail {
	us := ul.objectlist.(*orm.ClusterList)
	return Convert_Cluster_ClusterDetail_arr(us.Items)
}

func (r *ClusterDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ClusterDetail_Cluster(r)
	}
	return r.object
}

func (u *ClusterDetail) Data() *ClusterDetail {
	return Convert_Cluster_ClusterDetail(u.object.(*orm.Cluster))
}

func Convert_ClusterDetail_Cluster(f *ClusterDetail) *orm.Cluster {
	r := &orm.Cluster{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Mode = f.Mode
	r.Environments = Convert_EnvironmentCommon_Environment_arr(f.Environments)
	r.Primary = f.Primary
	r.APIServer = f.APIServer
	r.AgentAddr = f.AgentAddr
	r.AgentCert = f.AgentCert
	r.AgentKey = f.AgentKey
	r.Runtime = f.Runtime
	r.OversoldConfig = f.OversoldConfig
	r.ClusterResourceQuota = f.ClusterResourceQuota
	r.ClusterName = f.ClusterName
	r.KubeConfig = f.KubeConfig
	r.Version = f.Version
	r.AgentCA = f.AgentCA
	return r
}
func Convert_Cluster_ClusterDetail(f *orm.Cluster) *ClusterDetail {
	if f == nil {
		return nil
	}
	var r ClusterDetail
	r.APIServer = f.APIServer
	r.AgentAddr = f.AgentAddr
	r.AgentCert = f.AgentCert
	r.OversoldConfig = f.OversoldConfig
	r.ClusterResourceQuota = f.ClusterResourceQuota
	r.ClusterName = f.ClusterName
	r.KubeConfig = f.KubeConfig
	r.Version = f.Version
	r.AgentCA = f.AgentCA
	r.AgentKey = f.AgentKey
	r.Runtime = f.Runtime
	r.ID = f.ID
	r.Mode = f.Mode
	r.Environments = Convert_Environment_EnvironmentCommon_arr(f.Environments)
	r.Primary = f.Primary
	return &r
}
func Convert_ClusterDetail_Cluster_arr(arr []*ClusterDetail) []*orm.Cluster {
	r := []*orm.Cluster{}
	for _, u := range arr {
		r = append(r, Convert_ClusterDetail_Cluster(u))
	}
	return r
}

func Convert_Cluster_ClusterDetail_arr(arr []*orm.Cluster) []*ClusterDetail {
	r := []*ClusterDetail{}
	for _, u := range arr {
		r = append(r, Convert_Cluster_ClusterDetail(u))
	}
	return r
}

type ClusterCommonList struct {
	BaseListForm
	Items []*ClusterCommon
}

func (ul *ClusterCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ClusterList{}
	return ul.objectlist
}

func (ul *ClusterCommonList) AsListData() []*ClusterCommon {
	us := ul.objectlist.(*orm.ClusterList)
	return Convert_Cluster_ClusterCommon_arr(us.Items)
}

func (r *ClusterCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ClusterCommon_Cluster(r)
	}
	return r.object
}

func (u *ClusterCommon) Data() *ClusterCommon {
	return Convert_Cluster_ClusterCommon(u.object.(*orm.Cluster))
}

func Convert_ClusterCommon_Cluster(f *ClusterCommon) *orm.Cluster {
	r := &orm.Cluster{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.ClusterName = f.ClusterName
	r.Primary = f.Primary
	r.APIServer = f.APIServer
	r.Version = f.Version
	r.Runtime = f.Runtime
	return r
}
func Convert_Cluster_ClusterCommon(f *orm.Cluster) *ClusterCommon {
	if f == nil {
		return nil
	}
	var r ClusterCommon
	r.ID = f.ID
	r.ClusterName = f.ClusterName
	r.Primary = f.Primary
	r.APIServer = f.APIServer
	r.Version = f.Version
	r.Runtime = f.Runtime
	return &r
}
func Convert_ClusterCommon_Cluster_arr(arr []*ClusterCommon) []*orm.Cluster {
	r := []*orm.Cluster{}
	for _, u := range arr {
		r = append(r, Convert_ClusterCommon_Cluster(u))
	}
	return r
}

func Convert_Cluster_ClusterCommon_arr(arr []*orm.Cluster) []*ClusterCommon {
	r := []*ClusterCommon{}
	for _, u := range arr {
		r = append(r, Convert_Cluster_ClusterCommon(u))
	}
	return r
}

type ChartRepoCommonList struct {
	BaseListForm
	Items []*ChartRepoCommon
}

func (ul *ChartRepoCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ChartRepoList{}
	return ul.objectlist
}

func (ul *ChartRepoCommonList) AsListData() []*ChartRepoCommon {
	us := ul.objectlist.(*orm.ChartRepoList)
	return Convert_ChartRepo_ChartRepoCommon_arr(us.Items)
}

func (r *ChartRepoCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ChartRepoCommon_ChartRepo(r)
	}
	return r.object
}

func (u *ChartRepoCommon) Data() *ChartRepoCommon {
	return Convert_ChartRepo_ChartRepoCommon(u.object.(*orm.ChartRepo))
}

func Convert_ChartRepoCommon_ChartRepo(f *ChartRepoCommon) *orm.ChartRepo {
	r := &orm.ChartRepo{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ChartRepoName = f.ChartRepoName
	r.URL = f.URL
	r.LastSync = f.LastSync
	r.SyncStatus = f.SyncStatus
	r.SyncMessage = f.SyncMessage
	r.ID = f.ID
	return r
}
func Convert_ChartRepo_ChartRepoCommon(f *orm.ChartRepo) *ChartRepoCommon {
	if f == nil {
		return nil
	}
	var r ChartRepoCommon
	r.ID = f.ID
	r.ChartRepoName = f.ChartRepoName
	r.URL = f.URL
	r.LastSync = f.LastSync
	r.SyncStatus = f.SyncStatus
	r.SyncMessage = f.SyncMessage
	return &r
}
func Convert_ChartRepoCommon_ChartRepo_arr(arr []*ChartRepoCommon) []*orm.ChartRepo {
	r := []*orm.ChartRepo{}
	for _, u := range arr {
		r = append(r, Convert_ChartRepoCommon_ChartRepo(u))
	}
	return r
}

func Convert_ChartRepo_ChartRepoCommon_arr(arr []*orm.ChartRepo) []*ChartRepoCommon {
	r := []*ChartRepoCommon{}
	for _, u := range arr {
		r = append(r, Convert_ChartRepo_ChartRepoCommon(u))
	}
	return r
}

type AuthSourceCommonList struct {
	BaseListForm
	Items []*AuthSourceCommon
}

func (ul *AuthSourceCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.AuthSourceList{}
	return ul.objectlist
}

func (ul *AuthSourceCommonList) AsListData() []*AuthSourceCommon {
	us := ul.objectlist.(*orm.AuthSourceList)
	return Convert_AuthSource_AuthSourceCommon_arr(us.Items)
}

func (r *AuthSourceCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_AuthSourceCommon_AuthSource(r)
	}
	return r.object
}

func (u *AuthSourceCommon) Data() *AuthSourceCommon {
	return Convert_AuthSource_AuthSourceCommon(u.object.(*orm.AuthSource))
}

func Convert_AuthSourceCommon_AuthSource(f *AuthSourceCommon) *orm.AuthSource {
	r := &orm.AuthSource{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Name = f.Name
	r.Kind = f.Kind
	r.Config = f.Config
	r.Enabled = f.Enabled
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	return r
}
func Convert_AuthSource_AuthSourceCommon(f *orm.AuthSource) *AuthSourceCommon {
	if f == nil {
		return nil
	}
	var r AuthSourceCommon
	r.Config = f.Config
	r.Enabled = f.Enabled
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	r.ID = f.ID
	r.Name = f.Name
	r.Kind = f.Kind
	return &r
}
func Convert_AuthSourceCommon_AuthSource_arr(arr []*AuthSourceCommon) []*orm.AuthSource {
	r := []*orm.AuthSource{}
	for _, u := range arr {
		r = append(r, Convert_AuthSourceCommon_AuthSource(u))
	}
	return r
}

func Convert_AuthSource_AuthSourceCommon_arr(arr []*orm.AuthSource) []*AuthSourceCommon {
	r := []*AuthSourceCommon{}
	for _, u := range arr {
		r = append(r, Convert_AuthSource_AuthSourceCommon(u))
	}
	return r
}

type AuditLogCommonList struct {
	BaseListForm
	Items []*AuditLogCommon
}

func (ul *AuditLogCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.AuditLogList{}
	return ul.objectlist
}

func (ul *AuditLogCommonList) AsListData() []*AuditLogCommon {
	us := ul.objectlist.(*orm.AuditLogList)
	return Convert_AuditLog_AuditLogCommon_arr(us.Items)
}

func (r *AuditLogCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_AuditLogCommon_AuditLog(r)
	}
	return r.object
}

func (u *AuditLogCommon) Data() *AuditLogCommon {
	return Convert_AuditLog_AuditLogCommon(u.object.(*orm.AuditLog))
}

func Convert_AuditLogCommon_AuditLog(f *AuditLogCommon) *orm.AuditLog {
	r := &orm.AuditLog{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.DeletedAt = f.DeletedAt
	r.Username = f.Username
	r.Module = f.Module
	r.Name = f.Name
	r.RawData = f.RawData
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	r.Tenant = f.Tenant
	r.Action = f.Action
	r.Success = f.Success
	r.ClientIP = f.ClientIP
	r.Labels = f.Labels
	return r
}
func Convert_AuditLog_AuditLogCommon(f *orm.AuditLog) *AuditLogCommon {
	if f == nil {
		return nil
	}
	var r AuditLogCommon
	r.Username = f.Username
	r.Module = f.Module
	r.Name = f.Name
	r.RawData = f.RawData
	r.ID = f.ID
	r.DeletedAt = f.DeletedAt
	r.Tenant = f.Tenant
	r.Action = f.Action
	r.Success = f.Success
	r.ClientIP = f.ClientIP
	r.Labels = f.Labels
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	return &r
}
func Convert_AuditLogCommon_AuditLog_arr(arr []*AuditLogCommon) []*orm.AuditLog {
	r := []*orm.AuditLog{}
	for _, u := range arr {
		r = append(r, Convert_AuditLogCommon_AuditLog(u))
	}
	return r
}

func Convert_AuditLog_AuditLogCommon_arr(arr []*orm.AuditLog) []*AuditLogCommon {
	r := []*AuditLogCommon{}
	for _, u := range arr {
		r = append(r, Convert_AuditLog_AuditLogCommon(u))
	}
	return r
}

type ApplicationDetailList struct {
	BaseListForm
	Items []*ApplicationDetail
}

func (ul *ApplicationDetailList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ApplicationList{}
	return ul.objectlist
}

func (ul *ApplicationDetailList) AsListData() []*ApplicationDetail {
	us := ul.objectlist.(*orm.ApplicationList)
	return Convert_Application_ApplicationDetail_arr(us.Items)
}

func (r *ApplicationDetail) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ApplicationDetail_Application(r)
	}
	return r.object
}

func (u *ApplicationDetail) Data() *ApplicationDetail {
	return Convert_Application_ApplicationDetail(u.object.(*orm.Application))
}

func Convert_ApplicationDetail_Application(f *ApplicationDetail) *orm.Application {
	r := &orm.Application{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.Images = f.Images
	r.ID = f.ID
	r.Manifest = f.Manifest
	r.Kind = f.Kind
	r.Remark = f.Remark
	r.Creator = f.Creator
	r.ApplicationName = f.ApplicationName
	r.CreatedAt = f.CreatedAt
	r.EnvironmentID = f.EnvironmentID
	r.Enabled = f.Enabled
	r.Labels = f.Labels
	r.UpdatedAt = f.UpdatedAt
	r.Environment = Convert_EnvironmentCommon_Environment(f.Environment)
	r.ProjectID = f.ProjectID
	return r
}
func Convert_Application_ApplicationDetail(f *orm.Application) *ApplicationDetail {
	if f == nil {
		return nil
	}
	var r ApplicationDetail
	r.ID = f.ID
	r.Manifest = f.Manifest
	r.Kind = f.Kind
	r.Images = f.Images
	r.ApplicationName = f.ApplicationName
	r.CreatedAt = f.CreatedAt
	r.EnvironmentID = f.EnvironmentID
	r.Remark = f.Remark
	r.Creator = f.Creator
	r.UpdatedAt = f.UpdatedAt
	r.Environment = Convert_Environment_EnvironmentCommon(f.Environment)
	r.ProjectID = f.ProjectID
	r.Enabled = f.Enabled
	r.Labels = f.Labels
	r.Project = Convert_Project_ProjectCommon(f.Project)
	return &r
}
func Convert_ApplicationDetail_Application_arr(arr []*ApplicationDetail) []*orm.Application {
	r := []*orm.Application{}
	for _, u := range arr {
		r = append(r, Convert_ApplicationDetail_Application(u))
	}
	return r
}

func Convert_Application_ApplicationDetail_arr(arr []*orm.Application) []*ApplicationDetail {
	r := []*ApplicationDetail{}
	for _, u := range arr {
		r = append(r, Convert_Application_ApplicationDetail(u))
	}
	return r
}

type ApplicationCommonList struct {
	BaseListForm
	Items []*ApplicationCommon
}

func (ul *ApplicationCommonList) AsListObject() client.ObjectListIfe {
	ul.objectlist = &orm.ApplicationList{}
	return ul.objectlist
}

func (ul *ApplicationCommonList) AsListData() []*ApplicationCommon {
	us := ul.objectlist.(*orm.ApplicationList)
	return Convert_Application_ApplicationCommon_arr(us.Items)
}

func (r *ApplicationCommon) AsObject() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ApplicationCommon_Application(r)
	}
	return r.object
}

func (u *ApplicationCommon) Data() *ApplicationCommon {
	return Convert_Application_ApplicationCommon(u.object.(*orm.Application))
}

func Convert_ApplicationCommon_Application(f *ApplicationCommon) *orm.Application {
	r := &orm.Application{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.ApplicationName = f.ApplicationName
	r.CreatedAt = f.CreatedAt
	r.UpdatedAt = f.UpdatedAt
	return r
}
func Convert_Application_ApplicationCommon(f *orm.Application) *ApplicationCommon {
	if f == nil {
		return nil
	}
	var r ApplicationCommon
	r.UpdatedAt = f.UpdatedAt
	r.ID = f.ID
	r.ApplicationName = f.ApplicationName
	r.CreatedAt = f.CreatedAt
	return &r
}
func Convert_ApplicationCommon_Application_arr(arr []*ApplicationCommon) []*orm.Application {
	r := []*orm.Application{}
	for _, u := range arr {
		r = append(r, Convert_ApplicationCommon_Application(u))
	}
	return r
}

func Convert_Application_ApplicationCommon_arr(arr []*orm.Application) []*ApplicationCommon {
	r := []*ApplicationCommon{}
	for _, u := range arr {
		r = append(r, Convert_Application_ApplicationCommon(u))
	}
	return r
}
