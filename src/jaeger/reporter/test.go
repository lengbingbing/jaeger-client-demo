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
	tracer, closer := Init("jaeger-console-reporter-config-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)//StartspanFromContext创建新span时会用到


	http.HandleFunc("/reporter", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("reporter", ext.RPCServerOption(spanCtx))


		defer span.Finish()


		w.Write([]byte("reporter 上报追踪信息配置"))
	})

	panic(http.ListenAndServe(":11000", nil))

}
//初始化Go-client,提交数据配置
// ReporterConfig

func Init(service string) (opentracing.Tracer, io.Closer) {


	customHeaders := &jaeger.HeadersConfig{
		JaegerDebugHeader:        "custom-jaeger-debug-header",
		JaegerBaggageHeader:      "custom-jaeger-baggage-header",
		TraceContextHeaderName:   "custom-tracer-state-header-name",
		TraceBaggageHeaderPrefix: "custom-tracer-baggage-header-prefix",
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
