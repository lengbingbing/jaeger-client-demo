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

##  快速开始

### 一、常用Gin框架快速接入到全链路跟踪


```go

package main

import (
	"net/http"
	"io/ioutil"
	"github.com/gin-gonic/gin"
	"jaeger/lib/config"
	"github.com/opentracing/opentracing-go"
	"jaeger/gin/ginhttp"
)

func main() {
	//初始化Jaeger
	tracer, closer := config.Init("jaeger-gin-http")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
	router := gin.Default()
	//使用Jaeger中间件
	router.Use(ginhttp.Middleware(tracer))
	{
		//simple tracer
		router.GET("/tracer", func(c *gin.Context) {
			tracer.StartSpan("tracer")
			//获取Span
			span := opentracing.SpanFromContext(c.Request.Context())
			for k, v := range c.Request.Header {
				//写入http--header里的数据
				span.SetTag(k, v)

			}
			c.String(http.StatusOK, "Hello World")
		})


	}
	//绑定站点端口
	router.Run(":8000")
}



```
##  demo example
[Gin接入完整Demo](gin/main.go)

### 二、Http请求调用

```go
package main

import (
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go"
	"net/http"
	"io/ioutil"
	"github.com/opentracing/opentracing-go/log"
	"fmt"
	"context"
	"jaeger/lib/config"
)

func main() {
	tracer, closer := config.Init("jaeger-http-client")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)


	//发送站内短信接口
	http.HandleFunc("/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("sendMessage", ext.RPCServerOption(spanCtx))

		userIds, ok := r.URL.Query()["userIds"]
		var  verify  bool  = true
		var  result  string
		if !ok || len(userIds) < 1 {

			span.LogFields(
				log.String("Param", "Url Param 'key' is missing"),

			)
			result = "Url Param 'key' is missing"
			verify = false

		}
		//验证参数是否正确
		if verify{

			span.SetTag("userIds", userIds)
			ctx := opentracing.ContextWithSpan(context.Background(), span)
			//通过Http 调用后去用户信息接口
		    userInfo:=getUserById(userIds[0],ctx)
			w.Write([]byte(userInfo))

		}else{
			w.Write([]byte(result))
		}
		defer span.Finish()
	})

	panic(http.ListenAndServe(":10007", nil))
}


//通过Http 调用后去用户信息接口
func getUserById(userIds string,ctx context.Context) string {
	span, _ := opentracing.StartSpanFromContext(ctx, "getUserById")
	defer span.Finish()
	url := "http://localhost:10008/getUserById"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err.Error())
	}
	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	respStr := string(body)
	span.LogFields(

		log.String("userIds", userIds),
	)
	fmt.Println(respStr)
	return respStr
}

```
http/client/client.go 源码是对外提供服务可以发送站内短信的方法(sendMessage) Web站点服务，监听 10007 端口，对外接口 http://127.0.0.1:10007/sendMessage?userIds=1，
服务启动命令 `go run main.go`

http/user/userinfo.go 源码是调用用户接口查询用户基本信息的站点，提供 getUserById 方法根据用户Id获取用户信息的服务,监听 10008 端口，对外接口 http://127.0.0.1:10008/getUserById，
服务启动命令 `go run main.go`

##  demo example
[Http接入完整Demo](http/client/client.go)


### 三、手动埋点
```go
package main

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"time"
	"fmt"
	"jaeger/lib/config"
)

func main() {
	// 初始化配置
	tracer, closer := config.Init("jaeger-console-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)//StartspanFromContext创建新span时会用到
	span := tracer.StartSpan("span_root")
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	//执行任务
	r1 := getPublicById("100", ctx)
	fmt.Println(r1)
	span.Finish()
}

func getPublicById(id string, ctx context.Context) (reply string){
	//1.创建子span
	span, _ := opentracing.StartSpanFromContext(ctx, "getPublicById")
	defer func() {
		//模拟调用数据库查询
		span.SetTag("sql","select * from public where id = 100")
		span.SetBaggageItem("test","123")
		span.Finish()

	}()
	println(id)
	//2.模拟处理耗时
	time.Sleep(time.Second/2)
	//3.返回
	reply = "未找到信息"
	return
}

```
进入 lib/config/init.go 初始化 Jaeger client，Init 方法是 Jaeger client 配置方法，所有的demo程序初始化 Jaeger client 时，均需要调用此方法。
console 目录下是控制台 Go 的应用程序调用示例，服务启动命令 > `go run main.go`
http 目录下是Web 站点下 Go 的应用程序调用示例

##  demo example
[手动埋点完整Demo](console/main.go)





## Jaeger client 初始化更多配置
```
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			CollectorEndpoint: "http://127.0.0.1:14268/api/traces",
			LogSpans: true,
		},
	}


```
[lib/config/init.go](./lib/config/init.go).

### Configuration 数据结构
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
    // Sampler.type = const （是否全量采集)  Sampler.Param = 1 全量采集   Sampler.Param = 0 不采集任何追踪信息
    // Sampler.type = probabilistic （概率采集，默认万份之一)
    // Sampler.type = rateLimiting （限速采集，每秒只能采集一定量的数据)
    // Sampler.type = remote （一种动态采集策略，根据远程站点的配置采集策略)
    Sampler             *SamplerConfig             `yaml:"sampler"`

    // 上报追踪信息到指定的服务器地址配置
    // Reporter.LocalAgentHostPort    提交上报追踪信息的服务器地址信息
	Reporter            *ReporterConfig            `yaml:"reporter"`

	Headers             *jaeger.HeadersConfig      `yaml:"headers"`

	BaggageRestrictions *BaggageRestrictionsConfig `yaml:"baggage_restrictions"`

	Throttler           *ThrottlerConfig           `yaml:"throttler"`
}
```

##  Sampler 采样算法配置的配置Demo

- [const 全量采集实例](./smplers/test.go)
- [probabilistic 概率采集实例](./smplers/probabilistic/test.go)
- [rateLimiting 限速采集实例](./smplers/rateLimiting/test.go)
- [remote 动态采集策略实例]



##   Reporter配置Demo

- [Reporter配置实例](./reporter/test.go)


##   Headers配置Demo

- [初始化配置，添加自定义Header快速查看Debug日志](./headers/test.go)


##   baggage 随行数据，指定Key、Value的字符长度限制Demo

- [BaggageRestrictions配置实例](./baggage/server/test.go)