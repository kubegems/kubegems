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

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/workflow"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StrategyDeploymentControl struct {
	Command string                 `json:"command,omitempty"` // 指令名称 in(pause,restart,retry,promote,terminate,undo)
	Args    map[string]interface{} `json:"args,omitempty"`    // 一些指令可能会携带的参数,比如 undo {reversion=1} ; promote {full=true}
}

// @Tags         StrategyDeployment
// @Summary      更新过程中的控制
// @Description  更新控制
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @param        environment_id  path      int                                   true  "environment id"
// @Param        name            path      string                                true  "applicationname"
// @param        body            body      StrategyDeploymentControl             true  "command"
// @Success      200             {object}  handlers.ResponseStruct{Data=object}  "-"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/application/{name}/strategydeploycontrol [post]
// @Security     JWT
func (h *ApplicationHandler) StrategyDeploymentControl(c *gin.Context) {
	req := &StrategyDeploymentControl{}

	ctrlfunc := func(ctx context.Context, store GitStore, cli agents.Client, namespace string, ref PathRef) (interface{}, error) {
		log.FromContextOrDiscard(ctx).Info("strategy deployment control", "command", req.Command, "args", req.Args)

		deployment, err := ParseMainDeployment(ctx, store)
		if err != nil {
			return nil, err
		}
		name := deployment.Name

		rollout := &rolloutsv1alpha1.Rollout{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
		switch req.Command {
		case "pause":
			// set pause to true
			// https://github.com/argoproj/argo-rollouts/blob/v1.1.0/pkg/kubectl-argo-rollouts/cmd/pause/pause.go#L35
			patch := client.RawPatch(types.MergePatchType, []byte(`{"spec":{"paused":true}}`))
			if err := cli.Patch(ctx, rollout, patch); err != nil {
				return nil, err
			}
			return rollout, nil
		case "restart":
			// https://github.com/argoproj/argo-rollouts/blob/v1.1.0/pkg/kubectl-argo-rollouts/cmd/restart/restart.go#L70
			patchContent := fmt.Sprintf(`{"spec": {"restartAt": "%s"}}`, time.Now().UTC().Format(time.RFC3339))
			patch := client.RawPatch(types.MergePatchType, []byte(patchContent))
			if err := cli.Patch(ctx, rollout, patch); err != nil {
				return nil, err
			}
			return rollout, nil
		case "retry":
			// https://github.com/argoproj/argo-rollouts/blob/v1.1.0/pkg/kubectl-argo-rollouts/cmd/retry/retry.go#L84
			patch := client.RawPatch(types.MergePatchType, []byte(`{"status":{"abort":false}}`))
			if err := cli.Status().Patch(ctx, rollout, patch); err != nil {
				return nil, err
			}
			return rollout, nil
		case "promote":
			// 兼容两种类型
			full := false
			switch v := req.Args["full"].(type) {
			case bool:
				full = v
			case string:
				full, _ = strconv.ParseBool(v)
			}
			return PromoteRollout(ctx, cli, namespace, name, full)
		case "terminate":
			// TODO:
		case "undo":
			revison, _ := req.Args["revision"].(string)
			if revison == "" {
				return nil, fmt.Errorf("undo must have revision specified")
			}
			if issync, _ := strconv.ParseBool(c.Query("sync")); issync {
				if err := h.ApplicationProcessor.Undo(ctx, ref, revison); err != nil {
					return nil, err
				}
				_ = h.ApplicationProcessor.Sync(ctx, ref)
			} else {
				h.asyncUndo(ctx, ref, revison)
			}
		default:
		}
		return "ok", nil
	}
	h.LocalAndRemoteCliFunc(c, req, ctrlfunc, "")
}

func (h *ApplicationHandler) asyncUndo(ctx context.Context, ref PathRef, targetrev string) {
	steps := []workflow.Step{
		{
			Name:     "undo",
			Function: TaskFunction_Application_Undo,
			Args:     workflow.ArgsOf(ref, targetrev),
		},
		{
			Name:     "sync",
			Function: TaskFunction_Application_Sync,
			Args:     workflow.ArgsOf(ref),
		},
		// {
		// 	Name:     "wait-sync",
		// 	Function: TaskFunction_Application_WaitSync,
		// 	Args:     workflow.ArgsOf(ref),
		// },
	}
	_ = h.Task.Processor.SubmitTask(ctx, ref, "undo", steps)
}

var ignorelabels = []string{
	// deployment
	appsv1.DefaultDeploymentUniqueLabelKey,
	// rollouts
	rolloutsv1alpha1.DefaultRolloutUniqueLabelKey,
}

// https://github.com/kubernetes/kubernetes/blob/release-1.20/staging/src/k8s.io/kubectl/pkg/polymorphichelpers/rollback.go#L99
func getDeploymentPatch(rs appsv1.ReplicaSet) (client.Patch, error) {
	// ignore  labels
	for _, k := range ignorelabels {
		delete(rs.Spec.Template.Labels, k)
	}
	// Create a patch of the Deployment that replaces spec.template
	patch, err := json.Marshal([]interface{}{
		map[string]interface{}{
			"op":    "replace",
			"path":  "/spec/template",
			"value": rs.Spec.Template,
		},
	})
	return client.RawPatch(types.JSONPatchType, patch), err
}

const (
	unpausePatch                                = `{"spec":{"paused":false}}`
	clearPauseConditionsPatch                   = `{"status":{"pauseConditions":null}}`
	unpauseAndClearPauseConditionsPatch         = `{"spec":{"paused":false},"status":{"pauseConditions":null}}`
	promoteFullPatch                            = `{"status":{"promoteFull":true}}`
	clearPauseConditionsPatchWithStep           = `{"status":{"pauseConditions":null, "currentStepIndex":%d}}`
	unpauseAndClearPauseConditionsPatchWithStep = `{"spec":{"paused":false},"status":{"pauseConditions":null, "currentStepIndex":%d}}`
)

// copy from: https://github.com/argoproj/argo-rollouts/blob/v1.1.0/pkg/kubectl-argo-rollouts/cmd/promote/promote.go#L92
// 默认下一步，full则跳过所有
func PromoteRollout(ctx context.Context, cli client.Client, namespace, name string, full bool) (*rolloutsv1alpha1.Rollout, error) {
	ro := &rolloutsv1alpha1.Rollout{}
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, ro); err != nil {
		return nil, err
	}

	// This function is intended to be compatible with Rollouts v0.9 and Rollouts v0.10+, the latter
	// of which uses CRD status subresources. When using status subresource, status must be updated
	// separately from spec. Since we don't know which version is installed in the cluster, we
	// attempt status patching first. If it errors with NotFound, it indicates that status
	// subresource is not used (v0.9), at which point we need to use the unified patch that updates
	// both spec and status. Otherwise, we proceed with a spec only patch.
	specPatch, statusPatch, unifiedPatch := getPatches(ro, full)
	if statusPatch != nil {
		if err := cli.Status().Patch(ctx, ro, client.RawPatch(types.MergePatchType, statusPatch)); err != nil {
			// NOTE: in the future, we can simply return error here, if we wish to drop support for v0.9
			if !k8serrors.IsNotFound(err) {
				return nil, err
			}
			// we got a NotFound error. status subresource is not being used, so perform unifiedPatch
			specPatch = unifiedPatch
		}
	}
	if specPatch != nil {
		if err := cli.Patch(ctx, ro, client.RawPatch(types.MergePatchType, specPatch)); err != nil {
			return nil, err
		}
	}
	return ro, nil
}

func getPatches(rollout *rolloutsv1alpha1.Rollout, full bool) ([]byte, []byte, []byte) {
	var specPatch, statusPatch, unifiedPatch []byte

	if full {
		if rollout.Status.CurrentPodHash != rollout.Status.StableRS {
			statusPatch = []byte(promoteFullPatch)
		}
		return specPatch, statusPatch, unifiedPatch
	}

	unifiedPatch = []byte(unpauseAndClearPauseConditionsPatch)
	if rollout.Spec.Paused {
		specPatch = []byte(unpausePatch)
	}
	if len(rollout.Status.PauseConditions) > 0 {
		statusPatch = []byte(clearPauseConditionsPatch)
	} else if rollout.Spec.Strategy.Canary != nil {
		// we only want to clear pause conditions, or increment step index (never both)
		// this else block covers the case of promoting a rollout when it is in the middle of
		// running analysis/experiment
		// TODO: we currently do not handle promotion of two analysis steps in a row properly
		_, index := GetCurrentCanaryStep(rollout)
		// At this point, the controller knows that the rollout is a canary with steps and GetCurrentCanaryStep returns 0 if
		// the index is not set in the rollout
		if index != nil {
			if *index < int32(len(rollout.Spec.Strategy.Canary.Steps)) {
				*index++
			}
			statusPatch = []byte(fmt.Sprintf(clearPauseConditionsPatchWithStep, *index))
			unifiedPatch = []byte(fmt.Sprintf(unpauseAndClearPauseConditionsPatchWithStep, *index))
		}
	}
	return specPatch, statusPatch, unifiedPatch
}

// GetCurrentCanaryStep returns the current canary step. If there are no steps or the rollout
// has already executed the last step, the func returns nil
func GetCurrentCanaryStep(rollout *rolloutsv1alpha1.Rollout) (*rolloutsv1alpha1.CanaryStep, *int32) {
	if rollout.Spec.Strategy.Canary == nil || len(rollout.Spec.Strategy.Canary.Steps) == 0 {
		return nil, nil
	}
	currentStepIndex := int32(0)
	if rollout.Status.CurrentStepIndex != nil {
		currentStepIndex = *rollout.Status.CurrentStepIndex
	}
	if len(rollout.Spec.Strategy.Canary.Steps) <= int(currentStepIndex) {
		return nil, &currentStepIndex
	}
	return &rollout.Spec.Strategy.Canary.Steps[currentStepIndex], &currentStepIndex
}
