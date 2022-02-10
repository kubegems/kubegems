package application

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kubegems/gems/pkg/utils/git"
)

const BaseEnv = "base"

// PathRef 存储了从 租户项目环境应用 到 git仓库中的 group repo branch path的映射关系
type PathRef struct {
	Tenant  string `json:"tenant,omitempty"`
	Project string `json:"project,omitempty"`
	Env     string `json:"environment,omitempty"`
	Name    string `json:"name,omitempty"`
}

func (p PathRef) Encode() string {
	content, _ := json.Marshal(p)
	return string(content)
}

func (p *PathRef) DecodeFrom(str string) error {
	return json.Unmarshal([]byte(str), p)
}

func (p PathRef) GitRef() git.RepositoryRef {
	if p.Env == "" {
		p.Env = BaseEnv
	}
	return git.RepositoryRef{
		Org:    p.Tenant,
		Repo:   p.Project,
		Branch: p.Env,
		Path:   p.Name,
	}
}

// Path is where the manifests located in the repository
func (p PathRef) Path() string {
	return p.Name
}

func (p PathRef) GitBranch() string {
	if p.Env == "" {
		p.Env = BaseEnv
	}
	return p.Env
}

// FullName is the full name for the argo application
func (p PathRef) FullName() string {
	if p.Env == "" {
		p.Env = BaseEnv
	}
	splitor := '-'
	name := strings.Join([]string{p.Tenant, p.Project, p.Env, p.Name}, string(splitor))
	const maxlen = 53
	// k8s中对象名字长度需要小于53
	if len(name) > maxlen {
		// 先缩小租户/项目/环境
		//  8      4   4   4   {name}
		// {hash}-ten-pro-env-{name}
		// nolint: gomnd
		tpe := shortStrWithHashPrefix([]string{p.Tenant, p.Project, p.Env}, 3, 3, 3) // 20 字符;{hash}-[ten]-[pro]-[env]
		name = tpe + string(splitor) + p.Name
		if len(name) > maxlen { // // 53-20=33 如果name超过33字符则会被全部缩短
			// 全部缩小
			//  8    1 3 1 3 1 3 1 32 = 53
			// {hash}-ten-pro-env-{name[:]}
			// nolint: gomnd
			name = shortStrWithHashPrefix([]string{p.Tenant, p.Project, p.Env, p.Name}, 3, 3, 3, 32)
		}
	}
	return name
}

func shortStrWithHashPrefix(ss []string, everylen ...int) string {
	hasher := md5.New()
	hasher.Reset()
	for _, v := range ss {
		hasher.Write([]byte(v))
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))[8:16]

	ret := hash
	for i, vl := range everylen {
		if vl > len(ss[i]) || vl == 0 {
			vl = len(ss[i])
		}
		ret += "-" + ss[i][:vl]
	}
	return ret
}

func (p PathRef) JsonStringBase64() string {
	content, _ := json.Marshal(p)
	return base64.StdEncoding.EncodeToString(content)
}

func (p *PathRef) FromJsonBase64(str string) {
	content, _ := base64.StdEncoding.DecodeString(str)
	_ = json.Unmarshal([]byte(content), p)
}

func (p PathRef) IsEmpty() bool {
	return p.Tenant == "" && p.Project == "" && p.Env == "" && p.Name == ""
}

func (p *PathRef) FromAnnotations(annotations map[string]string) {
	if tenant, exist := annotations[ArgoLabelTenant]; exist {
		p.Tenant = tenant
	}
	if project, exist := annotations[ArgoLabelProject]; exist {
		p.Project = project
	}
	if env, exist := annotations[ArgoLabelEnvironment]; exist {
		p.Env = env
	}
	if app, exist := annotations[ArgoLabelApplication]; exist {
		p.Name = app
	}
}

func (p *PathRef) FromArgoLabel(labels map[string]string) {
	if labels == nil {
		return
	}
	p.Tenant = labels[ArgoLabelTenant]
	p.Project = labels[ArgoLabelProject]
	p.Env = labels[ArgoLabelEnvironment]
	p.Name = labels[ArgoLabelApplication]
}
