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
	tracer, closer := InitProbabilistic("jaeger-console-sampler-probabilistic-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)


	//test
	http.HandleFunc("/probabilistic", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("probabilistic", ext.RPCServerOption(spanCtx))
		defer span.Finish()
		w.Write([]byte("数据采样算法采样随机采样率"))
	})

	panic(http.ListenAndServe(":11000", nil))
}


//初始化Go-client 采样配置为随机采样
// ConstSampler
// Type =  probabilistic
// Param = 0-1 之间 随机采样，每个请求都有一定的概率被采样

func InitProbabilistic(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "probabilistic",
			Param: 0.5,
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