[![.github/workflows/build.yml](https://github.com/kubegems/gems/actions/workflows/build.yml/badge.svg)](https://github.com/kubegems/gems/actions/workflows/build.yml)

# kubegem

under construction ... 🚧 🚧 🚧

## Getting started

[quick-start](docs/quick-start.md)

## Development

### Run local

kubegems have 5 components:
- service: provide kubegems api server.
- msgbus: provide instant communication for `service`, `agent` and `dashboard`.
- worker: execute long time task.
- agent: proxy all request by service in a single cluster.
- controller: reconcile all kubegems CRD requests.

Choose one of these component you want to run, then:

1. prepare certs: `cd scripts && bash generate-tls-certs.sh`
2. `make build` 
3. `./bin/kubegems {component} gencfg > config/config.yaml`
4. Modify `config/config.yaml` yourself, for different component, config.yaml is different, you can also use args or enironment variables.
5. `./bin/kubegems {conpoment}`

### Debug by vscode

```json
{
  "name": "service",
  "type": "go",
  "request": "launch",
  "mode": "debug",
  "program": "${workspaceFolder}/cmd",
  "cwd": "${workspaceFolder}", 
  "args": ["service"] // may also be msgbus, worker, agent, controller
}
```
