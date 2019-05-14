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
	tracer, closer := InitRateLimiting("jaeger-console-sampler-rateLimiting-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)


	//test
	http.HandleFunc("/rateLimiting", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("rateLimiting", ext.RPCServerOption(spanCtx))
		defer span.Finish()
		w.Write([]byte("用于只允许每秒采样一定数量的记录道 ,每秒只有5个请求会被采样"))
	})

	panic(http.ListenAndServe(":11000", nil))
}


//初始化Go-client 用于只允许每秒采样一定数量的记录道
// ConstSampler
// Type =  rateLimiting
// Param =5  用于只允许每秒采样一定数量的记录道 ,每秒只有5个请求会被采样
// 测试命令 ab -c 1 -n 100  http://127.0.0.1:11000/rateLimiting;
func InitRateLimiting(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "rateLimiting",
			Param: 5,
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

