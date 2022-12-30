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

package task

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers/observability"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/workflow"
	"sigs.k8s.io/yaml"
)

type AlertRuleSyncTasker struct {
	DB *database.Database
	cs *agents.ClientSet
}

const (
	TaskFunction_AlertRuleStateSync   = "alertrule-state-sync"
	TaskFunction_AlertRuleConfigCheck = "alertrule-config-check"
)

func (t *AlertRuleSyncTasker) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		TaskFunction_AlertRuleStateSync:   t.SyncAlertRuleStatus,
		TaskFunction_AlertRuleConfigCheck: t.CheckAlertRuleConfig,
	}
}

func (s *AlertRuleSyncTasker) Crontasks() map[string]Task {
	return map[string]Task{
		"@every 2m": {
			Name:  "alertrule state sync",
			Group: "alertrule",
			Steps: []workflow.Step{{Function: TaskFunction_AlertRuleStateSync}},
		},
		"@every 10m": {
			Name:  "alertrule config check",
			Group: "alertrule",
			Steps: []workflow.Step{{Function: TaskFunction_AlertRuleConfigCheck}},
		},
	}
}

func (t *AlertRuleSyncTasker) SyncAlertRuleStatus(ctx context.Context) error {
	alertrules := []*models.AlertRule{}
	if err := t.DB.DB().Find(&alertrules).Error; err != nil {
		return err
	}
	alertStatusMap := sync.Map{} // key: cluster/gems-namespace-name
	if err := t.cs.ExecuteInEachCluster(ctx, func(ctx context.Context, cli agents.Client) error {
		promAlerts, err := cli.Extend().GetPromeAlertRules(ctx, "")
		if err != nil {
			log.Warnf("get prometheus alert failed in cluster: %s, err: %v", cli.Name(), err)
			return nil
		}
		lokiAlerts, err := cli.Extend().GetLokiAlertRules(ctx)
		if err != nil {
			log.Warnf("get loki alert failed in cluster: %s, err: %v", cli.Name(), err)
			return nil
		}
		for k, v := range promAlerts {
			alertStatusMap.Store(fmt.Sprintf("%s/%s", cli.Name(), k), v.State)
		}
		for k, v := range lokiAlerts {
			alertStatusMap.Store(fmt.Sprintf("%s/%s", cli.Name(), k), v.State)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, alertrule := range alertrules {
		state, ok := alertStatusMap.Load(fmt.Sprintf("%s/%s", alertrule.Cluster, prometheus.RealTimeAlertKey(alertrule.Namespace, alertrule.Name)))
		if ok {
			alertrule.State = state.(string)
		} else {
			alertrule.State = "inactive"
		}
		if err := t.DB.DB().Model(alertrule).Update("state", alertrule.State).Error; err != nil {
			return errors.Wrapf(err, "update alert rule %s state failed", alertrule.FullName())
		}
	}
	return nil
}

func (t *AlertRuleSyncTasker) CheckAlertRuleConfig(ctx context.Context) error {
	alertrules := []*models.AlertRule{}
	if err := t.DB.DB().Preload("Receivers.AlertChannel").Find(&alertrules).Error; err != nil {
		return err
	}
	k8sAlertCfg := sync.Map{}
	if err := t.cs.ExecuteInEachCluster(ctx, func(ctx context.Context, cli agents.Client) error {
		cfgs, err := observability.NewAlertRuleProcessor(cli, t.DB).GetK8sAlertCfg(ctx)
		if err != nil {
			log.Warnf("get k8s alert cfg failed in cluster: %s, err: %v", cli.Name(), err)
			return nil
		}
		for key, cfg := range cfgs {
			if _, ok := k8sAlertCfg.Load(key); ok {
				log.Warnf("duplicated alert rule: %s", key)
				continue
			}
			k8sAlertCfg.Store(key, cfg)
		}
		return nil
	}); err != nil {
		return err
	}
	for _, alertrule := range alertrules {
		cfgInDB := observability.K8sAlertCfg{
			RuleGroup:              observability.GenerateRuleGroup(alertrule),
			AlertmanagerConfigSpec: observability.GenerateAmcfgSpec(alertrule),
		}
		cfgInK8s, ok := k8sAlertCfg.Load(alertrule.FullName())
		if !ok {
			alertrule.K8sResourceStatus = alertCfgStatusError("k8s alert rule config lost")
		}
		diff := cmp.Diff(cfgInDB, cfgInK8s.(observability.K8sAlertCfg),
			cmpopts.EquateEmpty(),       // eg. slice nil equal to empty
			cmp.Comparer(compareRoutes), // compare for []apiextensionsv1.JSON
		)
		if diff == "" {
			alertrule.K8sResourceStatus = alertCfgStatusOK()
		} else {
			alertrule.K8sResourceStatus = alertCfgStatusError(diff)
			log.Warnf("alertrule: %s not matched, diff:\n%s", alertrule.FullName(), diff)
		}
		if err := t.DB.DB().Model(alertrule).Update("k8s_resource_status", alertrule.K8sResourceStatus).Error; err != nil {
			log.Warnf("update k8s_resource_status for alertrule: %s failed", alertrule.FullName())
		}
	}

	return nil
}

// routes order and content order may changed after umarshal, so we compare after marshal
func compareRoutes(a, b []apiextensionsv1.JSON) bool {
	bts1, _ := yaml.Marshal(a)
	bts2, _ := yaml.Marshal(b)
	return string(bts1) == string(bts2)
}

func alertCfgStatusOK() map[string]string {
	return map[string]string{
		"status": "ok",
	}
}

func alertCfgStatusError(reason string) map[string]string {
	return map[string]string{
		"status": "error",
		"reason": reason,
	}
}
