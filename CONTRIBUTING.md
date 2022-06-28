# Contributing

Kubegems uses GitHub to manage reviews of pull requests. It has 6 components in it:

* If you don't known it's components, see [Components](#Components)

* If you want to know how to run and develop in local, see [Development](#Development)


## Components

* If you 
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

### Run local

By example, if you want to run `service`, you can:

1. `./bin/kubegems service gencfg > config/config.yaml`
2. Modify `config/config.yaml` yourself, for different component, config.yaml is different, you can also use args or enironment variables.
3. `./bin/kubegems service`

### Debug by vscode

```json
{
  "name": "service",
  "type": "go",
  "request": "launch",
  "mode": "debug",
  "program": "${workspaceFolder}/cmd",
  "cwd": "${workspaceFolder}", 
  "args": ["service"] // may also be msgbus, worker, agent, controller, installer
}
```


* If you are a new contributor see: [Steps to Contribute](#steps-to-contribute)

* If you have a trivial fix or improvement, go ahead and create a pull request,
  addressing (with `@...`) a suitable maintainer of this repository (see
  [MAINTAINERS.md](MAINTAINERS.md)) in the description of the pull request.

* If you plan to do something more involved, first discuss your ideas
  on our [mailing list](https://groups.google.com/forum/?fromgroups#!forum/prometheus-developers).
  This will avoid unnecessary work and surely give you and us a good deal
  of inspiration. Also please see our [non-goals issue](https://github.com/prometheus/docs/issues/149) on areas that the Prometheus community doesn't plan to work on.

* Relevant coding style guidelines are the [Go Code Review
  Comments](https://code.google.com/p/go-wiki/wiki/CodeReviewComments)
  and the _Formatting and style_ section of Peter Bourgon's [Go: Best
  Practices for Production
  Environments](https://peter.bourgon.org/go-in-production/#formatting-and-style).

* Be sure to sign off on the [DCO](https://github.com/probot/dco#how-it-works).


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
