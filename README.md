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

[Documentation](https://kubegems.io) • [Demo](https://demo.kubegems.io)

## Introduction

> English | [中文](README_zh.md)

[KubeGems](https://kubegems.io) is an open source, enterprise-class multi-tenant container cloud platform. Built around a cloud-native community, and kubegems provides access to multiple kubernetes clusters with rich component management and resource cost analysis capabilities to help enterprises quickly build and build a localized, powerful and low-cost cloud management platform.

<p align="center">
<img src="https://github.com/kubegems/.github/blob/master/static/image/cluster.drawio.png?raw=true">
</p>

## Highlights

Kubegems follows the best practices of cloud-native applications and delivers them to users in the simplest and most efficient way.

<details>
  <summary><b>Multiple kubernetes cluster</b></summary>
</details>

<details>
  <summary><b>Multi-tenancy</b></summary>
</details>

<details>
  <summary><b>Plugins management</b></summary>
</details>

<details>
  <summary><b>GitOps with Argocd/Rollout</b></summary>
</details>

<details>
  <summary><b>Observability (OpenTelemetry)</b></summary>
</details>

<details>
  <summary><b>ServiceMesh based on istio</b></summary>
</details>

<details>
  <summary><b>Applications Store</b></summary>
</details>

<details>
  <summary><b>Smart ML(Machine Learning) Models Serving</b></summary>
</details>

## Snapshots

<br/>
<table>
    <tr>
      <td width="50%" align="center"><b>Tenant Overview</b></td>
      <td width="50%" align="center"><b>WorkSpace</b></td>
    </tr>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/tenant.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/workspace.jpg?raw=true"></td>
    </tr>
    <tr>
      <td width="50%" align="center"><b>Clusters</b></td>
      <td width="50%" align="center"><b>Plugins</b></td>
    </tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/cluster.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/plugins.jpg?raw=true"></td>
    <tr>
    </tr>
    <tr>
      <td width="50%" align="center"><b>Appstore</b></td>
      <td width="50%" align="center"><b>Observability</b></td>
    </tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/appstore.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/observability.jpg?raw=true"></td>
    <tr>
    </tr>
</table>

## Online Demo

You can visit our [KubeGems Online Demo](https://demo.kubegems.io)

>account: `admin`    password: `demo!@#admin`

## Getting started

### Install Kubernetes cluster

You can Install your k8s cluster using any of the following methods, supported k8s version is v1.20 +

1. [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/)
2. [kind](https://kind.sigs.k8s.io/)
3. [kubekey](https://github.com/kubesphere/kubekey)
4. Any other ways...

### Installation

Choose a kubegems version from [Kubegems Release](https://github.com/kubegems/kubegems/tags):

```sh
export KUBEGEMS_VERSION=v1.22.0-beta.2  # change to kubegems version
```

When your k8s cluster is ready, next you can install kubegems insatller operator on your cluster.

```sh
kubectl create namespace kubegems-installer
kubectl apply -f "https://github.com/kubegems/kubegems/raw/${KUBEGEMS_VERSION}/deploy/installer.yaml"
```

Install kubegems with installer operator.

```sh
kubectl create namespace kubegems

export STORAGE_CLASS=local-path  # set to your storageClass
curl -sL "https://github.com/kubegems/kubegems/raw/${KUBEGEMS_VERSION}/deploy/kubegems.yaml" \
| sed -e "s/local-path/${STORAGE_CLASS}/g" > kubegems.yaml

kubectl apply -f kubegems.yaml
```

More informations refer to <https://www.kubegems.io/docs/installation/quick-install>

## Contributing

If you find this project useful, help us:

- Support the development of this project and star this repo! ⭐
- If you use the KubeGems in a production environment, add yourself to the list of production [adopters](./ADOPTERS.md) 👌
- Help new users with issues they may encounter 🙋
- Send a pull request with your new features and bug fixes 🚀

We very much welcome your contribution and sharing in the KubeGems community in the fields of platform experience, standardized application, plugin sharing, etc.

More information refer to [CONTRIBUTING.md](https://github.com/kubegems/kubegems/blob/main/CONTRIBUTING.md).

## License

Apache License 2.0, see [LICENSE](https://github.com/kubegems/kubegems/blob/main/LICENSE).
