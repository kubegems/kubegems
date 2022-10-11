# Contributing

Kubegems uses GitHub to manage reviews of pull requests.

- If you don't known kubegems components, see [Components](#Components)

- If you want to know how to run and develop in local, see [Development](#Development)

## Components

- service: provide kubegems api server.
- msgbus: provide instant communication for `service`, `agent` and `dashboard`.
- worker: execute long time task.
- agent: proxy all request by service in a single cluster.
- controller: reconcile all kubegems CRD requests.
- installer: manage kubegems plugins install/update/uninstall.

## Development

### Build

Run `make build-binaries`

Then you can run `./bin/kubegems -h` to see how to run each of these components.

### Debug locally

See [local-development.md](docs/local-development.md)

## Pull Request Checklist

To be completed...

## Dependency management

The kubegems project uses [Go modules](https://golang.org/cmd/go/#hdr-Modules__module_versions__and_more) to manage dependencies on external packages.

To add or update a new dependency, use the `go get` command:

```bash
# Pick the latest tagged release.
go get example.com/some/module/pkg

# Pick a specific version.
go get example.com/some/module/pkg@vX.Y.Z
```

Tidy up the `go.mod` and `go.sum` files:

```bash
# The GO111MODULE variable can be omitted when the code isn't located in GOPATH.
GO111MODULE=on go mod tidy
```

You have to commit the changes to `go.mod` and `go.sum` before submitting the pull request.
