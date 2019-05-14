package main

import (
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go"
	"net/http"
	"io"
	"github.com/uber/jaeger-client-go"
	"fmt"
	"github.com/uber/jaeger-client-go/config"
)

func main() {
	tracer, closer := InitRateLimiting("jaeger-console-sampler-remote-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)


	//test
	http.HandleFunc("/remote", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("remote", ext.RPCServerOption(spanCtx))
		defer span.Finish()
		w.Write([]byte("采样算法根据远程服务器动态设置，并通过轮询外部服务器定期更新它。这允许对采样策略进行动态控制"))
	})

	panic(http.ListenAndServe(":11000", nil))
}


//初始化Go-client 采样算法根据远程服务器动态设置，并通过轮询外部服务器定期更新它。这允许对采样策略进行动态控制
// ConstSampler
// Type =  remote
// SamplingServerURL =http://127.0.0.1:11001/server   采样服务器地址

func InitRateLimiting(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "remote",

			SamplingServerURL:"http://127.0.0.1:11001/server",

		},
		Reporter: &config.ReporterConfig{
			LocalAgentHostPort:"127.0.0.1:6831",
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

