package alerthandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/alertmanager/pkg/labels"
	alerttypes "github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
)

const (
	allNamespace = "_all"
)

// Deprecated. use cli.Extend().ListSilences instead
func listSilences(ctx context.Context, namespace string, cli agents.Client) ([]*alerttypes.Silence, error) {
	silences := []*alerttypes.Silence{}
	url := "/custom/alertmanager/v1/silence"
	if namespace != "" {
		url = fmt.Sprintf(`%s?filter=%s="%s"`, url, prometheus.AlertNamespaceLabel, namespace)
	}
	if err := cli.DoRequest(ctx, agents.Request{
		Path: url,
		Into: agents.WrappedResponse(&silences),
	}); err != nil {
		return nil, err
	}

	// 只返回活跃的
	var ret []*alerttypes.Silence
	for _, v := range silences {
		if v.Status.State == alerttypes.SilenceStateActive && !strings.HasPrefix(v.Comment, silenceCommentPrefix) {
			ret = append(ret, v)
		}
	}
	return ret, nil
}

func getSilence(ctx context.Context, namespace, alertName string, cli agents.Client) (*alerttypes.Silence, error) {
	silences, err := listSilences(ctx, namespace, cli)
	if err != nil {
		return nil, err
	}

	// 只返回活跃的
	actives := []*alerttypes.Silence{}
	for _, silence := range silences {
		if silence.Status.State == alerttypes.SilenceStateActive &&
			silence.Matchers.Matches(model.LabelSet{
				prometheus.AlertNamespaceLabel: model.LabelValue(namespace),
				prometheus.AlertNameLabel:      model.LabelValue(alertName),
			}) { // 名称匹配
			actives = append(actives, silence)
		}
	}
	if len(actives) == 0 {
		return nil, nil
	}
	if len(actives) > 1 {
		return nil, errors.New("too many silences")
	}

	return actives[0], nil
}

func createSilenceIfNotExist(ctx context.Context, namespace, alertName string, cli agents.Client) error {
	silence, err := getSilence(ctx, namespace, alertName, cli)
	if err != nil {
		return err
	}
	// 不存在，创建
	if silence == nil {
		silence = &alerttypes.Silence{
			Comment:   fmt.Sprintf("silence for %s", alertName),
			CreatedBy: alertName,
			Matchers: labels.Matchers{
				&labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  prometheus.AlertNamespaceLabel,
					Value: namespace,
				},
				&labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  prometheus.AlertNameLabel,
					Value: alertName,
				},
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().AddDate(1000, 0, 0), // 100年
		}

		// create
		return cli.DoRequest(ctx, agents.Request{
			Method: http.MethodPost,
			Path:   "/custom/alertmanager/v1/silence/_/actions/create",
			Body:   silence,
		})
	}
	return nil
}

func deleteSilenceIfExist(ctx context.Context, namespace, alertName string, cli agents.Client) error {
	silence, err := getSilence(ctx, namespace, alertName, cli)
	if err != nil {
		return err
	}
	// 存在，删除
	if silence != nil {
		values := url.Values{}
		values.Add("id", silence.ID)

		return cli.DoRequest(ctx, agents.Request{
			Method: http.MethodDelete,
			Path:   "/custom/alertmanager/v1/silence/_/actions/delete",
			Query:  values,
		})
	}
	return nil
}
