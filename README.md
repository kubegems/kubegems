# kubegem

## Getting started

[quick-start](docs/quick-start.md)

## Development

### Run local
1. make build
2. ./bin/kubegems service gencfg > config/config.yaml
3. prepare file `certs/jwt/tls.crt` and `certs/jwt/tls.key`
4. ./bin/kubegems service

### Debug by vscode
```json
{
    "name": "service",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/cmd",
    "cwd": "${workspaceFolder}", // 不然找不到证书文件
    "args": ["service"]
}
```