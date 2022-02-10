package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/utils/harbor"
	"github.com/kubegems/gems/pkg/utils/workflow"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// @Tags Application
// @Summary 应用部署镜像
// @Description 部署镜像
// @Accept json
// @Produce json
// @Param tenant_id      path  int    	true "tenaut id"
// @Param project_id     path  int    	true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name		     path  string	true "应用名称，全部应用可设置为'_'"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/_/images [get]
// @Security JWT
func (h *ApplicationHandler) ListImages(c *gin.Context) {
	h.NoNameRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		dm, err := h.ApplicationProcessor.List(ctx, ref)
		if err != nil {
			return nil, err
		}
		ret := make([]DeployImages, 0, len(dm))
		// 转换为仅镜像格式
		for i := range dm {
			ret = append(ret, ConvertDeploiedManifestToView(*dm[i]))
		}
		return ret, nil
	})
}

// @Tags Application
// @Summary 更新应用镜像并部署
// @Description 更新部署镜像
// @Accept json
// @Produce json
// @Param tenant_id      path  int    	true "tenaut id"
// @Param project_id     path  int    	true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name		     path  string	true "应用名称，全部应用可设置为'_'"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/_/images [put]
// @Security JWT
func (h *ApplicationHandler) BatchUpdateImages(c *gin.Context) {
	body := []DeployImages{}
	h.NoNameRefFunc(c, &body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		updatednames := []string{}

		args := []UpdateImageArgs{}
		for _, item := range body {
			if item.Published {
				updatednames = append(updatednames, item.Name)
				args = append(args, UpdateImageArgs{
					Name:         item.Name,
					Images:       item.PublishImages(),
					IstioVersion: item.IstioVersion,
				})
			}
		}

		h.SetAuditData(c, "更新", "应用镜像", strings.Join(updatednames, ","))
		if err := h.asyncBatchUpdateImages(ctx, ref, args); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

func (h *ApplicationHandler) asyncBatchUpdateImages(ctx context.Context, ref PathRef, args []UpdateImageArgs) error {
	steps := []workflow.Step{
		{
			Name:     "update-image-git-step",
			Function: TaskFunction_Application_BatchUpdateImages,
			Args:     workflow.ArgsOf(ref, args),
		},
	}
	return h.Task.Processor.SubmitTask(ctx, ref, "update-image-git(batch)", steps)
}

// @Tags Application
// @Summary 更新应用镜像并部署
// @Description 更新部署镜像
// @Accept json
// @Produce json
// @Param tenant_id      path  int    	true "tenaut id"
// @Param project_id     path  int    	true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name		     path  string	true "应用名称"
// @Param body		     body  DeployImages	true "更新参数"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/images [put]
// @Security JWT
func (h *ApplicationHandler) UpdateImages(c *gin.Context) {
	item := &DeployImages{}
	h.NamedRefFunc(c, &item, func(ctx context.Context, ref PathRef) (interface{}, error) {
		h.SetAuditData(c, "更新", "应用镜像", ref.Name)

		images := []string{}
		for _, v := range item.Images {
			images = append(images, v.Publish)
		}
		arg := UpdateImageArgs{
			Name:         ref.Name,
			Images:       images,
			IstioVersion: item.IstioVersion,
		}
		if err := h.asyncUpdateImages(ctx, ref, arg); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

func (h *ApplicationHandler) asyncUpdateImages(ctx context.Context, ref PathRef, arg UpdateImageArgs) error {
	steps := []workflow.Step{
		{
			Name:     "update-image",
			Function: TaskFunction_Application_UpdateImages,
			Args:     workflow.ArgsOf(ref, arg.Images, arg.IstioVersion),
		},
		{
			Name:     "sync",
			Function: TaskFunction_Application_Sync,
			Args:     workflow.ArgsOf(ref),
		},
		// {
		// 	Name:     "wait-healthy",
		// 	Function: TaskFunction_Application_WaitHealthy,
		// 	Args:     workflow.ArgsOf(ref),
		// },
	}
	return h.Task.Processor.SubmitTask(ctx, ref, "update-image", steps)
}

type DeployImages struct {
	Name         string                  `json:"name"`
	IstioVersion string                  `json:"istioVersion"`
	Kind         string                  `json:"kind"`
	Images       map[string]*DeployImage `json:"images"`    // 当前准备发布的镜像(gitrepo 中)
	Published    bool                    `json:"published"` // 是否已经发布至 argo
	PublishAt    *metav1.Time            `json:"publishAt"` //
}

func (d *DeployImages) PublishImages() []string {
	images := make([]string, 0, len(d.Images))
	for _, v := range d.Images {
		images = append(images, v.Publish)
	}
	return images
}

type DeployImage struct {
	Running string `json:"running"` // 意为当前 argo 实际正在运行的版本
	Publish string `json:"publish"` // 意为需要更新到的版本，argo更新版本会失败，所以两个版本会存在差异
}

// @Tags Application
// @Summary 更新应用镜像并部署
// @Description 更新部署镜像
// @Accept json
// @Produce json
// @Param tenant_id      path  int    	true "tenaut id"
// @Param project_id     path  int    	true "project id"
// @Param environment_id path  int    true "environment_id"
// @Param name		     path  string	true "应用名称"
// @Param body		     body  DeployImages	true "更新参数"
// @Success 200 {object} handlers.ResponseStruct{Data=DeploiedManifest} "Application"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/images [get]
// @Security JWT
func (h *ApplicationHandler) GetImages(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		dm, err := h.ApplicationProcessor.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		return ConvertDeploiedManifestToView(*dm), nil
	})
}

func ConvertDeploiedManifestToView(dm DeploiedManifest) DeployImages {
	imageDetails := map[string]*DeployImage{}
	// publish img 默认将当前 repo 中的镜像作为发布镜像
	for _, img := range dm.Images {
		name, _ := harbor.SplitImageNameTag(img)
		imageDetails[name] = &DeployImage{Publish: img}
	}
	// runtime img
	for _, img := range dm.Runtime.Images {
		name, _ := harbor.SplitImageNameTag(img)
		if details, ok := imageDetails[name]; ok {
			details.Running = img
		}
	}
	return DeployImages{
		Name:         dm.Name,
		IstioVersion: dm.IstioVersion,
		Kind:         dm.Kind,
		Images:       imageDetails,
		PublishAt:    dm.Runtime.CreateAt.DeepCopy(),
	}
}

// @Tags Application
// @Summary 更新应用镜像
// @Description  更新应用镜像
// @Accept json
// @Produce json
// @Param tenant      path  string    true "租户名称"
// @Param project     path  string    true "项目名称"
// @param environment path  string	  true "环境名称"
// @Param application path  string    true "应用名称"
// @Param image		  query string    true "需要更新的完整镜像(会根据镜像名称寻找编排中相似的镜像并更新tag)"
// @Param version     query string    false "istio version to set"
// @Success 200 {object} handlers.ResponseStruct{Data=object} "status"
// @Router /tenants/{tenant}/projects/{project}/environments/{environment}/applications/{application}/images [post]
// @Security JWT
func (h *ApplicationHandler) DirectUpdateImage(c *gin.Context) {
	h.DirectNamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		// 审计
		h.SetAuditData(c, "更新", "应用镜像", ref.Name)

		image := c.Query("image")
		if image == "" {
			return nil, fmt.Errorf("no image set,please set query item 'image'")
		}
		istioversion := c.Query("version")

		// update
		if err := h.ApplicationProcessor.UpdateImages(ctx, ref, []string{image}, istioversion); err != nil {
			return nil, err
		}
		// sync
		if err := h.ApplicationProcessor.Sync(ctx, ref); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
