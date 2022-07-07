<div style="text-align: center"></div>
  <p align="center">
  <img src="https://www.kubegems.io/img/logo.svg" width="40%" height="40%">
      <br>
      <i>Let cloudnative management more easily.</i>
  </p>
</div>

[![kubegems](https://jaywcjlove.github.io/sb/lang/chinese.svg)](README_zh.md)
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
  <summary><b>multi-tenancy</b></summary>
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
  <summary><b>Appstore based on istio</b></summary>
</details>

<details>
  <summary><b>Smart ML(Machine Learning) Models Serving</b></summary>
</details>

## Snapshots

<br/>
<table>
    <tr>
      <td width="50%" align="center"><b>Tenant</b></td>
      <td width="50%" align="center"><b>WorkSpace</b></td>
    </tr>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/tenant.jpg?raw=true"></td>
        <td width="50%" align="center"><img src="https://github.com/kubegems/.github/blob/master/static/image/workspace.jpg?raw=true"></td>
    </tr>
    <tr>
      <td width="50%" align="center"><b>Cluster</b></td>
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

>account: admin    password: demo!@#admin

## Getting started

### Install kuberneotes cluster

You can Install your k8s cluster using any of the following methods, supported k8s version is v1.20 +
1. [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/)
2. [kind](https://kind.sigs.k8s.io/)
3. [kubekey](https://github.com/kubesphere/kubekey)
4. Any other ways...

### Installing KubeGems

When your k8s cluster is ready, next you can install kubegems insatller operator on your cluster.

```
kubectl create namespace kubegems-installer
kubectl apply -f https://github.com/kubegems/kubegems/raw/main/deploy/installer.yaml
```

Intallter kubegems with installer operator.

```
kubectl create namespace kubegems

export STORAGE_CLASS=local-path  # export your  storageClass
export KUBEGEMS_VERSION=v1.22.0  # change to specify kubegems version
curl -sL https://raw.githubusercontent.com/kubegems/kubegems/main/deploy/kubegems.yaml \
| sed -e "s/local-path/${STORAGE_CLASS}/g" -e "s/latest/${KUBEGEMS_VERSION}/g" \
> kubegems.yaml

kubectl apply -f kubegems.yaml
```

more information refer to https://www.kubegems.io/docs/installation/quick-install

## Contributing

We very much welcome you to participate in the contribution and sharing of platform experience, standardized applications, plug-in sharing and other fields in the kubegems community.

If you are a user who s using KubeGems, and you have a deep understanding of kubegems and agree with the technical route, and there is a great demand within your enterprise, we welcome you to participate in the development of kubegems project.

More information Refer to [CONTRIBUTING.md](https://github.com/kubegems/kubegems/blob/main/CONTRIBUTING.md).

## Let us know who is using KubeGems

You can submit issues to tell us about your case.

## License

Apache License 2.0, see [LICENSE](https://github.com/kubegems/kubegems/blob/main/LICENSE).
