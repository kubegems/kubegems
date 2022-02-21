# Code Style Guide

## 简介

本指南的目标是通过详细描述在编写代码时的注意事项。这些规则的存在是为了保持代码库的统一性，可管理性，可维护性。

本指南对 Go 的一般准则：

- [Effective Go](https://go.dev/doc/effective_go)
- [CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments)

以及代码风格限定工具,例如 `go fmt`,`go vet` 做了重点提取和补充，同时包含了一些最佳实践。

> 你应当先阅读上述准则，再阅读本准则。

## IDE 支持

这些格式指南大部分都可以通过设置适当的 linter 进行限制。需要将编辑器设置为：

- 在保存时运行 [gofumports](https://github.com/mvdan/gofumpt)
- 错误检查使用 [golangci-lint](https://github.com/golangci/golangci-lint)

若使用 vscode 则推荐使用以下设置：

```json
{
  // golang
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.useLanguageServer": true,
  "gopls": {
    "formatting.gofumpt": true
  },
  "go.addTags": {
    "transform": "camelcase"
  }
}
```

本指南中的大部分设置均已经配置进入 [.golangci.yaml](../.golangci.yaml) 中

并且你可以在本地使用或者手动运行：

```sh
make check
```

进行格式检查。

## 命名约定

变量/结构体命名遵循简洁，清晰的原则。言简意赅的表述其具体用途。

### 包命名

### 文件命名

对于 go 文件，文件命名**必须**使用完全小写。多个单词文件名时**建议**不分割(条件编译除外)，接受以"-"或"\_"分隔。

```txt
// 推荐
httpbackend.go
filebackend.go
ftpbackend.go
```

一个包内若包含多个文件的，**建议**将主要的文件与包名称相同。方便读者快速阅读。

```txt
backend
├── file.go
├── http.go
├── ftp.go
└── backend.go // 主文件
```

### 变量命名

- **禁止**无意义的缩写，拼音，模糊不清。
- **必须**使用 `camalCase` 风格命名.

```go
var dns string // 众所周知的缩写
var wordCount int // 表意明确
var i int // 常用于loop 变量
```

```go
var a1 int // 无意义
var count int // 不明确
var pinyin int // 拼音
var jkxl int // 不知名缩写
```

包内的变量/结构名称前/后缀**建议**不与该包名相同，若语义上相同时应当去掉前/后缀以保持简洁。

```go
package backend

// 正确
// use as: backend.File
type File struct {
    Name string
}

// 错误
// use as: backend.BackendFile, backend.Backend 重复表意
type BackendFile struct {
    FileName string // File前缀冗余
}
// 错误
// use as: backend.FileBackend，backend.Backend 重复表意
type FileBackend struct {}
```

## 准则

### 结构体 tag

对于需要被序列化/反序列化进行传递的结构：

- **必须**增加 tag。
- tag 名称**必须**使用 `lowerCamelCase` 方式进行命名。
- 对于固有缩写例如 ID,HTTP,JSON 等，若其为第一个单词则该词全部小写，否则保持原有格式。

```go
// 正确
type Foo struct {
	ID      string `json:"id,omitempty"`
	BarHTTP string `json:"barHTTP,omitempty"`
}
// 错误
type Bar struct {
	ID      string `json:"ID,omitempty"`
	BarHTTP string `json:"barHttp,omitempty"`
}
```

### 错误消息

- 错误消息封装时应当避免无意义的文字，例如 "failed to","error:" .
- 错误消息封装时应当使用 `%w` 而非 `%v`,[Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- 避免使用 `github.com/pkg/errors.Wrap`

```go
// 正确
fmt.Errorf("create file:%v",err)// errors.Error() -> "process: create file: file exists"
// 错误
fmt.Errorf("failed to create:%v",err)// errors.Error() -> "failed to process: failed to create: file exists"
```

### 全局变量

对于全局变量:

- 除非必要，**禁止** 使用全局变量。全局变量使用带来了不明确的依赖初始化顺序。
- 限制全局单例使用，若存在依赖，则将依赖内置。

```go
// 正确
package consumer

type Consumer struct {
    backend backend.Backend
}

func NewConsumer(backend backend.Backend) *Consumer{
    return &Consumer{
        backend backend
    }
}

func (c *Consumer)do(){
    c.backend.Call()
}
```

```go
// 错误
package backend

var DefaultBackend Backend // 无法确保在使用时值已经被初始化

func Initialize() {
    DefaultBacked = somebackend()
}
// other package
package consumer

func do() {
    backend.DefaultBackend.Call() // nil pointer
}
```

### 禁止 panic

无论何时都没有理由将 panic 用作错误处理,对于能够捕获的错误需要遵循[golang 错误处理](https://go.dev/doc/effective_go#errors) 就地处理或者向上传递。

```go
// 正确
func run() error{
  if err:=somefunc();err!=nil{
    return fmt.Errorf("somefunc %w",err)
  }
  ...
}
```

```go
// 错误
func run() {
  if err:=somefunc();err!=nil{
    panic(err)
  }
}
```

### 使用 context

- 尽量使用带 context 的方法。如 `http.NewRequestWithContext()`.
- 避免直接使用 context.Backend(),context.TODO()，而使用 context 树。
- 需要长时运行的方法定义时需要增加 context 且能够正确处理 context cancel 并退出。
- 相比于直接使用 channel 应当更倾向于使用 context 进行终止控制。

context 不仅用来传递上下文消息，还用来控制执行。例如当 http 请求客户端取消请求时，context 会被 cancel,此时服务端应当终止与该次请求有关的所有处理。
在执行一个长时运行方法时，也可通过 context 可以快速取消。

### 带缓存的 channel

- **限制**使用缓存大于 1 的 channel，一般情况下不需要使用 channel 进行缓存，如果需要缓存则应该仔细考虑 channel 大小和缓存满时可能发生的情况。

### 使用 duration 而非 int

- 在使用到时间长短时使用 time.Duration,避免使用 int;time.Duration 在 flag 上支持已经非常完善。
