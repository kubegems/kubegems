package promql

import (
	"fmt"

	"github.com/prometheus/prometheus/pkg/labels"
)

// MatchType is an enum for label matching types.
type MatchType string

// Possible MatchTypes.
const (
	MatchEqual     MatchType = "="
	MatchNotEqual            = "!="
	MatchRegexp              = "=~"
	MatchNotRegexp           = "!~"
)

type LabelMatcher struct {
	Type  MatchType `json:"type"`
	Name  string    `json:"name"`
	Value string    `json:"value"`
}

func (m *LabelMatcher) ToPromqlLabelMatcher() *labels.Matcher {
	ret := labels.Matcher{
		Name:  m.Name,
		Value: m.Value,
	}
	switch m.Type {
	case MatchEqual:
		ret.Type = labels.MatchEqual
	case MatchNotEqual:
		ret.Type = labels.MatchNotEqual
	case MatchRegexp:
		ret.Type = labels.MatchRegexp
	case MatchNotRegexp:
		ret.Type = labels.MatchNotRegexp
	}
	return &ret
}

func (m *LabelMatcher) String() string {
	return fmt.Sprintf("%s%s%q", m.Name, m.Type, m.Value)
}
