# helm-schema

Generate `values.schema.json` from `values.yaml` for kubegems charts and plugins.

## Install

```sh
go install kubegems.io/kubegems/tools/helm-schema@latest
```

or install from local source code

```sh
go install ./tools/helm-schema
```

## Usage

```sh
$ helm-schema deploy/plugins/kubegems
Reading deploy/plugins/kubegems/values.yaml
Writing deploy/plugins/kubegems/values.schema.json
```

## Example

See: [test/values.yaml](test/values.yaml)
