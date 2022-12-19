# Upgrade from 1.22 to 1.23

## Usage

Run on manager cluster:

```sh
# switch kubeconfig current context to target cluster
# run migrate
go run ./scripts/release-1.23-update --manager --kubegemsVersion v1.23.0[-xxx]
```

or specify the context name in kubeconfig:

```sh
go run ./scripts/release-1.23-update --context <context_name> --manager --kubegemsVersion v1.23.0[-xxx]
```

Run on per agent cluster(include manager cluster):

```sh
# switch kubeconfig context to target cluster
go run ./scripts/release-1.23-update --agent --kubegemsVersion v1.23.0[-xxx]
# apply 1.23 installer
kubectl apply -f https://github.com/kubegems/kubegems/raw/release-1.23/deploy/installer.yaml
```
