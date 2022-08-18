package templates

import (
	"fmt"

	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
)

type PromqlTpl struct {
	ScopeID          uint   `json:"scopeID"`
	ScopeName        string `json:"scopeName"`
	ScopeShowName    string `json:"scopeShowName"`
	ResourceID       uint   `json:"resourceID"`
	ResourceName     string `json:"resourceName"`
	ResourceShowName string `json:"resourceShowName"`
	RuleID           uint   `json:"ruleID"`
	RuleName         string `json:"ruleName"`
	RuleShowName     string `json:"ruleShowName"`

	TenantID *uint `json:"tenantID"`

	Namespaced bool                    `json:"namespaced"`
	Expr       string                  `json:"expr"`
	Unit       string                  `json:"unit"`
	Labels     gormdatatypes.JSONSlice `json:"labels"`
}

type TplGetter func(scope, resource, rule string) (*PromqlTpl, error)

func (tpl *PromqlTpl) String() string {
	return fmt.Sprintf("%s.%s.%s", tpl.ScopeName, tpl.ResourceName, tpl.RuleName)
}

type PromqlTplMapper struct {
	M   map[string]*PromqlTpl
	Err error
}

func (m *PromqlTplMapper) FindPromqlTpl(scope, resource, rule string) (*PromqlTpl, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	key := fmt.Sprintf("%s.%s.%s", scope, resource, rule)
	ret, ok := m.M[key]
	if !ok {
		return nil, fmt.Errorf("promql tpl: %s not found", key)
	}
	return ret, nil
}
