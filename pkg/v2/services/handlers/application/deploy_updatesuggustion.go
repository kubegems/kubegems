package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type ResourceSuggestion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceSuggestionSpec `json:"spec,omitempty"`
}

func (r *ResourceSuggestion) DeepCopyObject() runtime.Object {
	return r
}

type ResourceSuggestionSpec struct {
	Template PodTemplateSpec `json:"template"`
}

type PodTemplateSpec struct {
	Spec PodSpec `json:"spec,omitempty"`
}

type PodSpec struct {
	InitContainers      []Container `json:"initContainers,omitempty"`
	Containers          []Container `json:"containers"`
	EphemeralContainers []Container `json:"ephemeralContainers,omitempty"`
}

type Container struct {
	Name      string                      `json:"name,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
}

// @Tags         Application
// @Summary      更新资源建议至 gitrepo
// @Description  更新资源建议至 gitrepo
// @Accept       json
// @Produce      json
// @Param        cluster    path      string                                true  "-"
// @Param        group      path      string                                true  "-"
// @Param        version    path      string                                true  "-"
// @param        namespace  path      string                                true  "-"
// @Param        resource   path      string                                true  "-"
// @Param        name       path      string                                true  "-"
// @Success      200        {object}  handlers.ResponseStruct{Data=object}  "-"
// @Router       /v1/cluster/{cluster}/{group}/{version}/namespaces/{namespace}/{resource}/{name} [patch]
// @Security     JWT
func (h *ApplicationHandler) UpdateWorkloadResources(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	process := func() error {
		suggestion := ResourceSuggestion{}

		if err := req.ReadEntity(&suggestion); err != nil {
			return err
		}
		if suggestion.TypeMeta.GroupVersionKind().Empty() || suggestion.Name == "" {
			return errors.New("empty resource kind or name")
		}

		if suggestion.Annotations == nil {
			suggestion.Annotations = map[string]string{}
		}
		ref := &PathRef{}
		ref.FromJsonBase64(suggestion.Annotations[AnnotationRef])
		if ref.IsEmpty() {
			return errors.New("not a argo managed resource")
		}

		updatefunc := func(_ context.Context, fs billy.Filesystem) error {
			return ForFileContentFunc(fs, "", func(filename string, content []byte) error {
				if filepath.Ext(filename) != ".yaml" {
					return nil
				}
				obj, err := DecodeResource(content)
				if err != nil {
					return nil
				}
				// check Kind Name
				if (obj.GetObjectKind().GroupVersionKind() != suggestion.TypeMeta.GroupVersionKind()) || obj.GetName() != suggestion.Name {
					return nil
				}
				// update resource
				updated := UpdatedReourcesLimits(obj, suggestion)
				if updated {
					content, err := yaml.Marshal(obj)
					if err != nil {
						return err
					}
					return util.WriteFile(fs, filename, content, os.ModePerm)
				}
				return nil
			})
		}

		// update git
		msg := fmt.Sprintf("update resource suggestion for %s name=%s", suggestion.GroupVersionKind().String(), suggestion.ObjectMeta.Name)
		if err := h.Manifest.UpdateContentFunc(ctx, *ref, updatefunc, msg); err != nil {
			return err
		}
		// sync
		if err := h.ApplicationProcessor.Sync(ctx, *ref); err != nil {
			return err
		}
		return nil
	}

	if err := process(); err != nil {
		handlers.BadRequest(resp, err)
	} else {
		handlers.OK(resp, "ok")
	}
}

func UpdatedReourcesLimits(obj client.Object, suggestion ResourceSuggestion) bool {
	updated := false
	updatefunc := func(template *corev1.PodTemplateSpec) {
		for i, c := range template.Spec.Containers {
			for _, suggestioncontainer := range suggestion.Spec.Template.Spec.Containers {
				if suggestioncontainer.Name == c.Name {
					template.Spec.Containers[i].Resources = suggestioncontainer.Resources
					updated = true
				}
			}
		}
	}
	ObjectPodTemplateFunc(obj, updatefunc)
	return updated
}
