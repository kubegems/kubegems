# KubeGems Charts

kubegems charts for installer.

## Deployment

set up local helm repo

```sh
helm repo index .
python3 -m http.server
helm repo add local http://127.0.0.1:8000
```

```sh
helm dep update
```

## Build

```sh
sh pull-charts.sh
```
