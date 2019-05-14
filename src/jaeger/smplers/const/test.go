package main

import (
	"github.com/opentracing/opentracing-go"
	"fmt"

	"io"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"time"
)

func main() {

	// 初始化配置
	tracer, closer := InitSmplerConst("jaeger-console-sampler-const-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)//StartspanFromContext创建新span时会用到

	span := tracer.StartSpan("span_root")
	//2.模拟处理耗时
	time.Sleep(time.Second/2)
	span.Finish()



}
//初始化Go-client 采样配置为全量采集
// ConstSampler
// Type =  const
// Param = 1 全量收集采样数据
// Parm  = 0 不收集采样数据
func InitSmplerConst(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
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
