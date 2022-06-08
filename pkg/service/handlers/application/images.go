package application

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/containerd/containerd/reference"
	"github.com/gin-gonic/gin"
	"github.com/goharbor/harbor/src/pkg/scan/vuln"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/harbor"
)

const (
	unpublishableLabelKey   = "不可发布"
	unpublishableLabelValue = "该镜像被标记为不可被使用"
)

var ErrNotManagedRegistry = errors.New("unsupported image")

type ImageTag struct {
	TagName       string `json:"name"`
	Image         string `json:"image"`
	Unpublishable bool   `json:"unpublishable"` // 不可发布
}

type ImageHandler struct {
	BaseHandler
}

// @Tags         ProjectImage
// @Summary      镜像安全报告
// @Description  镜像安全报告
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                        true  "tenaut id"
// @Param        project_id  path      int                                        true  "project id"
// @Param        image       query     string                                     true  "eg. kubegems/nginx:v1.14"
// @Success      200         {object}  handlers.ResponseStruct{Data=vuln.Report}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/images/vulnerabilities [get]
// @Security     JWT
func (h *ImageHandler) Vulnerabilities(c *gin.Context) {
	onharborfunc := func(ctx context.Context, cli *harbor.Client, image string) (interface{}, error) {
		vulnerabilities, err := cli.GetArtifactVulnerabilities(ctx, image)
		if err != nil {
			return nil, err
		}
		report := vuln.Report{}
		for _, v := range *vulnerabilities {
			report = v
		}
		// sort 按照漏洞等级排序
		sort.Slice(report.Vulnerabilities, func(i, j int) bool {
			return report.Vulnerabilities[i].Severity.Code() > report.Vulnerabilities[j].Severity.Code()
		})
		return report, nil
	}
	ocifunc := func(ctx context.Context, cli *harbor.OCIDistributionClient, image string) (interface{}, error) {
		return vuln.Report{}, nil
	}
	h.OnHarborFunc(c, onharborfunc, ocifunc)
}

type ImageSummaryItem struct {
	Image            string         `json:"image,omitempty"`
	IsHarborRegistry bool           `json:"isHarborRegistry,omitempty"` // 是否为受集群管理的镜像仓库且为 harbor 仓库
	ScanStatus       string         `json:"scanStatus,omitempty"`       // 扫描状态
	Severity         vuln.Severity  `json:"severity,omitempty"`         // 从低到高依次为 "None""Unknown" "Negligible" "Low""Medium""High""Critical"
	Report           interface{}    `json:"report,omitempty"`           // harbor 报告原文： {"fixable": 122,"summary": {"Critical": 3,"High": 10,"Low": 61,"Medium": 125},"total": 199}
	Unpublishable    bool           `json:"unpublishable,omitempty"`    // 不可发布状态，若为true则不可发布
	Labels           []harbor.Label `json:"labels,omitempty"`           // harbor 标签
	Status           string         `json:"status,omitempty"`
	UpdatedAt        *metav1.Time   `json:"updatedAt,omitempty"` // time.Time.Format(RFC3339) 格式,若非可管理仓库则为空
}

// @Tags         ProjectImage
// @Summary      镜像summary
// @Description  镜像summary
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                        true  "tenaut id"
// @Param        project_id  path      int                                        true  "project id"
// @Param        image       query     string                                     true  "eg. kubegems/nginx:v1.14"
// @Success      200         {object}  handlers.ResponseStruct{Data=vuln.Report}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/images/summary [get]
// @Security     JWT
func (h *ImageHandler) Summary(c *gin.Context) {
	onharborfunc := func(ctx context.Context, cli *harbor.Client, image string) (interface{}, error) {
		artifacts, err := cli.ListArtifact(ctx, image, harbor.GetArtifactOptions{
			WithScanOverview: true,
			WithLabel:        true,
			WithTag:          true,
		})
		if err != nil {
			return nil, err
		}

		summaries := []ImageSummaryItem{}

		image, _ = harbor.SplitImageNameTag(image)
		for _, artifact := range artifacts {
			if artifact.Type != "IMAGE" || len(artifact.Tags) == 0 {
				continue
			}

			item := ImageSummaryItem{
				Image:            image + ":" + artifact.Tags[0].Name,
				UpdatedAt:        &metav1.Time{Time: artifact.Artifact.PushTime},
				Labels:           artifact.Labels,
				IsHarborRegistry: true,
				Unpublishable: func() bool {
					for _, label := range artifact.Labels {
						if label.Name == unpublishableLabelKey {
							return true
						}
					}
					return false
				}(),
			}
			for _, overview := range artifact.ScanOverview {
				item.Report = overview.Summary
				item.ScanStatus = overview.ScanStatus
				item.Severity = overview.Severity
			}
			summaries = append(summaries, item)
		}

		paged := handlers.NewPageDataFromContext(c, summaries, nil, nil)
		return paged, nil
	}

	ocifunc := func(_ context.Context, _ *harbor.OCIDistributionClient, _ string) (interface{}, error) {
		summaries := []ImageSummaryItem{}
		return handlers.NewPageDataFromContext(c, summaries, nil, nil), nil
	}

	h.OnHarborFunc(c, onharborfunc, ocifunc)
}

// @Tags         ProjectImage
// @Summary      镜像不可发布标记
// @Description  镜像不可发布标记
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                        true  "tenaut id"
// @Param        project_id  path      int                                        true  "project id"
// @Param        image       query     string                                     true  "eg. kubegems/nginx:v1.14"
// @Success      200         {object}  handlers.ResponseStruct{Data=vuln.Report}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/images/unpublishable [get]
// @Security     JWT
func (h *ImageHandler) Unpublishable(c *gin.Context) {
	onharborfunc := func(ctx context.Context, cli *harbor.Client, image string) (interface{}, error) {
		isunpublishable, _ := strconv.ParseBool(c.Query("unpublishable"))
		if isunpublishable {
			if err := cli.AddArtifactLabelFromKey(ctx, image, unpublishableLabelKey, unpublishableLabelValue); err != nil {
				return nil, err
			}
		} else {
			if err := cli.DeleteArtifactLabelFromKey(ctx, image, unpublishableLabelKey); err != nil {
				return nil, err
			}
		}
		return "ok", nil
	}
	ocifunc := func(ctx context.Context, cli *harbor.OCIDistributionClient, image string) (interface{}, error) {
		return "ok", nil
	}
	h.OnHarborFunc(c, onharborfunc, ocifunc)
}

// @Tags         ProjectImage
// @Summary      镜像扫描
// @Description  触发镜像扫描
// @Accept       json
// @Produce      json
// @Param        tenant_id   path      int                                        true  "tenaut id"
// @Param        project_id  path      int                                        true  "project id"
// @Param        image       query     string                                     true  "eg. kubegems/nginx:v1.14"
// @Success      200         {object}  handlers.ResponseStruct{Data=vuln.Report}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/images/scan [post]
// @Security     JWT
func (h *ImageHandler) Scan(c *gin.Context) {
	onharborfunc := func(ctx context.Context, cli *harbor.Client, image string) (interface{}, error) {
		if err := cli.ScanArtifact(ctx, image); err != nil {
			return nil, err
		}
		return "ok", nil
	}
	ocifunc := func(ctx context.Context, cli *harbor.OCIDistributionClient, image string) (interface{}, error) {
		return "ok", nil
	}
	h.OnHarborFunc(c, onharborfunc, ocifunc)
}

// @Tags         ProjectImage
// @Summary      镜像tags
// @Description  查询镜像tags
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                        true  "tenaut id"
// @Param        project_id      path      int                                        true  "project id"
// @Param        application_id  path      int                                        true  "application id"
// @Param        image           query     string                                     true  "eg. kubegems/nginx:v1.14"
// @Success      200             {object}  handlers.ResponseStruct{Data=vuln.Report}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/images/tags [post]
// @Security     JWT
func (h *ImageHandler) ImageTags(c *gin.Context) {
	harborfunc := func(ctx context.Context, cli *harbor.Client, image string) (interface{}, error) {
		arts, err := cli.ListArtifact(ctx, image, harbor.GetArtifactOptions{
			WithTag:   true,
			WithLabel: true,
		})
		if err != nil {
			return nil, err
		}

		unpublishableLabeld := func(labels []harbor.Label) bool {
			for _, label := range labels {
				if label.Name == unpublishableLabelKey {
					return true
				}
			}
			return false
		}

		name, _ := harbor.SplitImageNameTag(image)

		ret := []ImageTag{}
		for _, item := range arts {
			for _, tag := range item.Tags {
				ret = append(ret, ImageTag{
					TagName:       tag.Name,
					Image:         name + ":" + tag.Name,
					Unpublishable: unpublishableLabeld(item.Labels),
				})
			}
		}
		return ret, nil
	}

	ocifunc := func(ctx context.Context, cli *harbor.OCIDistributionClient, image string) (interface{}, error) {
		fullname, _ := harbor.SplitImageNameTag(image)
		tags, _ := cli.ListTags(ctx, image)
		ret := []ImageTag{}
		for _, tag := range tags.Tags {
			ret = append(ret, ImageTag{
				TagName: tag,
				Image:   fullname + ":" + tag,
			})
		}
		return ret, nil
	}
	h.OnHarborFunc(c, harborfunc, ocifunc)
}

type (
	HarborFunc func(ctx context.Context, cli *harbor.Client, image string) (interface{}, error)
	OCIFunc    func(ctx context.Context, cli *harbor.OCIDistributionClient, image string) (interface{}, error)
)

type RegistryOptions struct {
	URL      string
	Username string
	Password string
}

func (h *ImageHandler) OnHarborFunc(c *gin.Context, harborfun HarborFunc, ocifunc OCIFunc) {
	process := func(ctx context.Context) (interface{}, error) {
		params := struct {
			ProjectID uint `uri:"project_id" binding:"required"`
			TenautID  uint `uri:"tenant_id" binding:"required"`
		}{}
		if err := c.ShouldBindUri(&params); err != nil {
			return nil, err
		}

		image := c.Query("image")
		if image == "" {
			return nil, fmt.Errorf("empty image name")
		}

		u, _, _, _, _ := harbor.ParseImag(image)
		if u == "docker.io" {
			u = "index.docker.io"
		}
		u = "https://" + u

		options := &RegistryOptions{
			URL: u,
		}
		_ = h.completeRegistryOption(options, image, params.ProjectID) // ignore

		// try harbor
		cli, err := harbor.NewClient(options.URL, options.Username, options.Password)
		if err != nil {
			return nil, err
		}

		if _, err := cli.SystemInfo(ctx); err != nil {
			// try oci
			ocicli := harbor.NewOCIDistributionClient(options.URL, options.Username, options.Password)
			return ocifunc(ctx, ocicli, image)
		}
		// isharbor
		return harborfun(ctx, cli, image)
	}

	if data, err := process(c.Request.Context()); err != nil {
		handlers.NotOK(c, err)
	} else {
		handlers.OK(c, data)
	}
}

func (h *ImageHandler) completeRegistryOption(options *RegistryOptions, imgname string, projectid uint) error {
	spec, err := reference.Parse(imgname)
	if err != nil {
		return err
	}
	hostname := spec.Hostname()
	registries := []models.Registry{}

	if err := h.GetDataBase().DB().Where(&models.Registry{ProjectID: projectid}).Find(&registries).Error; err != nil {
		return err
	}
	for _, registry := range registries {
		u, err := url.Parse(registry.RegistryAddress)
		if err != nil {
			log.WithField("registry", registry.RegistryAddress).Warn("invalid addr skiped")
			continue
		}
		if u.Hostname() == hostname {
			// found
			options.URL = registry.RegistryAddress
			options.Password = registry.Password
			options.Username = registry.Username
			return nil
		}
	}
	// 不是内置的，或者该项目的镜像仓库
	return ErrNotManagedRegistry
}
