
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

##  demo example

- [console](https://github.com/lengbingbing/jaeger-client-demo/tree/master/src/jaeger/console)
- [http](https://github.com/lengbingbing/jaeger-client-demo/tree/master/src/jaeger/http)


## 3.目录结构如下图


![目录结构如下图](https://github.com/lengbingbing/jaeger-client-demo/blob/master/src/jaeger/pic/structure.png)

进入 lib/config/init.go 初始化 Jaeger client，Init 方法是 Jaeger client 配置方法，所有的demo程序初始化 Jaeger client 时，均需要调用此方法。

console 目录下是控制台 Go 的应用程序调用示例，服务启动命令 > `go run main.go`

http 目录下是Web 站点下 Go 的应用程序调用示例

http/client/client.go 源码是对外提供服务可以发送站内短信的方法(sendMessage) Web站点服务，监听 10007 端口，对外接口 http://127.0.0.1:10007/sendMessage?userIds=1，
服务启动命令 `go run main.go`

http/user/userinfo.go 源码是调用用户接口查询用户基本信息的站点，提供 getUserById 方法根据用户Id获取用户信息的服务,监听 10008 端口，对外接口 http://127.0.0.1:10008/getUserById，
服务启动命令 `go run main.go`




## Jaeger client 初始化

[lib/config/init.go](./lib/config/init.go).

### config.Configuration 常用配置属性
```
type Configuration struct {

	//Jeager 的服务名称
	ServiceName string `yaml:"serviceName"`

	// Disabled can be provided via environment variable named JAEGER_DISABLED
	Disabled bool `yaml:"disabled"`

	// RPCMetrics can be provided via environment variable named JAEGER_RPC_METRICS
	RPCMetrics bool `yaml:"rpc_metrics"`

	// Tags can be provided via environment variable named JAEGER_TAGS
	Tags []opentracing.Tag `yaml:"tags"`

    // 采样算法配置的配置
	Sampler             *SamplerConfig             `yaml:"sampler"`
    // 上报追踪信息到指定的服务器地址配置
	Reporter            *ReporterConfig            `yaml:"reporter"`
	Headers             *jaeger.HeadersConfig      `yaml:"headers"`
	BaggageRestrictions *BaggageRestrictionsConfig `yaml:"baggage_restrictions"`
	Throttler           *ThrottlerConfig           `yaml:"throttler"`
}
```

### 采样算法配置的配置


属性| 描述
--- | ---
Sampler.type="const" 是否全量采集采样数据 |  Sampler.param = 0 （不采集)   Sampler.param = 1 全采集
Sampler.type="const" 是否全量采集采样数据 |  Sampler.param = 0 （不采集)   Sampler.param = 1 全采集
Sampler.type="const" 是否全量采集采样数据 |  Sampler.param = 0 （不采集)   Sampler.param = 1 全采集
Sampler.type="const" 是否全量采集采样数据 |  Sampler.param = 0 （不采集)   Sampler.param = 1 全采集
