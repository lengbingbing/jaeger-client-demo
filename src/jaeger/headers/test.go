package main

import (
	"github.com/opentracing/opentracing-go"
	"fmt"

	"io"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"net/http"
	"github.com/opentracing/opentracing-go/ext"
)

func main() {

	// 初始化配置
	tracer, closer := Init("jaeger-console-header-config-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)//StartspanFromContext创建新span时会用到


	http.HandleFunc("/header", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("header", ext.RPCServerOption(spanCtx))


		defer span.Finish()


		w.Write([]byte("自定义调试追踪请求Header"))
	})

	panic(http.ListenAndServe(":11000", nil))

}
//初始化Go-client,开启自定义调试追踪请求Header
//设置:JaegerDebugHeader="customHeaderKey" 方法用途
// 1. 在发起的Http请求的Header中添加  customHeaderKey:test
// 2. 在jaeger 的 UI 的界面可以用 jaeger-debug-id:test 快速查看自己关心的数据

func Init(service string) (opentracing.Tracer, io.Closer) {




	customHeaders := &jaeger.HeadersConfig{
		JaegerDebugHeader:        "customHeaderKey",

	}

	customHeaders.ApplyDefaults()

	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{

			//在内存队列中保存的span个数，超过阈值保存到后端存储
			QueueSize : 1,
			//上报服务器地址
			LocalAgentHostPort:"127.0.0.1:6831",
			//是否开启 LoggingReporter
			LogSpans: true,
			CollectorEndpoint:"",
			User:"",
			Password:"",

		},
		Headers: customHeaders,

	}
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}
