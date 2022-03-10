package application

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	rollouts "github.com/argoproj/argo-rollouts/pkg/apis/rollouts"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/v2/services/handlers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceWhileList 允许编排的资源类型
var ResourceWhileList = []schema.GroupVersionKind{
	{Group: corev1.GroupName},                                   // pod configmap secret etc.
	{Group: appsv1.GroupName},                                   // deployment statefulset daemonset
	{Group: batchv1.GroupName},                                  // job cronjob (v1,v2)
	{Group: extensionsv1beta1.GroupName},                        // deprecated, ingress deployment scale etc
	{Group: networkingv1.GroupName},                             // ingress ingressclass network_policy
	{Group: storagev1.GroupName, Kind: "PersistentVolumeClaim"}, // pvc
	{Group: rollouts.Group},                                     // argo rollouts(rollouts,analysis_template etc.)
}

// @Tags ApplicationManifest
// @Summary 写入文件
// @Description 修改应用编排
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "name"
// @Param filename		path  string	true "file name"
// @Param body 			body  FileContent 	true "filecontent"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/files/{filename} [put]
// @Security JWT
func (h *ManifestHandler) UpdateFile(req *restful.Request, resp *restful.Response) {
	body := &FileContent{}
	h.NamedRefFunc(req, resp, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		filename := req.PathParameter("file")
		if filename == "" {
			return nil, fmt.Errorf("filename cannot be empty")
		}

		content := []byte(body.Content)

		// 初次验证
		obj, err := DecodeResource(content)
		if err != nil {
			return nil, err
		}
		if !IsPermmitedResource(obj) {
			return nil, fmt.Errorf("resource %s is not permitted to be create", obj.GetObjectKind().GroupVersionKind().String())
		}
		updatefunc := func(ctx context.Context, fs billy.Filesystem) error {
			return util.WriteFile(fs, filename, content, os.ModePerm)
		}
		if err := h.UpdateContentFunc(ctx, ref, updatefunc, fmt.Sprintf("put file %s", filename)); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags ApplicationManifest
// @Summary 写入多个文件
// @Description 修改应用编排
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "name"
// @Param filename		path  string	true "file name"
// @Param msg 			query string 	true "commit mesage"
// @Param body 			body  []FileContent 	true "files"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/files [put]
// @Security JWT
func (h *ManifestHandler) UpdateFiles(req *restful.Request, resp *restful.Response) {
	files := []FileContent{}
	h.NamedRefFunc(req, resp, &files, func(ctx context.Context, ref PathRef) (interface{}, error) {
		msg := req.QueryParameter("msg")
		updatefunc := func(_ context.Context, fs billy.Filesystem) error {
			for _, file := range files {
				util.WriteFile(fs, file.Name, []byte(file.Content), os.ModePerm)
			}
			return nil
		}
		if msg == "" {
			msg = "put files"
		}
		if err := h.UpdateContentFunc(ctx, ref, updatefunc, msg); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags ApplicationManifest
// @Summary 删除文件
// @Description 修改应用编排
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "name"
// @Param filename		path  string	true "file name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/files/{filename} [delete]
// @Security JWT
func (h *ManifestHandler) DeleteFile(req *restful.Request, resp *restful.Response) {
	body := []byte{}
	h.NamedRefFunc(req, resp, &body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		filename := req.PathParameter("file")
		if filename == "" {
			return nil, fmt.Errorf("filename cannot be empty")
		}
		updatefunc := func(ctx context.Context, fs billy.Filesystem) error {
			return util.RemoveAll(fs, filename)
		}
		if err := h.UpdateContentFunc(ctx, ref, updatefunc, fmt.Sprintf("remve file %s", filename)); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

type FileContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (h *ManifestHandler) GetFile(req *restful.Request, resp *restful.Response) {
}

// @Tags ApplicationManifest
// @Summary 列举文件
// @Description 应用编排内容
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=[]FileContent} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/files [get]
// @Security JWT
func (h *ManifestHandler) ListFiles(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		files := []FileContent{}

		fun := func(ctx context.Context, fs billy.Filesystem) error {
			return ForFileContentFunc(fs, "", func(filename string, content []byte) error {
				if filename == "" || filename[0] == '.' {
					return nil
				}
				files = append(files, FileContent{Name: filename, Content: string(content)})
				return nil
			})
		}

		if err := h.ContentFunc(ctx, ref, fun); err != nil {
			return nil, err
		}
		return files, nil
	})
}

type GitLog struct {
	Hash      string `json:"hash,omitempty"`
	Message   string `json:"message,omitempty"`
	Author    string `json:"author,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// @Tags ApplicationManifest
// @Summary 应用编排文件历史
// @Description 应用编排文件历史
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=[]GitLog} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/gitlog [get]
// @Security JWT
func (h *ManifestHandler) GitLog(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		logs := []GitLog{}

		fun := func(ctx context.Context, repository Repository) error {
			return repository.HistoryFunc(ctx, func(_ context.Context, commit git.Commit) error {
				logs = append(logs, GitLog{
					Hash:      commit.Hash,
					Message:   commit.Message,
					Author:    commit.Author.Name,
					Timestamp: commit.Author.When.Format(time.RFC3339),
				})
				return nil
			})
		}
		if err := h.Func(ctx, ref, fun); err != nil {
			return nil, err
		}
		paged := handlers.NewPageDataFromContext(req, logs, nil, nil)
		return paged, nil
	})
}

// @Tags ApplicationManifest
// @Summary 应用编排文件diff
// @Description 应用编排文件diff
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "application name"
// @Param hash			query  string	true "gitcommit hash"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/gitdiff [get]
// @Security JWT
func (h *ManifestHandler) GitDiff(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		hash := req.QueryParameter("hash")

		var ret []git.FileDiff
		fun := func(ctx context.Context, repository Repository) error {
			diff, err := repository.Diff(ctx, hash)
			if err != nil {
				return err
			}
			ret = diff
			return nil
		}

		if err := h.Func(ctx, ref, fun); err != nil {
			return nil, err
		}
		return ret, nil
	})
}

// @Tags ApplicationManifest
// @Summary 应用编排文件回滚
// @Description 回滚应用编排文件
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "application name"
// @Param hash			query  string	true "gitcommit hash to rollback"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/gitrevert [get]
// @Security JWT
func (h *ManifestHandler) GitRevert(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		hash := req.QueryParameter("hash")

		err := h.Func(ctx, ref,
			Revert(hash),
			UpdateKustomizeCommit(fmt.Sprintf("revert to %s", hash)),
		)
		if err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

func Revert(rev string) RepositoryFunc {
	return func(ctx context.Context, repository Repository) error {
		// get history
		commit, err := repository.HistoryFiles(ctx, rev)
		if err != nil {
			return err
		}

		fs, err := repository.FS(ctx)
		if err != nil {
			return nil
		}

		util.RemoveAll(fs, ".")
		// copy to
		for _, file := range commit.Files {
			_ = util.WriteFile(fs, file.Name, []byte(file.Content), os.ModePerm)
		}
		return nil
	}
}

func IsPermmitedResource(obj runtime.Object) bool {
	gvk := obj.GetObjectKind().GroupVersionKind()

	for _, white := range ResourceWhileList {
		if white.Group != gvk.Group {
			continue
		}
		// 如果为空则适配所有
		if white.Version != "" && white.Version != gvk.Version {
			continue
		}
		// 如果为空则适配所有
		if white.Kind != "" && white.Kind != gvk.Kind {
			continue
		}
		return true
	}
	return false
}

// @Tags ApplicationManifest
// @Summary 应用编排文件刷新
// @Description 应用编排文件刷新(git pull)
// @Accept json
// @Produce json
// @Param tenant_id     path  int    	true "tenaut id"
// @Param project_id    path  int    	true "project id"
// @Param name			path  string	true "application name"
// @Param hash			query  string	true "gitcommit hash"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "ok"
// @Router /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/gitpull [post]
// @Security JWT
func (h *ManifestHandler) GitPull(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		if err := h.Func(ctx, ref, Pull()); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

func Pull() RepositoryFunc {
	return func(ctx context.Context, repository Repository) error {
		return repository.repo.Pull(ctx)
	}
}

type CommitImageDetails struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
	Creator   string      `json:"creator,omitempty"`
	Images    []string    `json:"images,omitempty"`
}

func (h *ManifestHandler) parseCommitImagesFunc(ctx context.Context, ref PathRef, hash string) (*CommitImageDetails, error) {
	var details *CommitImageDetails

	err := h.Func(ctx, ref, func(ctx context.Context, repository Repository) error {
		commit, err := repository.HistoryFiles(ctx, hash)
		if err != nil {
			return err
		}
		images := []string{}
		for _, f := range commit.Files {
			if !strings.HasSuffix(f.Name, ".yaml") {
				continue
			}
			obj, err := DecodeResource([]byte(f.Content))
			if err != nil {
				// ignore
				continue
			}
			images = append(images, ParseImagesFrom(obj)...)
		}
		details = &CommitImageDetails{
			CreatedAt: metav1.NewTime(commit.Author.When),
			Creator:   commit.Author.Name,
			Images:    images,
		}
		return nil
	})

	return details, err
}

func ParseImagesFrom(obj client.Object) []string {
	images := []string{}
	updatefunc := func(template *corev1.PodTemplateSpec) {
		for _, c := range template.Spec.Containers {
			images = append(images, c.Image)
		}
	}
	ObjectPodTemplateFunc(obj, updatefunc)
	return images
}
