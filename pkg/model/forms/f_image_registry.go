package forms

import "time"

// +genform object:Registry
type RegistryCommon struct {
	BaseForm
	ID         uint           `json:"id,omitempty"`
	Name       string         `json:"registryName,omitempty"`
	Address    string         `json:"address,omitempty"`
	UpdateTime *time.Time     `json:"updateTime,omitempty"`
	Creator    *UserCommon    `json:"creator,omitempty"`
	CreatorID  uint           `json:"creatorID,omitempty"`
	Project    *ProjectCommon `json:"project,omitempty"`
	ProjectID  uint           `json:"projectID,omitempty"`
	IsDefault  bool           `json:"isDefault,omitempty"`
}

// +genform object:Registry
type RegistryDetail struct {
	BaseForm
	ID         uint           `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Address    string         `json:"address,omitempty"`
	Username   string         `json:"username,omitempty"`
	Password   string         `json:"password,omitempty"`
	Creator    *UserCommon    `json:"creator,omitempty"`
	UpdateTime *time.Time     `json:"updateTime,omitempty"`
	CreatorID  uint           `json:"creatorID,omitempty"`
	Project    *ProjectCommon `json:"project,omitempty"`
	ProjectID  uint           `json:"projectID,omitempty"`
	IsDefault  bool           `json:"isDefault,omitempty"`
}
