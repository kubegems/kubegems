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
	"os"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"kubegems.io/kubegems/pkg/installer/pluginmanager"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers/observability"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/workflow"
	"sigs.k8s.io/yaml"
)

type AlertRuleSyncTasker struct {
	DB *database.Database
	cs *agents.ClientSet
}

const (
	TaskFunction_SyncAlertRuleState   = "sync-alertrule-state"
	TaskFunction_CheckAlertRuleConfig = "check-alertrule-config"
	TaskFunction_SyncSystemAlertRule  = "sync-system-alertrule"
)

func (t *AlertRuleSyncTasker) ProvideFuntions() map[string]interface{} {
	return map[string]interface{}{
		TaskFunction_SyncAlertRuleState:   t.SyncAlertRuleState,
		TaskFunction_CheckAlertRuleConfig: t.CheckAlertRuleConfig,
		TaskFunction_SyncSystemAlertRule:  t.SyncSystemAlertRule,
	}
}

func (s *AlertRuleSyncTasker) Crontasks() map[string]Task {
	return map[string]Task{
		"@every 5m": {
			Name:  "sync alertrule state",
			Group: "alertrule",
			Steps: []workflow.Step{{Function: TaskFunction_SyncAlertRuleState}},
		},
		"@every 12h": {
			Name:  "check alertrule config",
			Group: "alertrule",
			Steps: []workflow.Step{{Function: TaskFunction_CheckAlertRuleConfig}},
		},
		"@daily": {
			Name:  "sync system alertrule",
			Group: "alertrule",
			Steps: []workflow.Step{{Function: TaskFunction_SyncSystemAlertRule}},
		},
	}
}

func (t *AlertRuleSyncTasker) SyncAlertRuleState(ctx context.Context) error {
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
		var newState string
		stateIf, ok := alertStatusMap.Load(fmt.Sprintf("%s/%s", alertrule.Cluster, prometheus.RealTimeAlertKey(alertrule.Namespace, alertrule.Name)))
		if ok {
			newState = stateIf.(string)
		} else {
			newState = "inactive"
		}
		if alertrule.State != newState {
			if err := t.DB.DB().Model(alertrule).Omit(clause.Associations).Update("state", newState).Error; err != nil {
				return errors.Wrapf(err, "update alert rule %s state failed", alertrule.FullName())
			}
		}
	}
	return nil
}

func (t *AlertRuleSyncTasker) CheckAlertRuleConfig(ctx context.Context) error {
	alertrules := []*models.AlertRule{}
	if err := t.DB.DB().Preload("Receivers.AlertChannel").Find(&alertrules).Error; err != nil {
		return err
	}
	allK8sAlertCfg := sync.Map{}
	if err := t.cs.ExecuteInEachCluster(ctx, func(ctx context.Context, cli agents.Client) error {
		cfgs, err := observability.NewAlertRuleProcessor(cli, t.DB).GetK8sAlertCfg(ctx)
		if err != nil {
			log.Warnf("get k8s alert cfg failed in cluster: %s, err: %v", cli.Name(), err)
			return nil
		}
		for key, cfg := range cfgs {
			if _, ok := allK8sAlertCfg.Load(key); ok {
				log.Warnf("duplicated alert rule: %s", key)
				continue
			}
			allK8sAlertCfg.Store(key, cfg)
		}
		return nil
	}); err != nil {
		return err
	}

	eg := errgroup.Group{}
	eg.SetLimit(5)
	for _, v := range alertrules {
		alertrule := v
		eg.Go(func() error {
			err := checkK8sAlertCfg(alertrule, &allK8sAlertCfg)
			if err == nil && alertrule.AlertType == prometheus.AlertTypeMonitor {
				err = t.checkExpr(ctx, alertrule)
			}
			newStatus := alertCfgStatus(err)
			if !cmp.Equal(alertrule.K8sResourceStatus, newStatus) {
				if err := t.DB.DB().Model(alertrule).Omit(clause.Associations).Update("k8s_resource_status", newStatus).Error; err != nil {
					log.Warnf("update k8s_resource_status for alertrule: %s failed", alertrule.FullName())
				}
			}
			return nil
		})
	}
	eg.Wait()

	return nil
}

func checkK8sAlertCfg(alertrule *models.AlertRule, k8sAlertCfgs *sync.Map) error {
	cfgInDB := observability.K8sAlertCfg{
		RuleGroup:              observability.GenerateRuleGroup(alertrule),
		AlertmanagerConfigSpec: observability.GenerateAmcfgSpec(alertrule),
	}
	cfgInK8s, ok := k8sAlertCfgs.Load(alertrule.FullName())
	if ok {
		diff := cmp.Diff(cfgInDB, cfgInK8s.(observability.K8sAlertCfg),
			cmpopts.EquateEmpty(),       // eg. slice nil equal to empty
			cmp.Comparer(compareRoutes), // compare for []apiextensionsv1.JSON
		)
		if diff != "" {
			return errors.Errorf(diff)
		}
	} else {
		return errors.Errorf("k8s alert rule config lost")
	}
	return nil
}

func (t *AlertRuleSyncTasker) checkExpr(ctx context.Context, alertrule *models.AlertRule) error {
	// check expr
	if alertrule.PromqlGenerator != nil {
		tpl, err := t.DB.FindPromqlTpl(alertrule.PromqlGenerator.Scope, alertrule.PromqlGenerator.Resource, alertrule.PromqlGenerator.Rule)
		if err != nil {
			return err
		}
		alertrule.PromqlGenerator.Tpl = tpl
	}
	generatedExpr, err := observability.GenerateExpr(alertrule)
	if err != nil {
		return err
	}
	if generatedExpr != alertrule.Expr {
		return errors.Errorf("generated expr:[%s] not equal to expr now: [%s]", generatedExpr, alertrule.Expr)
	}

	cli, err := t.cs.ClientOf(ctx, alertrule.Cluster)
	if err != nil {
		return err
	}
	vector, err := cli.Extend().PrometheusVector(ctx, alertrule.Expr)
	if err != nil {
		return err
	}
	// check result empty
	if vector.Len() == 0 {
		return errors.Errorf("query prometheus result is empty")
	}

	// check label namespace
	for _, v := range vector {
		if alertrule.Namespace != prometheus.GlobalAlertNamespace && v.Metric["namespace"] == "" {
			return errors.Errorf("query prometheus result should contains label [namespace]")
		}
	}
	return nil
}

func (t *AlertRuleSyncTasker) SyncSystemAlertRule(ctx context.Context) error {
	bts, err := os.ReadFile("config/system_alert.yaml")
	if err != nil {
		return errors.Wrap(err, "read system alert rule")
	}
	return t.cs.ExecuteInEachCluster(ctx, func(ctx context.Context, cli agents.Client) error {
		pm := &pluginmanager.PluginManager{Client: cli}
		plugin, err := pm.Get(ctx, "monitoring")
		if err != nil {
			log.Error(err, "get monitor plugin", "cluster", cli.Name())
			return nil
		}
		if plugin.Installed == nil || !plugin.Installed.Enabled {
			log.Errorf("monitor plugin not enabled in cluster: %s", cli.Name())
			return nil
		}

		sysRules := []*models.AlertRule{}
		if err := yaml.Unmarshal(bts, &sysRules); err != nil {
			return errors.Wrap(err, "unmarshal system alert rule")
		}
		p := observability.NewAlertRuleProcessor(cli, t.DB)
		for _, v := range sysRules {
			v.Cluster = cli.Name()
			synced, err := createOrUpdateSysAlertRule(ctx, p, v)
			if err != nil {
				log.Error(err, "sync system alertrule", "name", v.FullName())
				continue
			}
			if synced {
				log.Info("success to sync system alertrule", "name", v.FullName())
			}
		}
		return nil
	})
}

func createOrUpdateSysAlertRule(ctx context.Context, p *observability.AlertRuleProcessor, sysrule *models.AlertRule) (bool, error) {
	dbrule := &models.AlertRule{}
	if err := p.DBWithCtx(ctx).Preload("Receivers.AlertChannel").First(dbrule, "cluster = ? and namespace = ? and name = ?", sysrule.Cluster, sysrule.Namespace, sysrule.Name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := p.MutateAlertRule(ctx, sysrule); err != nil {
				return false, err
			}
			return true, p.CreateAlertRule(ctx, sysrule)
		} else {
			return false, errors.Wrapf(err, "get alertrule: %s", sysrule.FullName())
		}
	}
	if dbrule.K8sResourceStatus != nil && dbrule.K8sResourceStatus["status"] != "ok" {
		// only update when error
		if err := p.MutateAlertRule(ctx, sysrule); err != nil {
			return false, err
		}
		dbrule.AlertLevels = sysrule.AlertLevels
		dbrule.Expr = sysrule.Expr
		dbrule.For = sysrule.For
		dbrule.InhibitLabels = sysrule.InhibitLabels
		dbrule.LogqlGenerator = sysrule.LogqlGenerator
		dbrule.PromqlGenerator = sysrule.PromqlGenerator
		dbrule.Message = sysrule.Message
		return true, p.UpdateAlertRule(ctx, dbrule)
	}
	return false, nil
}

// routes order and content order may changed after umarshal, so we compare after marshal
func compareRoutes(a, b []apiextensionsv1.JSON) bool {
	bts1, _ := yaml.Marshal(a)
	bts2, _ := yaml.Marshal(b)
	return string(bts1) == string(bts2)
}

func alertCfgStatus(err error) gormdatatypes.JSONMap {
	if err != nil {
		return gormdatatypes.JSONMap{
			"status": "error",
			"reason": err.Error(),
		}
	}
	return gormdatatypes.JSONMap{
		"status": "ok",
	}
}
