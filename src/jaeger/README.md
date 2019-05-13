
# 使用 Jaeger tracing 方案追踪 Go 服务

* Jaeger 是由 Uber 开发的一套全链路追踪方案，符合 Opentracing 协议规范。

## 1.Go环境安装

### Mac 平台安装 Golang 方法
```shell
`brew update && brew upgrade && brew install go`
```

### Linux 平台安装 Golang 方法

`下载 Linux 平台的 Golang 安装包`

```shell
#解压缩到指定目录
tar -C /usr/local -xzf go$VERSION.$OS-$ARCH.tar.gz
# 配置 PATH 变量
* export PATH=$PATH:/usr/local/go/bin
```

### Windows 平台安装 Golang 方法

[下载 Windows 平台的 Golang 安装包](https://golang.org/dl/)

解压安装，并配置环境变量即可。


## 2.工具包引用

建议通过glide 管理项目的依赖管理 [glide](https://github.com/Masterminds/glide)

For example:

```yaml
- package: github.com/uber/jaeger-client-go
  version: ^2.7.0
```

通过 go get 命令直接安装最新的客户端版本

```shell
go get -u github.com/uber/jaeger-client-go/
cd $GOPATH/src/github.com/uber/jaeger-client-go/
git submodule update --init --recursive
make install
```

## 3.目录结构如下图
![目录结构如下图](https://github.com/lengbingbing/jaeger-client-demo/blob/master/src/jaeger/pic/structure.png)

进入 lib/config/init.go 初始化 Jaeger client，Init 方法是 Jaeger client 配置方法，所有的demo程序初始化 Jaeger client 时，均需要调用此方法。

console 目录下是控制台 Go 的应用程序调用示例，服务启动命令 > `go run main.go`