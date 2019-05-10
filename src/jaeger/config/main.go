package main

import (
	"log"
	"os"
	
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"

	"io"
	"fmt"
	"context"
	"time"
)



func main() {

	tracer, closer := initJaeger("jaeger-client-demo")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)//StartspanFromContext创建新span时会用到

	span := tracer.StartSpan("span_root")
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	r1 := foo3("Hello foo3", ctx)
	r2 := foo4("Hello foo4", ctx)
	fmt.Println(r1, r2)
	span.Finish()



}

func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := &jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,

		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
			QueueSize:1,
			LocalAgentHostPort:"127.0.0.1:6831",
		},

	}



	cfg.ServiceName = service
	cfg.Disabled =false


	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))


	//tracer, closer, err := cfg.New(service, jaegercfg.Logger(jaeger.StdLogger))


	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func foo3(req string, ctx context.Context) (reply string){
	//1.创建子span
	span, _ := opentracing.StartSpanFromContext(ctx, "span_foo3")
	defer func() {
		//4.接口调用完，在tag中设置request和reply
		span.SetTag("request", req)
		span.SetTag("reply", reply)
		span.SetBaggageItem("aa","ppc")

		span.SetTag("sql","select * from public where id = 1")
		span.Finish()

	}()

	println(req)
	//2.模拟处理耗时
	time.Sleep(time.Second/2)
	//3.返回reply
	reply = "foo3Reply"
	return
}
//跟foo3一样逻辑
func foo4(req string, ctx context.Context) (reply string){
	cc:= ctx.Value("aa")
	println(cc)
	span, _ := opentracing.StartSpanFromContext(ctx, "span_foo4")
	defer func() {
		span.SetTag("request", req)
		span.SetTag("reply", reply)
		aa := span.BaggageItem("aa")
		println(aa)
		span.Finish()
	}()

	println(req)
	time.Sleep(time.Second/2)
	reply = "foo4Reply"
	return
}

func ExampleConfiguration_InitGlobalTracer_testing() {
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
			LocalAgentHostPort:"127.0.0.1:6831",
		},

	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory


	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		"serviceName",
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)


	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}

	defer closer.Close()

	// continue main()
}

func ExampleConfiguration_InitGlobalTracer_production() {
	// Recommended configuration for production.
	cfg := jaegercfg.Configuration{



	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(

		"serviceName",

		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	// continue main()
}

func ExampleFromEnv() {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	opentracing.SetGlobalTracer(tracer)
	// continue main()
}

func ExampleFromEnv_override() {
	os.Setenv("JAEGER_SERVICE_NAME", "not-effective")

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	cfg.ServiceName = "this-will-be-the-service-name"

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	opentracing.SetGlobalTracer(tracer)
	// continue main()
}