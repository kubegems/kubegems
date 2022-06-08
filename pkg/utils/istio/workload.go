package istio

import (
	"sort"
	"strings"

	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/kubegems/pkg/log"
)

// COPY from kiali，有删改
func FetchWorkloads(pods []core_v1.Pod, repset []apps_v1.ReplicaSet, dep []apps_v1.Deployment, fulset []apps_v1.StatefulSet, daeset []apps_v1.DaemonSet, namespace, labelSelector string) (models.Workloads, error) {
	ws := models.Workloads{}
	// Key: name of controller; Value: type of controller
	controllers := map[string]string{}

	// Find controllers from pods
	for _, pod := range pods {
		if len(pod.OwnerReferences) != 0 {
			for _, ref := range pod.OwnerReferences {
				if ref.Controller != nil && *ref.Controller {
					if _, exist := controllers[ref.Name]; !exist {
						controllers[ref.Name] = ref.Kind
					} else {
						if controllers[ref.Name] != ref.Kind {
							controllers[ref.Name] = controllerPriority(controllers[ref.Name], ref.Kind)
						}
					}
				}
			}
		} else {
			if _, exist := controllers[pod.Name]; !exist {
				// Pod without controller
				controllers[pod.Name] = "Pod"
			}
		}
	}

	// Resolve ReplicaSets from Deployments
	// Resolve ReplicationControllers from DeploymentConfigs
	// Resolve Jobs from CronJobs
	for cname, ctype := range controllers {
		if ctype == kubernetes.ReplicaSetType {
			found := false
			iFound := -1
			for i, rs := range repset {
				if rs.Name == cname {
					iFound = i
					found = true
					break
				}
			}
			if found && len(repset[iFound].OwnerReferences) > 0 {
				for _, ref := range repset[iFound].OwnerReferences {
					if ref.Controller != nil && *ref.Controller {
						// Delete the child ReplicaSet and add the parent controller
						if _, exist := controllers[ref.Name]; !exist {
							controllers[ref.Name] = ref.Kind
						} else {
							if controllers[ref.Name] != ref.Kind {
								controllers[ref.Name] = controllerPriority(controllers[ref.Name], ref.Kind)
							}
						}
						delete(controllers, cname)
					}
				}
			}
		}
	}

	// Cornercase, check for controllers without pods, to show them as a workload
	var selector labels.Selector
	var selErr error
	if labelSelector != "" {
		selector, selErr = labels.Parse(labelSelector)
		if selErr != nil {
			log.Errorf("%s can not be processed as selector: %v", labelSelector, selErr)
		}
	}
	for _, d := range dep {
		selectorCheck := true
		if selector != nil {
			selectorCheck = selector.Matches(labels.Set(d.Spec.Template.Labels))
		}
		if _, exist := controllers[d.Name]; !exist && selectorCheck {
			controllers[d.Name] = "Deployment"
		}
	}
	for _, rs := range repset {
		selectorCheck := true
		if selector != nil {
			selectorCheck = selector.Matches(labels.Set(rs.Spec.Template.Labels))
		}
		if _, exist := controllers[rs.Name]; !exist && len(rs.OwnerReferences) == 0 && selectorCheck {
			controllers[rs.Name] = "ReplicaSet"
		}
	}
	for _, fs := range fulset {
		selectorCheck := true
		if selector != nil {
			selectorCheck = selector.Matches(labels.Set(fs.Spec.Template.Labels))
		}
		if _, exist := controllers[fs.Name]; !exist && selectorCheck {
			controllers[fs.Name] = "StatefulSet"
		}
	}
	for _, ds := range daeset {
		selectorCheck := true
		if selector != nil {
			selectorCheck = selector.Matches(labels.Set(ds.Spec.Template.Labels))
		}
		if _, exist := controllers[ds.Name]; !exist && selectorCheck {
			controllers[ds.Name] = "DaemonSet"
		}
	}

	// Build workloads from controllers
	var cnames []string
	for k := range controllers {
		cnames = append(cnames, k)
	}
	sort.Strings(cnames)
	for _, cname := range cnames {
		w := &models.Workload{
			Pods:     models.Pods{},
			Services: []models.ServiceOverview{},
		}
		ctype := controllers[cname]
		// Flag to add a controller if it is found
		cnFound := true
		switch ctype {
		case kubernetes.DeploymentType:
			found := false
			iFound := -1
			for i, dp := range dep {
				if dp.Name == cname {
					found = true
					iFound = i
					break
				}
			}
			if found {
				selector := labels.Set(dep[iFound].Spec.Template.Labels).AsSelector()
				w.SetPods(kubernetes.FilterPodsForSelector(selector, pods))
				w.ParseDeployment(&dep[iFound])
			} else {
				log.Errorf("Workload %s is not found as Deployment", cname)
				cnFound = false
			}
		case kubernetes.ReplicaSetType:
			found := false
			iFound := -1
			for i, rs := range repset {
				if rs.Name == cname {
					found = true
					iFound = i
					break
				}
			}
			if found {
				selector := labels.Set(repset[iFound].Spec.Template.Labels).AsSelector()
				w.SetPods(kubernetes.FilterPodsForSelector(selector, pods))
				w.ParseReplicaSet(&repset[iFound])
			} else {
				log.Errorf("Workload %s is not found as ReplicaSet", cname)
				cnFound = false
			}
		case kubernetes.StatefulSetType:
			found := false
			iFound := -1
			for i, fs := range fulset {
				if fs.Name == cname {
					found = true
					iFound = i
					break
				}
			}
			if found {
				selector := labels.Set(fulset[iFound].Spec.Template.Labels).AsSelector()
				w.SetPods(kubernetes.FilterPodsForSelector(selector, pods))
				w.ParseStatefulSet(&fulset[iFound])
			} else {
				log.Errorf("Workload %s is not found as StatefulSet", cname)
				cnFound = false
			}
		case kubernetes.PodType:
			found := false
			iFound := -1
			for i, pod := range pods {
				if pod.Name == cname {
					found = true
					iFound = i
					break
				}
			}
			if found {
				w.SetPods([]core_v1.Pod{pods[iFound]})
				w.ParsePod(&pods[iFound])
			} else {
				log.Errorf("Workload %s is not found as Pod", cname)
				cnFound = false
			}
		case kubernetes.DaemonSetType:
			found := false
			iFound := -1
			for i, ds := range daeset {
				if ds.Name == cname {
					found = true
					iFound = i
					break
				}
			}
			if found {
				selector := labels.Set(daeset[iFound].Spec.Template.Labels).AsSelector()
				w.SetPods(kubernetes.FilterPodsForSelector(selector, pods))
				w.ParseDaemonSet(&daeset[iFound])
			} else {
				log.Errorf("Workload %s is not found as Deployment", cname)
				cnFound = false
			}
		default:
			// ReplicaSet should be used to link Pods with a custom controller type i.e. Argo Rollout
			cPods := kubernetes.FilterPodsForController(cname, kubernetes.ReplicaSetType, pods)
			if len(cPods) == 0 {
				// If no pods we're found for a ReplicaSet type, it's possible the controller
				// is managing the pods itself i.e. the pod's have an owner ref directly to the controller type.
				cPods = kubernetes.FilterPodsForController(cname, ctype, pods)
			}
			w.SetPods(cPods)

			rsParsed := false
			for _, rs := range repset {
				if strings.HasPrefix(rs.Name, cname) {
					w.ParseReplicaSetParent(&rs, cname, ctype)
					rsParsed = true
					break
				}
			}
			if !rsParsed {
				log.Debugf("Workload %s of type %s has not a ReplicaSet as a child controller, it may need a revisit", cname, ctype)
				w.ParsePods(cname, ctype, cPods)
			}
		}

		if cnFound {
			ws = append(ws, w)
		}
	}
	return ws, nil
}

// KIALI-1730
// This method is used to decide the priority of the controller in the cornercase when two controllers have same labels
// on the selector. Note that this is a situation that user should control as it is described in the documentation:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
// But Istio only identifies one controller as workload (it doesn't note which one).
// Kiali can select one on the list of workloads and other in the details and this should be consistent.
var controllerOrder = map[string]int{
	"Deployment":            6,
	"DeploymentConfig":      5,
	"ReplicaSet":            4,
	"ReplicationController": 3,
	"StatefulSet":           2,
	"Job":                   1,
	"DaemonSet":             0,
	"Pod":                   -1,
}

func controllerPriority(type1, type2 string) string {
	w1, e1 := controllerOrder[type1]
	if !e1 {
		log.Errorf("This controller %s is assigned in a Pod and it's not properly managed", type1)
	}
	w2, e2 := controllerOrder[type2]
	if !e2 {
		log.Errorf("This controller %s is assigned in a Pod and it's not properly managed", type2)
	}
	if w1 >= w2 {
		return type1
	} else {
		return type2
	}
}

func ParseWorkload(w *models.Workload) *models.WorkloadListItem {
	workload := &models.WorkloadListItem{}
	workload.Name = w.Name
	workload.Type = w.Type
	workload.CreatedAt = w.CreatedAt
	workload.ResourceVersion = w.ResourceVersion
	workload.IstioSidecar = w.HasIstioSidecar()
	workload.Labels = w.Labels
	workload.PodCount = len(w.Pods)
	workload.ServiceAccountNames = w.Pods.ServiceAccounts()
	workload.AdditionalDetailSample = w.AdditionalDetailSample
	workload.HealthAnnotations = w.HealthAnnotations
	workload.IstioReferences = []*models.IstioValidationKey{}

	/** Check the labels app and version required by Istio in template Pods*/
	_, workload.AppLabel = w.Labels["app"]
	_, workload.VersionLabel = w.Labels["version"]
	return workload
}
