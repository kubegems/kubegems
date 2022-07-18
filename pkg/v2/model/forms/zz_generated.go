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

package forms

import (
	"kubegems.io/kubegems/pkg/v2/model/client"
	"kubegems.io/kubegems/pkg/v2/model/orm"
)

type WorkloadCommonList struct {
	BaseListForm
	Items []*WorkloadCommon
}

func (ul *WorkloadCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.WorkloadList{}
	return ul.objectlist
}

func (ul *WorkloadCommonList) Data() []*WorkloadCommon {
	if ul.data != nil {
		return ul.data.([]*WorkloadCommon)
	}
	us := ul.objectlist.(*orm.WorkloadList)
	tmp := Convert_Workload_WorkloadCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *WorkloadCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *WorkloadCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_WorkloadCommon_Workload(r)
	}
	return r.object
}

func (u *WorkloadCommon) Data() *WorkloadCommon {
	if u.data != nil {
		return u.data.(*WorkloadCommon)
	}
	tmp := Convert_Workload_WorkloadCommon(u.object.(*orm.Workload))
	u.data = tmp
	return tmp
}

func (u *WorkloadCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_WorkloadCommon_Workload(f *WorkloadCommon) *orm.Workload {
	r := &orm.Workload{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CPULimitStdvar = f.CPULimitStdvar
	r.Cluster = f.Cluster
	r.Containers = Convert_ContainerCommon_Container_slice(f.Containers)
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.MemoryLimitStdvar = f.MemoryLimitStdvar
	r.Name = f.Name
	r.Namespace = f.Namespace
	r.Type = f.Type
	return r
}
func Convert_Workload_WorkloadCommon(f *orm.Workload) *WorkloadCommon {
	if f == nil {
		return nil
	}
	var r WorkloadCommon
	r.CPULimitStdvar = f.CPULimitStdvar
	r.Cluster = f.Cluster
	r.Containers = Convert_Container_ContainerCommon_slice(f.Containers)
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.MemoryLimitStdvar = f.MemoryLimitStdvar
	r.Name = f.Name
	r.Namespace = f.Namespace
	r.Type = f.Type
	return &r
}
func Convert_WorkloadCommon_Workload_slice(arr []*WorkloadCommon) []*orm.Workload {
	r := []*orm.Workload{}
	for _, u := range arr {
		r = append(r, Convert_WorkloadCommon_Workload(u))
	}
	return r
}

func Convert_Workload_WorkloadCommon_slice(arr []*orm.Workload) []*WorkloadCommon {
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

func (ul *VirtualSpaceUserRelCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.VirtualSpaceUserRelList{}
	return ul.objectlist
}

func (ul *VirtualSpaceUserRelCommonList) Data() []*VirtualSpaceUserRelCommon {
	if ul.data != nil {
		return ul.data.([]*VirtualSpaceUserRelCommon)
	}
	us := ul.objectlist.(*orm.VirtualSpaceUserRelList)
	tmp := Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *VirtualSpaceUserRelCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *VirtualSpaceUserRelCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel(r)
	}
	return r.object
}

func (u *VirtualSpaceUserRelCommon) Data() *VirtualSpaceUserRelCommon {
	if u.data != nil {
		return u.data.(*VirtualSpaceUserRelCommon)
	}
	tmp := Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon(u.object.(*orm.VirtualSpaceUserRel))
	u.data = tmp
	return tmp
}

func (u *VirtualSpaceUserRelCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel(f *VirtualSpaceUserRelCommon) *orm.VirtualSpaceUserRel {
	r := &orm.VirtualSpaceUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Role = f.Role
	r.User = Convert_UserCommon_User(f.User)
	r.UserID = f.UserID
	r.VirtualSpace = Convert_VirtualSpaceCommon_VirtualSpace(f.VirtualSpace)
	r.VirtualSpaceID = f.VirtualSpaceID
	return r
}
func Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon(f *orm.VirtualSpaceUserRel) *VirtualSpaceUserRelCommon {
	if f == nil {
		return nil
	}
	var r VirtualSpaceUserRelCommon
	r.ID = f.ID
	r.Role = f.Role
	r.User = Convert_User_UserCommon(f.User)
	r.UserID = f.UserID
	r.VirtualSpace = Convert_VirtualSpace_VirtualSpaceCommon(f.VirtualSpace)
	r.VirtualSpaceID = f.VirtualSpaceID
	return &r
}
func Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel_slice(arr []*VirtualSpaceUserRelCommon) []*orm.VirtualSpaceUserRel {
	r := []*orm.VirtualSpaceUserRel{}
	for _, u := range arr {
		r = append(r, Convert_VirtualSpaceUserRelCommon_VirtualSpaceUserRel(u))
	}
	return r
}

func Convert_VirtualSpaceUserRel_VirtualSpaceUserRelCommon_slice(arr []*orm.VirtualSpaceUserRel) []*VirtualSpaceUserRelCommon {
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

func (ul *VirtualSpaceCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.VirtualSpaceList{}
	return ul.objectlist
}

func (ul *VirtualSpaceCommonList) Data() []*VirtualSpaceCommon {
	if ul.data != nil {
		return ul.data.([]*VirtualSpaceCommon)
	}
	us := ul.objectlist.(*orm.VirtualSpaceList)
	tmp := Convert_VirtualSpace_VirtualSpaceCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *VirtualSpaceCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *VirtualSpaceCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_VirtualSpaceCommon_VirtualSpace(r)
	}
	return r.object
}

func (u *VirtualSpaceCommon) Data() *VirtualSpaceCommon {
	if u.data != nil {
		return u.data.(*VirtualSpaceCommon)
	}
	tmp := Convert_VirtualSpace_VirtualSpaceCommon(u.object.(*orm.VirtualSpace))
	u.data = tmp
	return tmp
}

func (u *VirtualSpaceCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_VirtualSpaceCommon_VirtualSpace(f *VirtualSpaceCommon) *orm.VirtualSpace {
	r := &orm.VirtualSpace{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Name = f.Name
	return r
}
func Convert_VirtualSpace_VirtualSpaceCommon(f *orm.VirtualSpace) *VirtualSpaceCommon {
	if f == nil {
		return nil
	}
	var r VirtualSpaceCommon
	r.ID = f.ID
	r.Name = f.Name
	return &r
}
func Convert_VirtualSpaceCommon_VirtualSpace_slice(arr []*VirtualSpaceCommon) []*orm.VirtualSpace {
	r := []*orm.VirtualSpace{}
	for _, u := range arr {
		r = append(r, Convert_VirtualSpaceCommon_VirtualSpace(u))
	}
	return r
}

func Convert_VirtualSpace_VirtualSpaceCommon_slice(arr []*orm.VirtualSpace) []*VirtualSpaceCommon {
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

func (ul *VirtualDomainCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.VirtualDomainList{}
	return ul.objectlist
}

func (ul *VirtualDomainCommonList) Data() []*VirtualDomainCommon {
	if ul.data != nil {
		return ul.data.([]*VirtualDomainCommon)
	}
	us := ul.objectlist.(*orm.VirtualDomainList)
	tmp := Convert_VirtualDomain_VirtualDomainCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *VirtualDomainCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *VirtualDomainCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_VirtualDomainCommon_VirtualDomain(r)
	}
	return r.object
}

func (u *VirtualDomainCommon) Data() *VirtualDomainCommon {
	if u.data != nil {
		return u.data.(*VirtualDomainCommon)
	}
	tmp := Convert_VirtualDomain_VirtualDomainCommon(u.object.(*orm.VirtualDomain))
	u.data = tmp
	return tmp
}

func (u *VirtualDomainCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_VirtualDomainCommon_VirtualDomain(f *VirtualDomainCommon) *orm.VirtualDomain {
	r := &orm.VirtualDomain{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.CreatedBy = f.CreatedBy
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.Name = f.Name
	r.UpdatedAt = f.UpdatedAt
	return r
}
func Convert_VirtualDomain_VirtualDomainCommon(f *orm.VirtualDomain) *VirtualDomainCommon {
	if f == nil {
		return nil
	}
	var r VirtualDomainCommon
	r.CreatedAt = f.CreatedAt
	r.CreatedBy = f.CreatedBy
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.Name = f.Name
	r.UpdatedAt = f.UpdatedAt
	return &r
}
func Convert_VirtualDomainCommon_VirtualDomain_slice(arr []*VirtualDomainCommon) []*orm.VirtualDomain {
	r := []*orm.VirtualDomain{}
	for _, u := range arr {
		r = append(r, Convert_VirtualDomainCommon_VirtualDomain(u))
	}
	return r
}

func Convert_VirtualDomain_VirtualDomainCommon_slice(arr []*orm.VirtualDomain) []*VirtualDomainCommon {
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

func (ul *UserSettingList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserSettingList) Data() []*UserSetting {
	if ul.data != nil {
		return ul.data.([]*UserSetting)
	}
	us := ul.objectlist.(*orm.UserList)
	tmp := Convert_User_UserSetting_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *UserSettingList) DataPtr() interface{} {
	return ul.Data()
}

func (r *UserSetting) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserSetting_User(r)
	}
	return r.object
}

func (u *UserSetting) Data() *UserSetting {
	if u.data != nil {
		return u.data.(*UserSetting)
	}
	tmp := Convert_User_UserSetting(u.object.(*orm.User))
	u.data = tmp
	return tmp
}

func (u *UserSetting) DataPtr() interface{} {
	return u.Data()
}

func Convert_UserSetting_User(f *UserSetting) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Email = f.Email
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.Name = f.Name
	r.Password = f.Password
	r.Phone = f.Phone
	r.SystemRole = Convert_SystemRoleCommon_SystemRole(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return r
}
func Convert_User_UserSetting(f *orm.User) *UserSetting {
	if f == nil {
		return nil
	}
	var r UserSetting
	r.Email = f.Email
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.Name = f.Name
	r.Password = f.Password
	r.Phone = f.Phone
	r.SystemRole = Convert_SystemRole_SystemRoleCommon(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return &r
}
func Convert_UserSetting_User_slice(arr []*UserSetting) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserSetting_User(u))
	}
	return r
}

func Convert_User_UserSetting_slice(arr []*orm.User) []*UserSetting {
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

func (ul *UserMessageStatusCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.UserMessageStatusList{}
	return ul.objectlist
}

func (ul *UserMessageStatusCommonList) Data() []*UserMessageStatusCommon {
	if ul.data != nil {
		return ul.data.([]*UserMessageStatusCommon)
	}
	us := ul.objectlist.(*orm.UserMessageStatusList)
	tmp := Convert_UserMessageStatus_UserMessageStatusCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *UserMessageStatusCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *UserMessageStatusCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserMessageStatusCommon_UserMessageStatus(r)
	}
	return r.object
}

func (u *UserMessageStatusCommon) Data() *UserMessageStatusCommon {
	if u.data != nil {
		return u.data.(*UserMessageStatusCommon)
	}
	tmp := Convert_UserMessageStatus_UserMessageStatusCommon(u.object.(*orm.UserMessageStatus))
	u.data = tmp
	return tmp
}

func (u *UserMessageStatusCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_UserMessageStatusCommon_UserMessageStatus(f *UserMessageStatusCommon) *orm.UserMessageStatus {
	r := &orm.UserMessageStatus{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.IsRead = f.IsRead
	r.Message = Convert_MessageCommon_Message(f.Message)
	r.MessageID = f.MessageID
	r.User = Convert_UserCommon_User(f.User)
	r.UserID = f.UserID
	return r
}
func Convert_UserMessageStatus_UserMessageStatusCommon(f *orm.UserMessageStatus) *UserMessageStatusCommon {
	if f == nil {
		return nil
	}
	var r UserMessageStatusCommon
	r.ID = f.ID
	r.IsRead = f.IsRead
	r.Message = Convert_Message_MessageCommon(f.Message)
	r.MessageID = f.MessageID
	r.User = Convert_User_UserCommon(f.User)
	r.UserID = f.UserID
	return &r
}
func Convert_UserMessageStatusCommon_UserMessageStatus_slice(arr []*UserMessageStatusCommon) []*orm.UserMessageStatus {
	r := []*orm.UserMessageStatus{}
	for _, u := range arr {
		r = append(r, Convert_UserMessageStatusCommon_UserMessageStatus(u))
	}
	return r
}

func Convert_UserMessageStatus_UserMessageStatusCommon_slice(arr []*orm.UserMessageStatus) []*UserMessageStatusCommon {
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

func (ul *UserInternalList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserInternalList) Data() []*UserInternal {
	if ul.data != nil {
		return ul.data.([]*UserInternal)
	}
	us := ul.objectlist.(*orm.UserList)
	tmp := Convert_User_UserInternal_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *UserInternalList) DataPtr() interface{} {
	return ul.Data()
}

func (r *UserInternal) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserInternal_User(r)
	}
	return r.object
}

func (u *UserInternal) Data() *UserInternal {
	if u.data != nil {
		return u.data.(*UserInternal)
	}
	tmp := Convert_User_UserInternal(u.object.(*orm.User))
	u.data = tmp
	return tmp
}

func (u *UserInternal) DataPtr() interface{} {
	return u.Data()
}

func Convert_UserInternal_User(f *UserInternal) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.Email = f.Email
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.LastLoginAt = f.LastLoginAt
	r.Name = f.Name
	r.Password = f.Password
	r.Phone = f.Phone
	r.Role = f.Role
	r.Source = f.Source
	r.SystemRole = Convert_SystemRoleCommon_SystemRole(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return r
}
func Convert_User_UserInternal(f *orm.User) *UserInternal {
	if f == nil {
		return nil
	}
	var r UserInternal
	r.CreatedAt = f.CreatedAt
	r.Email = f.Email
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.LastLoginAt = f.LastLoginAt
	r.Name = f.Name
	r.Password = f.Password
	r.Phone = f.Phone
	r.Role = f.Role
	r.Source = f.Source
	r.SystemRole = Convert_SystemRole_SystemRoleCommon(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return &r
}
func Convert_UserInternal_User_slice(arr []*UserInternal) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserInternal_User(u))
	}
	return r
}

func Convert_User_UserInternal_slice(arr []*orm.User) []*UserInternal {
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

func (ul *UserDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserDetailList) Data() []*UserDetail {
	if ul.data != nil {
		return ul.data.([]*UserDetail)
	}
	us := ul.objectlist.(*orm.UserList)
	tmp := Convert_User_UserDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *UserDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *UserDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserDetail_User(r)
	}
	return r.object
}

func (u *UserDetail) Data() *UserDetail {
	if u.data != nil {
		return u.data.(*UserDetail)
	}
	tmp := Convert_User_UserDetail(u.object.(*orm.User))
	u.data = tmp
	return tmp
}

func (u *UserDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_UserDetail_User(f *UserDetail) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.Email = f.Email
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.LastLoginAt = f.LastLoginAt
	r.Name = f.Name
	r.Phone = f.Phone
	r.Role = f.Role
	r.Source = f.Source
	r.SystemRole = Convert_SystemRoleCommon_SystemRole(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return r
}
func Convert_User_UserDetail(f *orm.User) *UserDetail {
	if f == nil {
		return nil
	}
	var r UserDetail
	r.CreatedAt = f.CreatedAt
	r.Email = f.Email
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.LastLoginAt = f.LastLoginAt
	r.Name = f.Name
	r.Phone = f.Phone
	r.Role = f.Role
	r.Source = f.Source
	r.SystemRole = Convert_SystemRole_SystemRoleCommon(f.SystemRole)
	r.SystemRoleID = f.SystemRoleID
	return &r
}
func Convert_UserDetail_User_slice(arr []*UserDetail) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserDetail_User(u))
	}
	return r
}

func Convert_User_UserDetail_slice(arr []*orm.User) []*UserDetail {
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

func (ul *UserCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.UserList{}
	return ul.objectlist
}

func (ul *UserCommonList) Data() []*UserCommon {
	if ul.data != nil {
		return ul.data.([]*UserCommon)
	}
	us := ul.objectlist.(*orm.UserList)
	tmp := Convert_User_UserCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *UserCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *UserCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_UserCommon_User(r)
	}
	return r.object
}

func (u *UserCommon) Data() *UserCommon {
	if u.data != nil {
		return u.data.(*UserCommon)
	}
	tmp := Convert_User_UserCommon(u.object.(*orm.User))
	u.data = tmp
	return tmp
}

func (u *UserCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_UserCommon_User(f *UserCommon) *orm.User {
	r := &orm.User{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Email = f.Email
	r.ID = f.ID
	r.Name = f.Name
	r.Role = f.Role
	return r
}
func Convert_User_UserCommon(f *orm.User) *UserCommon {
	if f == nil {
		return nil
	}
	var r UserCommon
	r.Email = f.Email
	r.ID = f.ID
	r.Name = f.Name
	r.Role = f.Role
	return &r
}
func Convert_UserCommon_User_slice(arr []*UserCommon) []*orm.User {
	r := []*orm.User{}
	for _, u := range arr {
		r = append(r, Convert_UserCommon_User(u))
	}
	return r
}

func Convert_User_UserCommon_slice(arr []*orm.User) []*UserCommon {
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

func (ul *TenantUserRelCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.TenantUserRelList{}
	return ul.objectlist
}

func (ul *TenantUserRelCommonList) Data() []*TenantUserRelCommon {
	if ul.data != nil {
		return ul.data.([]*TenantUserRelCommon)
	}
	us := ul.objectlist.(*orm.TenantUserRelList)
	tmp := Convert_TenantUserRel_TenantUserRelCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *TenantUserRelCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *TenantUserRelCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantUserRelCommon_TenantUserRel(r)
	}
	return r.object
}

func (u *TenantUserRelCommon) Data() *TenantUserRelCommon {
	if u.data != nil {
		return u.data.(*TenantUserRelCommon)
	}
	tmp := Convert_TenantUserRel_TenantUserRelCommon(u.object.(*orm.TenantUserRel))
	u.data = tmp
	return tmp
}

func (u *TenantUserRelCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_TenantUserRelCommon_TenantUserRel(f *TenantUserRelCommon) *orm.TenantUserRel {
	r := &orm.TenantUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Role = f.Role
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.TenantID = f.TenantID
	r.User = Convert_UserCommon_User(f.User)
	r.UserID = f.UserID
	return r
}
func Convert_TenantUserRel_TenantUserRelCommon(f *orm.TenantUserRel) *TenantUserRelCommon {
	if f == nil {
		return nil
	}
	var r TenantUserRelCommon
	r.ID = f.ID
	r.Role = f.Role
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.TenantID = f.TenantID
	r.User = Convert_User_UserCommon(f.User)
	r.UserID = f.UserID
	return &r
}
func Convert_TenantUserRelCommon_TenantUserRel_slice(arr []*TenantUserRelCommon) []*orm.TenantUserRel {
	r := []*orm.TenantUserRel{}
	for _, u := range arr {
		r = append(r, Convert_TenantUserRelCommon_TenantUserRel(u))
	}
	return r
}

func Convert_TenantUserRel_TenantUserRelCommon_slice(arr []*orm.TenantUserRel) []*TenantUserRelCommon {
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

func (ul *TenantResourceQuotaCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.TenantResourceQuotaList{}
	return ul.objectlist
}

func (ul *TenantResourceQuotaCommonList) Data() []*TenantResourceQuotaCommon {
	if ul.data != nil {
		return ul.data.([]*TenantResourceQuotaCommon)
	}
	us := ul.objectlist.(*orm.TenantResourceQuotaList)
	tmp := Convert_TenantResourceQuota_TenantResourceQuotaCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *TenantResourceQuotaCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *TenantResourceQuotaCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantResourceQuotaCommon_TenantResourceQuota(r)
	}
	return r.object
}

func (u *TenantResourceQuotaCommon) Data() *TenantResourceQuotaCommon {
	if u.data != nil {
		return u.data.(*TenantResourceQuotaCommon)
	}
	tmp := Convert_TenantResourceQuota_TenantResourceQuotaCommon(u.object.(*orm.TenantResourceQuota))
	u.data = tmp
	return tmp
}

func (u *TenantResourceQuotaCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_TenantResourceQuotaCommon_TenantResourceQuota(f *TenantResourceQuotaCommon) *orm.TenantResourceQuota {
	r := &orm.TenantResourceQuota{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.Content = f.Content
	r.ID = f.ID
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.TenantID = f.TenantID
	return r
}
func Convert_TenantResourceQuota_TenantResourceQuotaCommon(f *orm.TenantResourceQuota) *TenantResourceQuotaCommon {
	if f == nil {
		return nil
	}
	var r TenantResourceQuotaCommon
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.Content = f.Content
	r.ID = f.ID
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.TenantID = f.TenantID
	return &r
}
func Convert_TenantResourceQuotaCommon_TenantResourceQuota_slice(arr []*TenantResourceQuotaCommon) []*orm.TenantResourceQuota {
	r := []*orm.TenantResourceQuota{}
	for _, u := range arr {
		r = append(r, Convert_TenantResourceQuotaCommon_TenantResourceQuota(u))
	}
	return r
}

func Convert_TenantResourceQuota_TenantResourceQuotaCommon_slice(arr []*orm.TenantResourceQuota) []*TenantResourceQuotaCommon {
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

func (ul *TenantResourceQuotaApplyCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.TenantResourceQuotaApplyList{}
	return ul.objectlist
}

func (ul *TenantResourceQuotaApplyCommonList) Data() []*TenantResourceQuotaApplyCommon {
	if ul.data != nil {
		return ul.data.([]*TenantResourceQuotaApplyCommon)
	}
	us := ul.objectlist.(*orm.TenantResourceQuotaApplyList)
	tmp := Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *TenantResourceQuotaApplyCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *TenantResourceQuotaApplyCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(r)
	}
	return r.object
}

func (u *TenantResourceQuotaApplyCommon) Data() *TenantResourceQuotaApplyCommon {
	if u.data != nil {
		return u.data.(*TenantResourceQuotaApplyCommon)
	}
	tmp := Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon(u.object.(*orm.TenantResourceQuotaApply))
	u.data = tmp
	return tmp
}

func (u *TenantResourceQuotaApplyCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(f *TenantResourceQuotaApplyCommon) *orm.TenantResourceQuotaApply {
	r := &orm.TenantResourceQuotaApply{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.Content = f.Content
	r.CreateAt = f.CreateAt
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.Status = f.Status
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.TenantID = f.TenantID
	return r
}
func Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon(f *orm.TenantResourceQuotaApply) *TenantResourceQuotaApplyCommon {
	if f == nil {
		return nil
	}
	var r TenantResourceQuotaApplyCommon
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.Content = f.Content
	r.CreateAt = f.CreateAt
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.Status = f.Status
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.TenantID = f.TenantID
	return &r
}
func Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply_slice(arr []*TenantResourceQuotaApplyCommon) []*orm.TenantResourceQuotaApply {
	r := []*orm.TenantResourceQuotaApply{}
	for _, u := range arr {
		r = append(r, Convert_TenantResourceQuotaApplyCommon_TenantResourceQuotaApply(u))
	}
	return r
}

func Convert_TenantResourceQuotaApply_TenantResourceQuotaApplyCommon_slice(arr []*orm.TenantResourceQuotaApply) []*TenantResourceQuotaApplyCommon {
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

func (ul *TenantDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.TenantList{}
	return ul.objectlist
}

func (ul *TenantDetailList) Data() []*TenantDetail {
	if ul.data != nil {
		return ul.data.([]*TenantDetail)
	}
	us := ul.objectlist.(*orm.TenantList)
	tmp := Convert_Tenant_TenantDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *TenantDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *TenantDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantDetail_Tenant(r)
	}
	return r.object
}

func (u *TenantDetail) Data() *TenantDetail {
	if u.data != nil {
		return u.data.(*TenantDetail)
	}
	tmp := Convert_Tenant_TenantDetail(u.object.(*orm.Tenant))
	u.data = tmp
	return tmp
}

func (u *TenantDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_TenantDetail_Tenant(f *TenantDetail) *orm.Tenant {
	r := &orm.Tenant{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.Name = f.Name
	r.Remark = f.Remark
	r.Users = Convert_UserCommon_User_slice(f.Users)
	return r
}
func Convert_Tenant_TenantDetail(f *orm.Tenant) *TenantDetail {
	if f == nil {
		return nil
	}
	var r TenantDetail
	r.ID = f.ID
	r.IsActive = f.IsActive
	r.Name = f.Name
	r.Remark = f.Remark
	r.Users = Convert_User_UserCommon_slice(f.Users)
	return &r
}
func Convert_TenantDetail_Tenant_slice(arr []*TenantDetail) []*orm.Tenant {
	r := []*orm.Tenant{}
	for _, u := range arr {
		r = append(r, Convert_TenantDetail_Tenant(u))
	}
	return r
}

func Convert_Tenant_TenantDetail_slice(arr []*orm.Tenant) []*TenantDetail {
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

func (ul *TenantCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.TenantList{}
	return ul.objectlist
}

func (ul *TenantCommonList) Data() []*TenantCommon {
	if ul.data != nil {
		return ul.data.([]*TenantCommon)
	}
	us := ul.objectlist.(*orm.TenantList)
	tmp := Convert_Tenant_TenantCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *TenantCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *TenantCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_TenantCommon_Tenant(r)
	}
	return r.object
}

func (u *TenantCommon) Data() *TenantCommon {
	if u.data != nil {
		return u.data.(*TenantCommon)
	}
	tmp := Convert_Tenant_TenantCommon(u.object.(*orm.Tenant))
	u.data = tmp
	return tmp
}

func (u *TenantCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_TenantCommon_Tenant(f *TenantCommon) *orm.Tenant {
	r := &orm.Tenant{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Name = f.Name
	return r
}
func Convert_Tenant_TenantCommon(f *orm.Tenant) *TenantCommon {
	if f == nil {
		return nil
	}
	var r TenantCommon
	r.ID = f.ID
	r.Name = f.Name
	return &r
}
func Convert_TenantCommon_Tenant_slice(arr []*TenantCommon) []*orm.Tenant {
	r := []*orm.Tenant{}
	for _, u := range arr {
		r = append(r, Convert_TenantCommon_Tenant(u))
	}
	return r
}

func Convert_Tenant_TenantCommon_slice(arr []*orm.Tenant) []*TenantCommon {
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

func (ul *SystemRoleDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.SystemRoleList{}
	return ul.objectlist
}

func (ul *SystemRoleDetailList) Data() []*SystemRoleDetail {
	if ul.data != nil {
		return ul.data.([]*SystemRoleDetail)
	}
	us := ul.objectlist.(*orm.SystemRoleList)
	tmp := Convert_SystemRole_SystemRoleDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *SystemRoleDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *SystemRoleDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_SystemRoleDetail_SystemRole(r)
	}
	return r.object
}

func (u *SystemRoleDetail) Data() *SystemRoleDetail {
	if u.data != nil {
		return u.data.(*SystemRoleDetail)
	}
	tmp := Convert_SystemRole_SystemRoleDetail(u.object.(*orm.SystemRole))
	u.data = tmp
	return tmp
}

func (u *SystemRoleDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_SystemRoleDetail_SystemRole(f *SystemRoleDetail) *orm.SystemRole {
	r := &orm.SystemRole{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Code = f.Code
	r.ID = f.ID
	r.Name = f.Name
	r.Users = Convert_UserCommon_User_slice(f.Users)
	return r
}
func Convert_SystemRole_SystemRoleDetail(f *orm.SystemRole) *SystemRoleDetail {
	if f == nil {
		return nil
	}
	var r SystemRoleDetail
	r.Code = f.Code
	r.ID = f.ID
	r.Name = f.Name
	r.Users = Convert_User_UserCommon_slice(f.Users)
	return &r
}
func Convert_SystemRoleDetail_SystemRole_slice(arr []*SystemRoleDetail) []*orm.SystemRole {
	r := []*orm.SystemRole{}
	for _, u := range arr {
		r = append(r, Convert_SystemRoleDetail_SystemRole(u))
	}
	return r
}

func Convert_SystemRole_SystemRoleDetail_slice(arr []*orm.SystemRole) []*SystemRoleDetail {
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

func (ul *SystemRoleCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.SystemRoleList{}
	return ul.objectlist
}

func (ul *SystemRoleCommonList) Data() []*SystemRoleCommon {
	if ul.data != nil {
		return ul.data.([]*SystemRoleCommon)
	}
	us := ul.objectlist.(*orm.SystemRoleList)
	tmp := Convert_SystemRole_SystemRoleCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *SystemRoleCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *SystemRoleCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_SystemRoleCommon_SystemRole(r)
	}
	return r.object
}

func (u *SystemRoleCommon) Data() *SystemRoleCommon {
	if u.data != nil {
		return u.data.(*SystemRoleCommon)
	}
	tmp := Convert_SystemRole_SystemRoleCommon(u.object.(*orm.SystemRole))
	u.data = tmp
	return tmp
}

func (u *SystemRoleCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_SystemRoleCommon_SystemRole(f *SystemRoleCommon) *orm.SystemRole {
	r := &orm.SystemRole{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Code = f.Code
	r.ID = f.ID
	r.Name = f.Name
	return r
}
func Convert_SystemRole_SystemRoleCommon(f *orm.SystemRole) *SystemRoleCommon {
	if f == nil {
		return nil
	}
	var r SystemRoleCommon
	r.Code = f.Code
	r.ID = f.ID
	r.Name = f.Name
	return &r
}
func Convert_SystemRoleCommon_SystemRole_slice(arr []*SystemRoleCommon) []*orm.SystemRole {
	r := []*orm.SystemRole{}
	for _, u := range arr {
		r = append(r, Convert_SystemRoleCommon_SystemRole(u))
	}
	return r
}

func Convert_SystemRole_SystemRoleCommon_slice(arr []*orm.SystemRole) []*SystemRoleCommon {
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

func (ul *RegistryDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.RegistryList{}
	return ul.objectlist
}

func (ul *RegistryDetailList) Data() []*RegistryDetail {
	if ul.data != nil {
		return ul.data.([]*RegistryDetail)
	}
	us := ul.objectlist.(*orm.RegistryList)
	tmp := Convert_Registry_RegistryDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *RegistryDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *RegistryDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_RegistryDetail_Registry(r)
	}
	return r.object
}

func (u *RegistryDetail) Data() *RegistryDetail {
	if u.data != nil {
		return u.data.(*RegistryDetail)
	}
	tmp := Convert_Registry_RegistryDetail(u.object.(*orm.Registry))
	u.data = tmp
	return tmp
}

func (u *RegistryDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_RegistryDetail_Registry(f *RegistryDetail) *orm.Registry {
	r := &orm.Registry{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Address = f.Address
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.IsDefault = f.IsDefault
	r.Name = f.Name
	r.Password = f.Password
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.UpdateTime = f.UpdateTime
	r.Username = f.Username
	return r
}
func Convert_Registry_RegistryDetail(f *orm.Registry) *RegistryDetail {
	if f == nil {
		return nil
	}
	var r RegistryDetail
	r.Address = f.Address
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.IsDefault = f.IsDefault
	r.Name = f.Name
	r.Password = f.Password
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	r.UpdateTime = f.UpdateTime
	r.Username = f.Username
	return &r
}
func Convert_RegistryDetail_Registry_slice(arr []*RegistryDetail) []*orm.Registry {
	r := []*orm.Registry{}
	for _, u := range arr {
		r = append(r, Convert_RegistryDetail_Registry(u))
	}
	return r
}

func Convert_Registry_RegistryDetail_slice(arr []*orm.Registry) []*RegistryDetail {
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

func (ul *RegistryCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.RegistryList{}
	return ul.objectlist
}

func (ul *RegistryCommonList) Data() []*RegistryCommon {
	if ul.data != nil {
		return ul.data.([]*RegistryCommon)
	}
	us := ul.objectlist.(*orm.RegistryList)
	tmp := Convert_Registry_RegistryCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *RegistryCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *RegistryCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_RegistryCommon_Registry(r)
	}
	return r.object
}

func (u *RegistryCommon) Data() *RegistryCommon {
	if u.data != nil {
		return u.data.(*RegistryCommon)
	}
	tmp := Convert_Registry_RegistryCommon(u.object.(*orm.Registry))
	u.data = tmp
	return tmp
}

func (u *RegistryCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_RegistryCommon_Registry(f *RegistryCommon) *orm.Registry {
	r := &orm.Registry{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Address = f.Address
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.IsDefault = f.IsDefault
	r.Name = f.Name
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.UpdateTime = f.UpdateTime
	return r
}
func Convert_Registry_RegistryCommon(f *orm.Registry) *RegistryCommon {
	if f == nil {
		return nil
	}
	var r RegistryCommon
	r.Address = f.Address
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.ID = f.ID
	r.IsDefault = f.IsDefault
	r.Name = f.Name
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	r.UpdateTime = f.UpdateTime
	return &r
}
func Convert_RegistryCommon_Registry_slice(arr []*RegistryCommon) []*orm.Registry {
	r := []*orm.Registry{}
	for _, u := range arr {
		r = append(r, Convert_RegistryCommon_Registry(u))
	}
	return r
}

func Convert_Registry_RegistryCommon_slice(arr []*orm.Registry) []*RegistryCommon {
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

func (ul *ProjectUserRelCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ProjectUserRelList{}
	return ul.objectlist
}

func (ul *ProjectUserRelCommonList) Data() []*ProjectUserRelCommon {
	if ul.data != nil {
		return ul.data.([]*ProjectUserRelCommon)
	}
	us := ul.objectlist.(*orm.ProjectUserRelList)
	tmp := Convert_ProjectUserRel_ProjectUserRelCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ProjectUserRelCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ProjectUserRelCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ProjectUserRelCommon_ProjectUserRel(r)
	}
	return r.object
}

func (u *ProjectUserRelCommon) Data() *ProjectUserRelCommon {
	if u.data != nil {
		return u.data.(*ProjectUserRelCommon)
	}
	tmp := Convert_ProjectUserRel_ProjectUserRelCommon(u.object.(*orm.ProjectUserRel))
	u.data = tmp
	return tmp
}

func (u *ProjectUserRelCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_ProjectUserRelCommon_ProjectUserRel(f *ProjectUserRelCommon) *orm.ProjectUserRel {
	r := &orm.ProjectUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.Role = f.Role
	r.User = Convert_UserCommon_User(f.User)
	r.UserID = f.UserID
	return r
}
func Convert_ProjectUserRel_ProjectUserRelCommon(f *orm.ProjectUserRel) *ProjectUserRelCommon {
	if f == nil {
		return nil
	}
	var r ProjectUserRelCommon
	r.ID = f.ID
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	r.Role = f.Role
	r.User = Convert_User_UserCommon(f.User)
	r.UserID = f.UserID
	return &r
}
func Convert_ProjectUserRelCommon_ProjectUserRel_slice(arr []*ProjectUserRelCommon) []*orm.ProjectUserRel {
	r := []*orm.ProjectUserRel{}
	for _, u := range arr {
		r = append(r, Convert_ProjectUserRelCommon_ProjectUserRel(u))
	}
	return r
}

func Convert_ProjectUserRel_ProjectUserRelCommon_slice(arr []*orm.ProjectUserRel) []*ProjectUserRelCommon {
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

func (ul *ProjectDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ProjectList{}
	return ul.objectlist
}

func (ul *ProjectDetailList) Data() []*ProjectDetail {
	if ul.data != nil {
		return ul.data.([]*ProjectDetail)
	}
	us := ul.objectlist.(*orm.ProjectList)
	tmp := Convert_Project_ProjectDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ProjectDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ProjectDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ProjectDetail_Project(r)
	}
	return r.object
}

func (u *ProjectDetail) Data() *ProjectDetail {
	if u.data != nil {
		return u.data.(*ProjectDetail)
	}
	tmp := Convert_Project_ProjectDetail(u.object.(*orm.Project))
	u.data = tmp
	return tmp
}

func (u *ProjectDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_ProjectDetail_Project(f *ProjectDetail) *orm.Project {
	r := &orm.Project{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Applications = Convert_ApplicationCommon_Application_slice(f.Applications)
	r.CreatedAt = f.CreatedAt
	r.Environments = Convert_EnvironmentCommon_Environment_slice(f.Environments)
	r.ID = f.ID
	r.Name = f.Name
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	r.ResourceQuota = f.ResourceQuota
	r.Tenant = Convert_TenantCommon_Tenant(f.Tenant)
	r.TenantID = f.TenantID
	r.Users = Convert_UserCommon_User_slice(f.Users)
	return r
}
func Convert_Project_ProjectDetail(f *orm.Project) *ProjectDetail {
	if f == nil {
		return nil
	}
	var r ProjectDetail
	r.Applications = Convert_Application_ApplicationCommon_slice(f.Applications)
	r.CreatedAt = f.CreatedAt
	r.Environments = Convert_Environment_EnvironmentCommon_slice(f.Environments)
	r.ID = f.ID
	r.Name = f.Name
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	r.ResourceQuota = f.ResourceQuota
	r.Tenant = Convert_Tenant_TenantCommon(f.Tenant)
	r.TenantID = f.TenantID
	r.Users = Convert_User_UserCommon_slice(f.Users)
	return &r
}
func Convert_ProjectDetail_Project_slice(arr []*ProjectDetail) []*orm.Project {
	r := []*orm.Project{}
	for _, u := range arr {
		r = append(r, Convert_ProjectDetail_Project(u))
	}
	return r
}

func Convert_Project_ProjectDetail_slice(arr []*orm.Project) []*ProjectDetail {
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

func (ul *ProjectCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ProjectList{}
	return ul.objectlist
}

func (ul *ProjectCommonList) Data() []*ProjectCommon {
	if ul.data != nil {
		return ul.data.([]*ProjectCommon)
	}
	us := ul.objectlist.(*orm.ProjectList)
	tmp := Convert_Project_ProjectCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ProjectCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ProjectCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ProjectCommon_Project(r)
	}
	return r.object
}

func (u *ProjectCommon) Data() *ProjectCommon {
	if u.data != nil {
		return u.data.(*ProjectCommon)
	}
	tmp := Convert_Project_ProjectCommon(u.object.(*orm.Project))
	u.data = tmp
	return tmp
}

func (u *ProjectCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_ProjectCommon_Project(f *ProjectCommon) *orm.Project {
	r := &orm.Project{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.Name = f.Name
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	return r
}
func Convert_Project_ProjectCommon(f *orm.Project) *ProjectCommon {
	if f == nil {
		return nil
	}
	var r ProjectCommon
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.Name = f.Name
	r.ProjectAlias = f.ProjectAlias
	r.Remark = f.Remark
	return &r
}
func Convert_ProjectCommon_Project_slice(arr []*ProjectCommon) []*orm.Project {
	r := []*orm.Project{}
	for _, u := range arr {
		r = append(r, Convert_ProjectCommon_Project(u))
	}
	return r
}

func Convert_Project_ProjectCommon_slice(arr []*orm.Project) []*ProjectCommon {
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

func (ul *OpenAPPDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.OpenAPPList{}
	return ul.objectlist
}

func (ul *OpenAPPDetailList) Data() []*OpenAPPDetail {
	if ul.data != nil {
		return ul.data.([]*OpenAPPDetail)
	}
	us := ul.objectlist.(*orm.OpenAPPList)
	tmp := Convert_OpenAPP_OpenAPPDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *OpenAPPDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *OpenAPPDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_OpenAPPDetail_OpenAPP(r)
	}
	return r.object
}

func (u *OpenAPPDetail) Data() *OpenAPPDetail {
	if u.data != nil {
		return u.data.(*OpenAPPDetail)
	}
	tmp := Convert_OpenAPP_OpenAPPDetail(u.object.(*orm.OpenAPP))
	u.data = tmp
	return tmp
}

func (u *OpenAPPDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_OpenAPPDetail_OpenAPP(f *OpenAPPDetail) *orm.OpenAPP {
	r := &orm.OpenAPP{}
	if f == nil {
		return nil
	}
	f.object = r
	r.AppID = f.AppID
	r.AppSecret = f.AppSecret
	r.ID = f.ID
	r.Name = f.Name
	r.PermScopes = f.PermScopes
	r.RequestLimiter = f.RequestLimiter
	r.TenantScope = f.TenantScope
	return r
}
func Convert_OpenAPP_OpenAPPDetail(f *orm.OpenAPP) *OpenAPPDetail {
	if f == nil {
		return nil
	}
	var r OpenAPPDetail
	r.AppID = f.AppID
	r.AppSecret = f.AppSecret
	r.ID = f.ID
	r.Name = f.Name
	r.PermScopes = f.PermScopes
	r.RequestLimiter = f.RequestLimiter
	r.TenantScope = f.TenantScope
	return &r
}
func Convert_OpenAPPDetail_OpenAPP_slice(arr []*OpenAPPDetail) []*orm.OpenAPP {
	r := []*orm.OpenAPP{}
	for _, u := range arr {
		r = append(r, Convert_OpenAPPDetail_OpenAPP(u))
	}
	return r
}

func Convert_OpenAPP_OpenAPPDetail_slice(arr []*orm.OpenAPP) []*OpenAPPDetail {
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

func (ul *OpenAPPCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.OpenAPPList{}
	return ul.objectlist
}

func (ul *OpenAPPCommonList) Data() []*OpenAPPCommon {
	if ul.data != nil {
		return ul.data.([]*OpenAPPCommon)
	}
	us := ul.objectlist.(*orm.OpenAPPList)
	tmp := Convert_OpenAPP_OpenAPPCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *OpenAPPCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *OpenAPPCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_OpenAPPCommon_OpenAPP(r)
	}
	return r.object
}

func (u *OpenAPPCommon) Data() *OpenAPPCommon {
	if u.data != nil {
		return u.data.(*OpenAPPCommon)
	}
	tmp := Convert_OpenAPP_OpenAPPCommon(u.object.(*orm.OpenAPP))
	u.data = tmp
	return tmp
}

func (u *OpenAPPCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_OpenAPPCommon_OpenAPP(f *OpenAPPCommon) *orm.OpenAPP {
	r := &orm.OpenAPP{}
	if f == nil {
		return nil
	}
	f.object = r
	r.AppID = f.AppID
	r.ID = f.ID
	r.Name = f.Name
	r.PermScopes = f.PermScopes
	r.RequestLimiter = f.RequestLimiter
	r.TenantScope = f.TenantScope
	return r
}
func Convert_OpenAPP_OpenAPPCommon(f *orm.OpenAPP) *OpenAPPCommon {
	if f == nil {
		return nil
	}
	var r OpenAPPCommon
	r.AppID = f.AppID
	r.ID = f.ID
	r.Name = f.Name
	r.PermScopes = f.PermScopes
	r.RequestLimiter = f.RequestLimiter
	r.TenantScope = f.TenantScope
	return &r
}
func Convert_OpenAPPCommon_OpenAPP_slice(arr []*OpenAPPCommon) []*orm.OpenAPP {
	r := []*orm.OpenAPP{}
	for _, u := range arr {
		r = append(r, Convert_OpenAPPCommon_OpenAPP(u))
	}
	return r
}

func Convert_OpenAPP_OpenAPPCommon_slice(arr []*orm.OpenAPP) []*OpenAPPCommon {
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

func (ul *MessageCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.MessageList{}
	return ul.objectlist
}

func (ul *MessageCommonList) Data() []*MessageCommon {
	if ul.data != nil {
		return ul.data.([]*MessageCommon)
	}
	us := ul.objectlist.(*orm.MessageList)
	tmp := Convert_Message_MessageCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *MessageCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *MessageCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_MessageCommon_Message(r)
	}
	return r.object
}

func (u *MessageCommon) Data() *MessageCommon {
	if u.data != nil {
		return u.data.(*MessageCommon)
	}
	tmp := Convert_Message_MessageCommon(u.object.(*orm.Message))
	u.data = tmp
	return tmp
}

func (u *MessageCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_MessageCommon_Message(f *MessageCommon) *orm.Message {
	r := &orm.Message{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Content = f.Content
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.MessageType = f.MessageType
	r.Title = f.Title
	return r
}
func Convert_Message_MessageCommon(f *orm.Message) *MessageCommon {
	if f == nil {
		return nil
	}
	var r MessageCommon
	r.Content = f.Content
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.MessageType = f.MessageType
	r.Title = f.Title
	return &r
}
func Convert_MessageCommon_Message_slice(arr []*MessageCommon) []*orm.Message {
	r := []*orm.Message{}
	for _, u := range arr {
		r = append(r, Convert_MessageCommon_Message(u))
	}
	return r
}

func Convert_Message_MessageCommon_slice(arr []*orm.Message) []*MessageCommon {
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

func (ul *LogQuerySnapshotCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.LogQuerySnapshotList{}
	return ul.objectlist
}

func (ul *LogQuerySnapshotCommonList) Data() []*LogQuerySnapshotCommon {
	if ul.data != nil {
		return ul.data.([]*LogQuerySnapshotCommon)
	}
	us := ul.objectlist.(*orm.LogQuerySnapshotList)
	tmp := Convert_LogQuerySnapshot_LogQuerySnapshotCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *LogQuerySnapshotCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *LogQuerySnapshotCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_LogQuerySnapshotCommon_LogQuerySnapshot(r)
	}
	return r.object
}

func (u *LogQuerySnapshotCommon) Data() *LogQuerySnapshotCommon {
	if u.data != nil {
		return u.data.(*LogQuerySnapshotCommon)
	}
	tmp := Convert_LogQuerySnapshot_LogQuerySnapshotCommon(u.object.(*orm.LogQuerySnapshot))
	u.data = tmp
	return tmp
}

func (u *LogQuerySnapshotCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_LogQuerySnapshotCommon_LogQuerySnapshot(f *LogQuerySnapshotCommon) *orm.LogQuerySnapshot {
	r := &orm.LogQuerySnapshot{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.CreateAt = f.CreateAt
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.DownloadURL = f.DownloadURL
	r.EndTime = f.EndTime
	r.ID = f.ID
	r.Name = f.Name
	r.SnapshotCount = f.SnapshotCount
	r.SourceFile = f.SourceFile
	r.StartTime = f.StartTime
	return r
}
func Convert_LogQuerySnapshot_LogQuerySnapshotCommon(f *orm.LogQuerySnapshot) *LogQuerySnapshotCommon {
	if f == nil {
		return nil
	}
	var r LogQuerySnapshotCommon
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.CreateAt = f.CreateAt
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.DownloadURL = f.DownloadURL
	r.EndTime = f.EndTime
	r.ID = f.ID
	r.Name = f.Name
	r.SnapshotCount = f.SnapshotCount
	r.SourceFile = f.SourceFile
	r.StartTime = f.StartTime
	return &r
}
func Convert_LogQuerySnapshotCommon_LogQuerySnapshot_slice(arr []*LogQuerySnapshotCommon) []*orm.LogQuerySnapshot {
	r := []*orm.LogQuerySnapshot{}
	for _, u := range arr {
		r = append(r, Convert_LogQuerySnapshotCommon_LogQuerySnapshot(u))
	}
	return r
}

func Convert_LogQuerySnapshot_LogQuerySnapshotCommon_slice(arr []*orm.LogQuerySnapshot) []*LogQuerySnapshotCommon {
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

func (ul *LogQueryHistoryCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.LogQueryHistoryList{}
	return ul.objectlist
}

func (ul *LogQueryHistoryCommonList) Data() []*LogQueryHistoryCommon {
	if ul.data != nil {
		return ul.data.([]*LogQueryHistoryCommon)
	}
	us := ul.objectlist.(*orm.LogQueryHistoryList)
	tmp := Convert_LogQueryHistory_LogQueryHistoryCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *LogQueryHistoryCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *LogQueryHistoryCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_LogQueryHistoryCommon_LogQueryHistory(r)
	}
	return r.object
}

func (u *LogQueryHistoryCommon) Data() *LogQueryHistoryCommon {
	if u.data != nil {
		return u.data.(*LogQueryHistoryCommon)
	}
	tmp := Convert_LogQueryHistory_LogQueryHistoryCommon(u.object.(*orm.LogQueryHistory))
	u.data = tmp
	return tmp
}

func (u *LogQueryHistoryCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_LogQueryHistoryCommon_LogQueryHistory(f *LogQueryHistoryCommon) *orm.LogQueryHistory {
	r := &orm.LogQueryHistory{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.CreateAt = f.CreateAt
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.FilterJSON = f.FilterJSON
	r.ID = f.ID
	r.LabelJSON = f.LabelJSON
	r.LogQL = f.LogQL
	return r
}
func Convert_LogQueryHistory_LogQueryHistoryCommon(f *orm.LogQueryHistory) *LogQueryHistoryCommon {
	if f == nil {
		return nil
	}
	var r LogQueryHistoryCommon
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.CreateAt = f.CreateAt
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.FilterJSON = f.FilterJSON
	r.ID = f.ID
	r.LabelJSON = f.LabelJSON
	r.LogQL = f.LogQL
	return &r
}
func Convert_LogQueryHistoryCommon_LogQueryHistory_slice(arr []*LogQueryHistoryCommon) []*orm.LogQueryHistory {
	r := []*orm.LogQueryHistory{}
	for _, u := range arr {
		r = append(r, Convert_LogQueryHistoryCommon_LogQueryHistory(u))
	}
	return r
}

func Convert_LogQueryHistory_LogQueryHistoryCommon_slice(arr []*orm.LogQueryHistory) []*LogQueryHistoryCommon {
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

func (ul *EnvironmentUserRelCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.EnvironmentUserRelList{}
	return ul.objectlist
}

func (ul *EnvironmentUserRelCommonList) Data() []*EnvironmentUserRelCommon {
	if ul.data != nil {
		return ul.data.([]*EnvironmentUserRelCommon)
	}
	us := ul.objectlist.(*orm.EnvironmentUserRelList)
	tmp := Convert_EnvironmentUserRel_EnvironmentUserRelCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *EnvironmentUserRelCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *EnvironmentUserRelCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentUserRelCommon_EnvironmentUserRel(r)
	}
	return r.object
}

func (u *EnvironmentUserRelCommon) Data() *EnvironmentUserRelCommon {
	if u.data != nil {
		return u.data.(*EnvironmentUserRelCommon)
	}
	tmp := Convert_EnvironmentUserRel_EnvironmentUserRelCommon(u.object.(*orm.EnvironmentUserRel))
	u.data = tmp
	return tmp
}

func (u *EnvironmentUserRelCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_EnvironmentUserRelCommon_EnvironmentUserRel(f *EnvironmentUserRelCommon) *orm.EnvironmentUserRel {
	r := &orm.EnvironmentUserRel{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Environment = Convert_EnvironmentCommon_Environment(f.Environment)
	r.EnvironmentID = f.EnvironmentID
	r.ID = f.ID
	r.Role = f.Role
	r.User = Convert_UserCommon_User(f.User)
	r.UserID = f.UserID
	return r
}
func Convert_EnvironmentUserRel_EnvironmentUserRelCommon(f *orm.EnvironmentUserRel) *EnvironmentUserRelCommon {
	if f == nil {
		return nil
	}
	var r EnvironmentUserRelCommon
	r.Environment = Convert_Environment_EnvironmentCommon(f.Environment)
	r.EnvironmentID = f.EnvironmentID
	r.ID = f.ID
	r.Role = f.Role
	r.User = Convert_User_UserCommon(f.User)
	r.UserID = f.UserID
	return &r
}
func Convert_EnvironmentUserRelCommon_EnvironmentUserRel_slice(arr []*EnvironmentUserRelCommon) []*orm.EnvironmentUserRel {
	r := []*orm.EnvironmentUserRel{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentUserRelCommon_EnvironmentUserRel(u))
	}
	return r
}

func Convert_EnvironmentUserRel_EnvironmentUserRelCommon_slice(arr []*orm.EnvironmentUserRel) []*EnvironmentUserRelCommon {
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

func (ul *EnvironmentResourceCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.EnvironmentResourceList{}
	return ul.objectlist
}

func (ul *EnvironmentResourceCommonList) Data() []*EnvironmentResourceCommon {
	if ul.data != nil {
		return ul.data.([]*EnvironmentResourceCommon)
	}
	us := ul.objectlist.(*orm.EnvironmentResourceList)
	tmp := Convert_EnvironmentResource_EnvironmentResourceCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *EnvironmentResourceCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *EnvironmentResourceCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentResourceCommon_EnvironmentResource(r)
	}
	return r.object
}

func (u *EnvironmentResourceCommon) Data() *EnvironmentResourceCommon {
	if u.data != nil {
		return u.data.(*EnvironmentResourceCommon)
	}
	tmp := Convert_EnvironmentResource_EnvironmentResourceCommon(u.object.(*orm.EnvironmentResource))
	u.data = tmp
	return tmp
}

func (u *EnvironmentResourceCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_EnvironmentResourceCommon_EnvironmentResource(f *EnvironmentResourceCommon) *orm.EnvironmentResource {
	r := &orm.EnvironmentResource{}
	if f == nil {
		return nil
	}
	f.object = r
	r.AvgCPUUsageCore = f.AvgCPUUsageCore
	r.AvgMemoryUsageByte = f.AvgMemoryUsageByte
	r.AvgPVCUsageByte = f.AvgPVCUsageByte
	r.Cluster = f.Cluster
	r.CreatedAt = f.CreatedAt
	r.Environment = f.Environment
	r.ID = f.ID
	r.MaxCPUUsageCore = f.MaxCPUUsageCore
	r.MaxMemoryUsageByte = f.MaxMemoryUsageByte
	r.MaxPVCUsageByte = f.MaxPVCUsageByte
	r.MinCPUUsageCore = f.MinCPUUsageCore
	r.MinMemoryUsageByte = f.MinMemoryUsageByte
	r.MinPVCUsageByte = f.MinPVCUsageByte
	r.NetworkReceiveByte = f.NetworkReceiveByte
	r.NetworkSendByte = f.NetworkSendByte
	r.Project = f.Project
	r.Tenant = f.Tenant
	return r
}
func Convert_EnvironmentResource_EnvironmentResourceCommon(f *orm.EnvironmentResource) *EnvironmentResourceCommon {
	if f == nil {
		return nil
	}
	var r EnvironmentResourceCommon
	r.AvgCPUUsageCore = f.AvgCPUUsageCore
	r.AvgMemoryUsageByte = f.AvgMemoryUsageByte
	r.AvgPVCUsageByte = f.AvgPVCUsageByte
	r.Cluster = f.Cluster
	r.CreatedAt = f.CreatedAt
	r.Environment = f.Environment
	r.ID = f.ID
	r.MaxCPUUsageCore = f.MaxCPUUsageCore
	r.MaxMemoryUsageByte = f.MaxMemoryUsageByte
	r.MaxPVCUsageByte = f.MaxPVCUsageByte
	r.MinCPUUsageCore = f.MinCPUUsageCore
	r.MinMemoryUsageByte = f.MinMemoryUsageByte
	r.MinPVCUsageByte = f.MinPVCUsageByte
	r.NetworkReceiveByte = f.NetworkReceiveByte
	r.NetworkSendByte = f.NetworkSendByte
	r.Project = f.Project
	r.Tenant = f.Tenant
	return &r
}
func Convert_EnvironmentResourceCommon_EnvironmentResource_slice(arr []*EnvironmentResourceCommon) []*orm.EnvironmentResource {
	r := []*orm.EnvironmentResource{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentResourceCommon_EnvironmentResource(u))
	}
	return r
}

func Convert_EnvironmentResource_EnvironmentResourceCommon_slice(arr []*orm.EnvironmentResource) []*EnvironmentResourceCommon {
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

func (ul *EnvironmentDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.EnvironmentList{}
	return ul.objectlist
}

func (ul *EnvironmentDetailList) Data() []*EnvironmentDetail {
	if ul.data != nil {
		return ul.data.([]*EnvironmentDetail)
	}
	us := ul.objectlist.(*orm.EnvironmentList)
	tmp := Convert_Environment_EnvironmentDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *EnvironmentDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *EnvironmentDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentDetail_Environment(r)
	}
	return r.object
}

func (u *EnvironmentDetail) Data() *EnvironmentDetail {
	if u.data != nil {
		return u.data.(*EnvironmentDetail)
	}
	tmp := Convert_Environment_EnvironmentDetail(u.object.(*orm.Environment))
	u.data = tmp
	return tmp
}

func (u *EnvironmentDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_EnvironmentDetail_Environment(f *EnvironmentDetail) *orm.Environment {
	r := &orm.Environment{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Applications = Convert_ApplicationCommon_Application_slice(f.Applications)
	r.Cluster = Convert_ClusterCommon_Cluster(f.Cluster)
	r.ClusterID = f.ClusterID
	r.Creator = Convert_UserCommon_User(f.Creator)
	r.CreatorID = f.CreatorID
	r.DeletePolicy = f.DeletePolicy
	r.ID = f.ID
	r.LimitRange = f.LimitRange
	r.MetaType = f.MetaType
	r.Name = f.Name
	r.Namespace = f.Namespace
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.Remark = f.Remark
	r.ResourceQuota = f.ResourceQuota
	r.Users = Convert_UserCommon_User_slice(f.Users)
	r.VirtualSpace = Convert_VirtualSpaceCommon_VirtualSpace(f.VirtualSpace)
	r.VirtualSpaceID = f.VirtualSpaceID
	return r
}
func Convert_Environment_EnvironmentDetail(f *orm.Environment) *EnvironmentDetail {
	if f == nil {
		return nil
	}
	var r EnvironmentDetail
	r.Applications = Convert_Application_ApplicationCommon_slice(f.Applications)
	r.Cluster = Convert_Cluster_ClusterCommon(f.Cluster)
	r.ClusterID = f.ClusterID
	r.Creator = Convert_User_UserCommon(f.Creator)
	r.CreatorID = f.CreatorID
	r.DeletePolicy = f.DeletePolicy
	r.ID = f.ID
	r.LimitRange = f.LimitRange
	r.MetaType = f.MetaType
	r.Name = f.Name
	r.Namespace = f.Namespace
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	r.Remark = f.Remark
	r.ResourceQuota = f.ResourceQuota
	r.Users = Convert_User_UserCommon_slice(f.Users)
	r.VirtualSpace = Convert_VirtualSpace_VirtualSpaceCommon(f.VirtualSpace)
	r.VirtualSpaceID = f.VirtualSpaceID
	return &r
}
func Convert_EnvironmentDetail_Environment_slice(arr []*EnvironmentDetail) []*orm.Environment {
	r := []*orm.Environment{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentDetail_Environment(u))
	}
	return r
}

func Convert_Environment_EnvironmentDetail_slice(arr []*orm.Environment) []*EnvironmentDetail {
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

func (ul *EnvironmentCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.EnvironmentList{}
	return ul.objectlist
}

func (ul *EnvironmentCommonList) Data() []*EnvironmentCommon {
	if ul.data != nil {
		return ul.data.([]*EnvironmentCommon)
	}
	us := ul.objectlist.(*orm.EnvironmentList)
	tmp := Convert_Environment_EnvironmentCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *EnvironmentCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *EnvironmentCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_EnvironmentCommon_Environment(r)
	}
	return r.object
}

func (u *EnvironmentCommon) Data() *EnvironmentCommon {
	if u.data != nil {
		return u.data.(*EnvironmentCommon)
	}
	tmp := Convert_Environment_EnvironmentCommon(u.object.(*orm.Environment))
	u.data = tmp
	return tmp
}

func (u *EnvironmentCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_EnvironmentCommon_Environment(f *EnvironmentCommon) *orm.Environment {
	r := &orm.Environment{}
	if f == nil {
		return nil
	}
	f.object = r
	r.DeletePolicy = f.DeletePolicy
	r.ID = f.ID
	r.MetaType = f.MetaType
	r.Name = f.Name
	r.Namespace = f.Namespace
	r.Remark = f.Remark
	return r
}
func Convert_Environment_EnvironmentCommon(f *orm.Environment) *EnvironmentCommon {
	if f == nil {
		return nil
	}
	var r EnvironmentCommon
	r.DeletePolicy = f.DeletePolicy
	r.ID = f.ID
	r.MetaType = f.MetaType
	r.Name = f.Name
	r.Namespace = f.Namespace
	r.Remark = f.Remark
	return &r
}
func Convert_EnvironmentCommon_Environment_slice(arr []*EnvironmentCommon) []*orm.Environment {
	r := []*orm.Environment{}
	for _, u := range arr {
		r = append(r, Convert_EnvironmentCommon_Environment(u))
	}
	return r
}

func Convert_Environment_EnvironmentCommon_slice(arr []*orm.Environment) []*EnvironmentCommon {
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

func (ul *ContainerCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ContainerList{}
	return ul.objectlist
}

func (ul *ContainerCommonList) Data() []*ContainerCommon {
	if ul.data != nil {
		return ul.data.([]*ContainerCommon)
	}
	us := ul.objectlist.(*orm.ContainerList)
	tmp := Convert_Container_ContainerCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ContainerCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ContainerCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ContainerCommon_Container(r)
	}
	return r.object
}

func (u *ContainerCommon) Data() *ContainerCommon {
	if u.data != nil {
		return u.data.(*ContainerCommon)
	}
	tmp := Convert_Container_ContainerCommon(u.object.(*orm.Container))
	u.data = tmp
	return tmp
}

func (u *ContainerCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_ContainerCommon_Container(f *ContainerCommon) *orm.Container {
	r := &orm.Container{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CPULimitCore = f.CPULimitCore
	r.CPUPercent = f.CPUPercent
	r.CPUUsageCore = f.CPUUsageCore
	r.ID = f.ID
	r.MemoryLimitBytes = f.MemoryLimitBytes
	r.MemoryPercent = f.MemoryPercent
	r.MemoryUsageBytes = f.MemoryUsageBytes
	r.Name = f.Name
	r.Workload = Convert_WorkloadCommon_Workload(f.Workload)
	r.WorkloadID = f.WorkloadID
	return r
}
func Convert_Container_ContainerCommon(f *orm.Container) *ContainerCommon {
	if f == nil {
		return nil
	}
	var r ContainerCommon
	r.CPULimitCore = f.CPULimitCore
	r.CPUPercent = f.CPUPercent
	r.CPUUsageCore = f.CPUUsageCore
	r.ID = f.ID
	r.MemoryLimitBytes = f.MemoryLimitBytes
	r.MemoryPercent = f.MemoryPercent
	r.MemoryUsageBytes = f.MemoryUsageBytes
	r.Name = f.Name
	r.Workload = Convert_Workload_WorkloadCommon(f.Workload)
	r.WorkloadID = f.WorkloadID
	return &r
}
func Convert_ContainerCommon_Container_slice(arr []*ContainerCommon) []*orm.Container {
	r := []*orm.Container{}
	for _, u := range arr {
		r = append(r, Convert_ContainerCommon_Container(u))
	}
	return r
}

func Convert_Container_ContainerCommon_slice(arr []*orm.Container) []*ContainerCommon {
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

func (ul *ClusterDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ClusterList{}
	return ul.objectlist
}

func (ul *ClusterDetailList) Data() []*ClusterDetail {
	if ul.data != nil {
		return ul.data.([]*ClusterDetail)
	}
	us := ul.objectlist.(*orm.ClusterList)
	tmp := Convert_Cluster_ClusterDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ClusterDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ClusterDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ClusterDetail_Cluster(r)
	}
	return r.object
}

func (u *ClusterDetail) Data() *ClusterDetail {
	if u.data != nil {
		return u.data.(*ClusterDetail)
	}
	tmp := Convert_Cluster_ClusterDetail(u.object.(*orm.Cluster))
	u.data = tmp
	return tmp
}

func (u *ClusterDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_ClusterDetail_Cluster(f *ClusterDetail) *orm.Cluster {
	r := &orm.Cluster{}
	if f == nil {
		return nil
	}
	f.object = r
	r.APIServer = f.APIServer
	r.AgentAddr = f.AgentAddr
	r.AgentCA = f.AgentCA
	r.AgentCert = f.AgentCert
	r.AgentKey = f.AgentKey
	r.ClusterResourceQuota = f.ClusterResourceQuota
	r.Environments = Convert_EnvironmentCommon_Environment_slice(f.Environments)
	r.ID = f.ID
	r.KubeConfig = f.KubeConfig
	r.Mode = f.Mode
	r.Name = f.Name
	r.OversoldConfig = f.OversoldConfig
	r.Primary = f.Primary
	r.Runtime = f.Runtime
	r.Version = f.Version
	return r
}
func Convert_Cluster_ClusterDetail(f *orm.Cluster) *ClusterDetail {
	if f == nil {
		return nil
	}
	var r ClusterDetail
	r.APIServer = f.APIServer
	r.AgentAddr = f.AgentAddr
	r.AgentCA = f.AgentCA
	r.AgentCert = f.AgentCert
	r.AgentKey = f.AgentKey
	r.ClusterResourceQuota = f.ClusterResourceQuota
	r.Environments = Convert_Environment_EnvironmentCommon_slice(f.Environments)
	r.ID = f.ID
	r.KubeConfig = f.KubeConfig
	r.Mode = f.Mode
	r.Name = f.Name
	r.OversoldConfig = f.OversoldConfig
	r.Primary = f.Primary
	r.Runtime = f.Runtime
	r.Version = f.Version
	return &r
}
func Convert_ClusterDetail_Cluster_slice(arr []*ClusterDetail) []*orm.Cluster {
	r := []*orm.Cluster{}
	for _, u := range arr {
		r = append(r, Convert_ClusterDetail_Cluster(u))
	}
	return r
}

func Convert_Cluster_ClusterDetail_slice(arr []*orm.Cluster) []*ClusterDetail {
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

func (ul *ClusterCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ClusterList{}
	return ul.objectlist
}

func (ul *ClusterCommonList) Data() []*ClusterCommon {
	if ul.data != nil {
		return ul.data.([]*ClusterCommon)
	}
	us := ul.objectlist.(*orm.ClusterList)
	tmp := Convert_Cluster_ClusterCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ClusterCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ClusterCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ClusterCommon_Cluster(r)
	}
	return r.object
}

func (u *ClusterCommon) Data() *ClusterCommon {
	if u.data != nil {
		return u.data.(*ClusterCommon)
	}
	tmp := Convert_Cluster_ClusterCommon(u.object.(*orm.Cluster))
	u.data = tmp
	return tmp
}

func (u *ClusterCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_ClusterCommon_Cluster(f *ClusterCommon) *orm.Cluster {
	r := &orm.Cluster{}
	if f == nil {
		return nil
	}
	f.object = r
	r.APIServer = f.APIServer
	r.ID = f.ID
	r.Name = f.Name
	r.Primary = f.Primary
	r.Runtime = f.Runtime
	r.Version = f.Version
	return r
}
func Convert_Cluster_ClusterCommon(f *orm.Cluster) *ClusterCommon {
	if f == nil {
		return nil
	}
	var r ClusterCommon
	r.APIServer = f.APIServer
	r.ID = f.ID
	r.Name = f.Name
	r.Primary = f.Primary
	r.Runtime = f.Runtime
	r.Version = f.Version
	return &r
}
func Convert_ClusterCommon_Cluster_slice(arr []*ClusterCommon) []*orm.Cluster {
	r := []*orm.Cluster{}
	for _, u := range arr {
		r = append(r, Convert_ClusterCommon_Cluster(u))
	}
	return r
}

func Convert_Cluster_ClusterCommon_slice(arr []*orm.Cluster) []*ClusterCommon {
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

func (ul *ChartRepoCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ChartRepoList{}
	return ul.objectlist
}

func (ul *ChartRepoCommonList) Data() []*ChartRepoCommon {
	if ul.data != nil {
		return ul.data.([]*ChartRepoCommon)
	}
	us := ul.objectlist.(*orm.ChartRepoList)
	tmp := Convert_ChartRepo_ChartRepoCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ChartRepoCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ChartRepoCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ChartRepoCommon_ChartRepo(r)
	}
	return r.object
}

func (u *ChartRepoCommon) Data() *ChartRepoCommon {
	if u.data != nil {
		return u.data.(*ChartRepoCommon)
	}
	tmp := Convert_ChartRepo_ChartRepoCommon(u.object.(*orm.ChartRepo))
	u.data = tmp
	return tmp
}

func (u *ChartRepoCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_ChartRepoCommon_ChartRepo(f *ChartRepoCommon) *orm.ChartRepo {
	r := &orm.ChartRepo{}
	if f == nil {
		return nil
	}
	f.object = r
	r.ID = f.ID
	r.LastSync = f.LastSync
	r.Name = f.Name
	r.SyncMessage = f.SyncMessage
	r.SyncStatus = f.SyncStatus
	r.URL = f.URL
	return r
}
func Convert_ChartRepo_ChartRepoCommon(f *orm.ChartRepo) *ChartRepoCommon {
	if f == nil {
		return nil
	}
	var r ChartRepoCommon
	r.ID = f.ID
	r.LastSync = f.LastSync
	r.Name = f.Name
	r.SyncMessage = f.SyncMessage
	r.SyncStatus = f.SyncStatus
	r.URL = f.URL
	return &r
}
func Convert_ChartRepoCommon_ChartRepo_slice(arr []*ChartRepoCommon) []*orm.ChartRepo {
	r := []*orm.ChartRepo{}
	for _, u := range arr {
		r = append(r, Convert_ChartRepoCommon_ChartRepo(u))
	}
	return r
}

func Convert_ChartRepo_ChartRepoCommon_slice(arr []*orm.ChartRepo) []*ChartRepoCommon {
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

func (ul *AuthSourceCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.AuthSourceList{}
	return ul.objectlist
}

func (ul *AuthSourceCommonList) Data() []*AuthSourceCommon {
	if ul.data != nil {
		return ul.data.([]*AuthSourceCommon)
	}
	us := ul.objectlist.(*orm.AuthSourceList)
	tmp := Convert_AuthSource_AuthSourceCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *AuthSourceCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *AuthSourceCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_AuthSourceCommon_AuthSource(r)
	}
	return r.object
}

func (u *AuthSourceCommon) Data() *AuthSourceCommon {
	if u.data != nil {
		return u.data.(*AuthSourceCommon)
	}
	tmp := Convert_AuthSource_AuthSourceCommon(u.object.(*orm.AuthSource))
	u.data = tmp
	return tmp
}

func (u *AuthSourceCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_AuthSourceCommon_AuthSource(f *AuthSourceCommon) *orm.AuthSource {
	r := &orm.AuthSource{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Config = f.Config
	r.CreatedAt = f.CreatedAt
	r.Enabled = f.Enabled
	r.ID = f.ID
	r.Kind = f.Kind
	r.Name = f.Name
	r.TokenType = f.TokenType
	r.UpdatedAt = f.UpdatedAt
	return r
}
func Convert_AuthSource_AuthSourceCommon(f *orm.AuthSource) *AuthSourceCommon {
	if f == nil {
		return nil
	}
	var r AuthSourceCommon
	r.Config = f.Config
	r.CreatedAt = f.CreatedAt
	r.Enabled = f.Enabled
	r.ID = f.ID
	r.Kind = f.Kind
	r.Name = f.Name
	r.TokenType = f.TokenType
	r.UpdatedAt = f.UpdatedAt
	return &r
}
func Convert_AuthSourceCommon_AuthSource_slice(arr []*AuthSourceCommon) []*orm.AuthSource {
	r := []*orm.AuthSource{}
	for _, u := range arr {
		r = append(r, Convert_AuthSourceCommon_AuthSource(u))
	}
	return r
}

func Convert_AuthSource_AuthSourceCommon_slice(arr []*orm.AuthSource) []*AuthSourceCommon {
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

func (ul *AuditLogCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.AuditLogList{}
	return ul.objectlist
}

func (ul *AuditLogCommonList) Data() []*AuditLogCommon {
	if ul.data != nil {
		return ul.data.([]*AuditLogCommon)
	}
	us := ul.objectlist.(*orm.AuditLogList)
	tmp := Convert_AuditLog_AuditLogCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *AuditLogCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *AuditLogCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_AuditLogCommon_AuditLog(r)
	}
	return r.object
}

func (u *AuditLogCommon) Data() *AuditLogCommon {
	if u.data != nil {
		return u.data.(*AuditLogCommon)
	}
	tmp := Convert_AuditLog_AuditLogCommon(u.object.(*orm.AuditLog))
	u.data = tmp
	return tmp
}

func (u *AuditLogCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_AuditLogCommon_AuditLog(f *AuditLogCommon) *orm.AuditLog {
	r := &orm.AuditLog{}
	if f == nil {
		return nil
	}
	f.object = r
	r.Action = f.Action
	r.ClientIP = f.ClientIP
	r.CreatedAt = f.CreatedAt
	r.DeletedAt = f.DeletedAt
	r.ID = f.ID
	r.Labels = f.Labels
	r.Module = f.Module
	r.Name = f.Name
	r.RawData = f.RawData
	r.Success = f.Success
	r.Tenant = f.Tenant
	r.UpdatedAt = f.UpdatedAt
	r.Username = f.Username
	return r
}
func Convert_AuditLog_AuditLogCommon(f *orm.AuditLog) *AuditLogCommon {
	if f == nil {
		return nil
	}
	var r AuditLogCommon
	r.Action = f.Action
	r.ClientIP = f.ClientIP
	r.CreatedAt = f.CreatedAt
	r.DeletedAt = f.DeletedAt
	r.ID = f.ID
	r.Labels = f.Labels
	r.Module = f.Module
	r.Name = f.Name
	r.RawData = f.RawData
	r.Success = f.Success
	r.Tenant = f.Tenant
	r.UpdatedAt = f.UpdatedAt
	r.Username = f.Username
	return &r
}
func Convert_AuditLogCommon_AuditLog_slice(arr []*AuditLogCommon) []*orm.AuditLog {
	r := []*orm.AuditLog{}
	for _, u := range arr {
		r = append(r, Convert_AuditLogCommon_AuditLog(u))
	}
	return r
}

func Convert_AuditLog_AuditLogCommon_slice(arr []*orm.AuditLog) []*AuditLogCommon {
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

func (ul *ApplicationDetailList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ApplicationList{}
	return ul.objectlist
}

func (ul *ApplicationDetailList) Data() []*ApplicationDetail {
	if ul.data != nil {
		return ul.data.([]*ApplicationDetail)
	}
	us := ul.objectlist.(*orm.ApplicationList)
	tmp := Convert_Application_ApplicationDetail_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ApplicationDetailList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ApplicationDetail) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ApplicationDetail_Application(r)
	}
	return r.object
}

func (u *ApplicationDetail) Data() *ApplicationDetail {
	if u.data != nil {
		return u.data.(*ApplicationDetail)
	}
	tmp := Convert_Application_ApplicationDetail(u.object.(*orm.Application))
	u.data = tmp
	return tmp
}

func (u *ApplicationDetail) DataPtr() interface{} {
	return u.Data()
}

func Convert_ApplicationDetail_Application(f *ApplicationDetail) *orm.Application {
	r := &orm.Application{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.Creator = f.Creator
	r.Enabled = f.Enabled
	r.Environment = Convert_EnvironmentCommon_Environment(f.Environment)
	r.EnvironmentID = f.EnvironmentID
	r.ID = f.ID
	r.Images = f.Images
	r.Kind = f.Kind
	r.Labels = f.Labels
	r.Manifest = f.Manifest
	r.Name = f.Name
	r.Project = Convert_ProjectCommon_Project(f.Project)
	r.ProjectID = f.ProjectID
	r.Remark = f.Remark
	r.UpdatedAt = f.UpdatedAt
	return r
}
func Convert_Application_ApplicationDetail(f *orm.Application) *ApplicationDetail {
	if f == nil {
		return nil
	}
	var r ApplicationDetail
	r.CreatedAt = f.CreatedAt
	r.Creator = f.Creator
	r.Enabled = f.Enabled
	r.Environment = Convert_Environment_EnvironmentCommon(f.Environment)
	r.EnvironmentID = f.EnvironmentID
	r.ID = f.ID
	r.Images = f.Images
	r.Kind = f.Kind
	r.Labels = f.Labels
	r.Manifest = f.Manifest
	r.Name = f.Name
	r.Project = Convert_Project_ProjectCommon(f.Project)
	r.ProjectID = f.ProjectID
	r.Remark = f.Remark
	r.UpdatedAt = f.UpdatedAt
	return &r
}
func Convert_ApplicationDetail_Application_slice(arr []*ApplicationDetail) []*orm.Application {
	r := []*orm.Application{}
	for _, u := range arr {
		r = append(r, Convert_ApplicationDetail_Application(u))
	}
	return r
}

func Convert_Application_ApplicationDetail_slice(arr []*orm.Application) []*ApplicationDetail {
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

func (ul *ApplicationCommonList) Object() client.ObjectListIface {
	if ul.objectlist != nil {
		return ul.objectlist
	}
	ul.objectlist = &orm.ApplicationList{}
	return ul.objectlist
}

func (ul *ApplicationCommonList) Data() []*ApplicationCommon {
	if ul.data != nil {
		return ul.data.([]*ApplicationCommon)
	}
	us := ul.objectlist.(*orm.ApplicationList)
	tmp := Convert_Application_ApplicationCommon_slice(us.Items)
	ul.data = tmp
	return tmp
}

func (ul *ApplicationCommonList) DataPtr() interface{} {
	return ul.Data()
}

func (r *ApplicationCommon) Object() client.Object {
	if r.object != nil {
		return r.object
	} else {
		r.object = Convert_ApplicationCommon_Application(r)
	}
	return r.object
}

func (u *ApplicationCommon) Data() *ApplicationCommon {
	if u.data != nil {
		return u.data.(*ApplicationCommon)
	}
	tmp := Convert_Application_ApplicationCommon(u.object.(*orm.Application))
	u.data = tmp
	return tmp
}

func (u *ApplicationCommon) DataPtr() interface{} {
	return u.Data()
}

func Convert_ApplicationCommon_Application(f *ApplicationCommon) *orm.Application {
	r := &orm.Application{}
	if f == nil {
		return nil
	}
	f.object = r
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.Name = f.Name
	r.UpdatedAt = f.UpdatedAt
	return r
}
func Convert_Application_ApplicationCommon(f *orm.Application) *ApplicationCommon {
	if f == nil {
		return nil
	}
	var r ApplicationCommon
	r.CreatedAt = f.CreatedAt
	r.ID = f.ID
	r.Name = f.Name
	r.UpdatedAt = f.UpdatedAt
	return &r
}
func Convert_ApplicationCommon_Application_slice(arr []*ApplicationCommon) []*orm.Application {
	r := []*orm.Application{}
	for _, u := range arr {
		r = append(r, Convert_ApplicationCommon_Application(u))
	}
	return r
}

func Convert_Application_ApplicationCommon_slice(arr []*orm.Application) []*ApplicationCommon {
	r := []*ApplicationCommon{}
	for _, u := range arr {
		r = append(r, Convert_Application_ApplicationCommon(u))
	}
	return r
}
