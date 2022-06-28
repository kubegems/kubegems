# Kubegems

[![.github/workflows/build.yml](https://github.com/kubegems/kubegems/actions/workflows/build.yml/badge.svg)](https://github.com/kubegems/kubegems/actions/workflows/build.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/kubegems/kubegems.svg?maxAge=604800)](https://hub.docker.com/r/kubegems/kubegems)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubegems/kubegems)](https://goreportcard.com/report/github.com/kubegems/kubegems)

![Kubegems](https://www.kubegems.io/img/logo.svg)

Visit [kubegems.io](https://kubegems.io) for the full documentation,
examples and guides.

KubeGems is a general-purpose open source PaaS cloud management platform built around Kubernetes through self-developed and integrated cloud native projects. At present, the core functions of KubeGems have preliminary unified management in multi-cloud and multi-tenant scenarios. And through the plug-in method, the user interface can flexibly control the enabling and closing of many plug-ins including monitoring system, log system, microservice governance and so on.

As a cloud-native general-purpose cloud platform, KubeGems has taken resource isolation to support multi-cluster and multi-tenant scenarios as its main design goal since its establishment. Users can make tenant-level custom resource planning for the Kubernetes cluster connected to the platform. In addition, we provide a UI interface that is richer and more user-friendly than the native Dashboard, allowing users/enterprises to plan platform metadata according to their own scenarios, without worrying about their business and data confusion. At the same time, KubeGems also provides many rich functional modules to bring a better user experience for individual or enterprise users, such as access control, resource planning, network isolation, tenant gateway, storage volume, observability, user auditing, certificate management , canary release, istio governance and other functions.

## Install

### Install kubernetes cluster
You can Install your k8s cluster using any of the following methods, supported k8s version is 1.18~1.24:
1. [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/)
2. [kind](https://kind.sigs.k8s.io/)
3. [kubekey](https://github.com/kubesphere/kubekey)
4. Any other ways...

### Install kubegems
When your k8s cluster is ready, next you can install kubegems refer to doc: [Install kubegems](https://www.kubegems.io/docs/installation/quick-install)

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/kubegems/kubegems/blob/main/CONTRIBUTING.md).

## License

Apache License 2.0, see [LICENSE](https://github.com/kubegems/kubegems/blob/main/LICENSE).
