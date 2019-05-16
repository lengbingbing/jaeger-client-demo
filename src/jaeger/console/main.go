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
