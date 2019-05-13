
# 使用 Jaeger tracing 方案追踪 Go 服务

* Jaeger 是由 Uber 开发的一套全链路追踪方案，符合 Opentracing 协议规范。



# 1.环境准备


`syntax: httpc = http.new()`

Creates the http object. In case of failures, returns `nil` and a string describing the error.

## Mac 平台安装 Golang 方法

`brew update && brew upgrade && brew install go`


## Linux 平台安装 Golang 方法

`下载 Linux 平台的 Golang 安装包`

* # 解压缩到指定目录
* tar -C /usr/local -xzf go$VERSION.$OS-$ARCH.tar.gz
* # 配置 PATH 变量
* export PATH=$PATH:/usr/local/go/bin

