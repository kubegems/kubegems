<div style="text-align: center"></div>
  <p align="center">
  <img src="https://www.kubegems.io/img/logo.svg" width="40%" height="40%">
      <br>
      <i>Let cloudnative management more easily.</i>
  </p>
</div>

[![.github/workflows/build.yml](https://github.com/kubegems/kubegems/actions/workflows/build.yml/badge.svg)](https://github.com/kubegems/kubegems/actions/workflows/build.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/kubegems/kubegems.svg?maxAge=604800)](https://hub.docker.com/r/kubegems/kubegems)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubegems/kubegems)](https://goreportcard.com/report/github.com/kubegems/kubegems)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kubegems/kubegems?logo=go)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubegems/kubegems?logo=github&sort=semver)](https://github.com/kubegems/kubegems/releases/latest)
[![Artifact HUB](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubegems)](https://artifacthub.io/packages/search?repo=kubegems)
![license](https://img.shields.io/github/license/kubegems/kubegems)

<h2>
  <a href="https://kubegems.io/">Website</a>
  <span> • </span>
  <a href="https://www.kubegems.io/docs/concepts/what-is-kubegems">Docs</a>
  <span> • </span>
  <a href="https://demo.kubegems.io/">Demo</a>
  <span> • </span>
  <a href="https://github.com/kubegems/.github/blob/master/static/image/wechat.jpg?raw=true">Wechat</a>
</h2>

 🇨🇳 简体中文  🇭🇰 繁体中文  🇺🇸 英文  🇯🇵 日语

## 介绍

> 中文 | [English](README.md)

[KubeGems](https://kubegems.io) 是一款以围绕 Kubernetes 通过自研和集成云原生项目而构建的通用性开源 PaaS 云管理平台。经过我们内部近一年的持续迭代，当前 KubeGems 的核心功能已经初步具备多云多租户场景下的统一管理。并通过插件化的方式，在用户界面中灵活控制包括 监控系统、日志系统、微服务治理 等众多插件的启用和关闭。

## 功能

Kubegems遵循云原生应用程序的最佳实践，以最简单、最有效的方式向用户提供服务。

<details>
  <summary><b> 多 Kubernetes 集群管理</b></summary>
</details>

<details>
  <summary><b>多租户</b></summary>
</details>

<details>
  <summary><b>插件管理</b></summary>
</details>

<details>
  <summary><b>基于 ArgoCD 的 GitOps</b></summary>
</details>

<details>
  <summary><b> 可观测性 (OpenTelemetry)</b></summary>
</details>

<details>
  <summary><b>基于 Istio 的微服务治理</b></summary>
</details>

<details>
  <summary><b>应用商店</b></summary>
</details>

<details>
  <summary><b> AI模型服务</b></summary>
</details>

## 截图

<br/>
<table>
    <tr>
      <td width="50%" align="center"><b>集群管理</b></td>
      <td width="50%" align="center"><b>租户工作台</b></td>
    </tr>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/cluster_en.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/tenant.jpg?raw=true"></td>
    </tr>
    <tr>
      <td width="50%" align="center"><b>应用商店</b></td>
      <td width="50%" align="center"><b>AI 算法商店</b></td>
    </tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/appstore.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/model_store_en.jpg?raw=true"></td>
    <tr>
    </tr>
    <tr>
      <td width="50%" align="center"><b>微服务治理</b></td>
      <td width="50%" align="center"><b>可观测性</b></td>
    </tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/istio_en.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/appdash.jpg?raw=true"></td>
    <tr>
    </tr>
</table>

## 在线环境

您可以访问 KubeGems 的[在线环境](https://demo.kubegems.io)

> 用户名：`admin`   密码： `demo!@#admin`

## 快速开始

### 安装 Kubernetes 集群

您可以通过以下方式安装 Kubernetes 集群，我们推荐 Kubernetes 的版本 v1.20 +

1. [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/)
2. [kind](https://kind.sigs.k8s.io/)
3. [kubekey](https://github.com/kubesphere/kubekey)
4. 其他方式

### 安装 KubeGems

确定部署版本:

您可以前往 [Kubegems Release](https://github.com/kubegems/kubegems/tags) 查询到最新的版本号.

```sh
export KUBEGEMS_VERSION=v1.22.0  #  安装 kubegems 的版本
```

当你的 Kubernetes 集群状态 Ready 后，执行下述命令安装 KuebGems Installer Operator。

```sh
kubectl create namespace kubegems-installer
kubectl apply -f "https://github.com/kubegems/kubegems/raw/${KUBEGEMS_VERSION}/deploy/installer.yaml"
```

通过 operator 安装 KuebGems 核心服务

```sh
kubectl create namespace kubegems

export STORAGE_CLASS=local-path  # 声明storageclass
curl -sL "https://github.com/kubegems/kubegems/raw/${KUBEGEMS_VERSION}/deploy/kubegems.yaml" \
| sed -e "s/local-path/${STORAGE_CLASS}/g" \
> kubegems.yaml

kubectl apply -f kubegems.yaml
```

更多信息，请访问 <https://www.kubegems.io/docs/installation/quick-install>

## 参与贡献

如果您认为这个项目有用，请帮助我们：

- 支持这个项目的开发并star这个repo ⭐
- 如果您在生产环境中使用KubeGems，请将自己添加到[adopters](./ADOPTERS.md)列表中 👌
- 帮助新用户解决他们可能遇到的问题 🙋
- 发送带有新功能和错误修复的拉取请求 🚀

我们非常欢迎您在KubeGems社区在平台体验、标准化应用程序、插件共享等领域的贡献和分享。

更多信息，请访问 <https://github.com/kubegems/kubegems/blob/main/CONTRIBUTING.md>

### 感谢以下贡献者 !

[//]: contributor-faces
<a href="https://github.com/pepesi"><img src="https://avatars.githubusercontent.com/u/12043478?v=4" title="pepesi" width="80" height="80"></a>
<a href="https://github.com/chenshunliang"><img src="https://avatars.githubusercontent.com/u/6768455?v=4" title="chenshunliang" width="80" height="80"></a>
<a href="https://github.com/cnfatal"><img src="https://avatars.githubusercontent.com/u/15731850?v=4" title="cnfatal" width="80" height="80"></a>
<a href="https://github.com/LinkMaq"><img src="https://avatars.githubusercontent.com/u/2688646?v=4" title="LinkMaq" width="80" height="80"></a>
<a href="https://github.com/jojotong"><img src="https://avatars.githubusercontent.com/u/100849526?v=4" title="jojotong" width="80" height="80"></a>
<a href="https://github.com/sunlintong"><img src="https://avatars.githubusercontent.com/u/32826013?v=4" title="sunlintong" width="80" height="80"></a>
<a href="https://github.com/zhanghe9702"><img src="https://avatars.githubusercontent.com/u/16931323?v=4" title="zhanghe9702" width="80" height="80"></a>
<a href="https://github.com/Jianwen-Li"><img src="https://avatars.githubusercontent.com/u/29349603?v=4" title="Jianwen-Li" width="80" height="80"></a>
<a href="https://github.com/KinglyWayne"><img src="https://avatars.githubusercontent.com/u/3232817?v=4" title="KinglyWayne" width="80" height="80"></a>
<a href="https://github.com/itxx00"><img src="https://avatars.githubusercontent.com/u/1866789?v=4" title="itxx00" width="80" height="80"></a>
<a href="https://github.com/VioZhang"><img src="https://avatars.githubusercontent.com/u/41519383?v=4" title="VioZhang" width="80" height="80"></a>
<a href="https://github.com/liutao-east"><img src="https://avatars.githubusercontent.com/u/20122705?v=4" title="liutao-east" width="80" height="80"></a>

