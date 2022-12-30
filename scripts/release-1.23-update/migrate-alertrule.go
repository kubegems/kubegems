package main

import (
	"context"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers/observability"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/observe"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
)

func exportOldAlertRulesToDB(ctx context.Context, cs agents.ClientSet, db *database.Database) error {
	alertrules := []*models.AlertRule{}
	if err := cs.ExecuteInEachCluster(ctx, func(ctx context.Context, cli agents.Client) error {
		observecli := observe.NewClient(cli, db.DB())
		monitorAlertRules, err := observecli.ListMonitorAlertRules(ctx, "", false, db.NewPromqlTplMapperFromDB().FindPromqlTpl)
		if err != nil {
			log.Warnf("ListMonitorAlertRules in cluster: %s, err: %v", cli.Name(), err)
		}
		loggingAlertRules, err := observecli.ListLoggingAlertRules(ctx, "", false)
		if err != nil {
			log.Warnf("ListLoggingAlertRules in cluster: %s, err: %v", cli.Name(), err)
		}
		for _, v := range monitorAlertRules {
			alertrules = append(alertrules, convertMonitorAlertRule(cli.Name(), v))
		}
		for _, v := range loggingAlertRules {
			alertrules = append(alertrules, convertLoggingAlertRule(cli.Name(), v))
		}
		return nil
	}); err != nil {
		return err
	}
	for _, v := range alertrules {
		if err := observability.SetReceivers(v, db.DB()); err != nil {
			log.Warnf("SetReceivers for: %s failed: %v", v.FullName(), err)
			continue
		}
		if err := db.DB().Create(v).Error; err != nil {
			log.Warnf("create alertrule %s in db failed: %v", v.FullName(), err)
		}
		log.Info("export alertrule success", "name", v.FullName())
	}
	return nil
}

func convertMonitorAlertRule(cluster string, monitorRule observe.MonitorAlertRule) *models.AlertRule {
	ret := &models.AlertRule{
		Cluster:       cluster,
		Namespace:     monitorRule.Namespace,
		Name:          monitorRule.Name,
		AlertType:     prometheus.AlertTypeMonitor,
		Expr:          monitorRule.Expr,
		Message:       monitorRule.Message,
		For:           monitorRule.For,
		InhibitLabels: monitorRule.InhibitLabels,
		// AlertLevels: ,
		// Receivers: ,
		// PromqlGenerator: ,
		IsOpen: monitorRule.IsOpen,
	}
	for _, level := range monitorRule.AlertLevels {
		ret.AlertLevels = append(ret.AlertLevels, models.AlertLevel{
			CompareOp:    level.CompareOp,
			CompareValue: level.CompareValue,
			Severity:     level.Severity,
		})
	}
	for _, rec := range monitorRule.Receivers {
		ret.Receivers = append(ret.Receivers, &models.AlertReceiver{
			AlertChannelID: rec.AlertChannel.ID,
			Interval:       rec.Interval,
		})
	}
	if monitorRule.PromqlGenerator != nil {
		ret.PromqlGenerator = &models.PromqlGenerator{
			Scope:    monitorRule.PromqlGenerator.Scope,
			Resource: monitorRule.PromqlGenerator.Resource,
			Rule:     monitorRule.PromqlGenerator.Rule,
			Unit:     monitorRule.PromqlGenerator.Unit,
		}
		for k, v := range monitorRule.PromqlGenerator.LabelPairs {
			ret.PromqlGenerator.LabelMatchers = append(ret.PromqlGenerator.LabelMatchers, promql.LabelMatcher{
				Type:  promql.MatchEqual,
				Name:  k,
				Value: v,
			})
		}
	}
	return ret
}

func convertLoggingAlertRule(cluster string, loggingRule observe.LoggingAlertRule) *models.AlertRule {
	ret := &models.AlertRule{
		Cluster:       cluster,
		Namespace:     loggingRule.Namespace,
		Name:          loggingRule.Name,
		AlertType:     prometheus.AlertTypeMonitor,
		Expr:          loggingRule.Expr,
		Message:       loggingRule.Message,
		For:           loggingRule.For,
		InhibitLabels: loggingRule.InhibitLabels,
		// AlertLevels: ,
		// Receivers: ,
		// LogqlGenerator: ,
		IsOpen: loggingRule.IsOpen,
	}
	for _, level := range loggingRule.AlertLevels {
		ret.AlertLevels = append(ret.AlertLevels, models.AlertLevel{
			CompareOp:    level.CompareOp,
			CompareValue: level.CompareValue,
			Severity:     level.Severity,
		})
	}
	for _, rec := range loggingRule.Receivers {
		ret.Receivers = append(ret.Receivers, &models.AlertReceiver{
			AlertChannelID: rec.AlertChannel.ID,
			Interval:       rec.Interval,
		})
	}
	if loggingRule.LogqlGenerator != nil {
		ret.LogqlGenerator = &models.LogqlGenerator{
			Duration: loggingRule.LogqlGenerator.Duration,
			Match:    loggingRule.LogqlGenerator.Match,
		}
		for k, v := range loggingRule.LogqlGenerator.LabelPairs {
			ret.LogqlGenerator.LabelMatchers = append(ret.LogqlGenerator.LabelMatchers, promql.LabelMatcher{
				Type:  promql.MatchEqual,
				Name:  k,
				Value: v,
			})
		}
	}
	return ret
}
