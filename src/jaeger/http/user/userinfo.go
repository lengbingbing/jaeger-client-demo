package main

import (
	"net/http"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"jaeger/lib/config"
	"time"
)

func main() {


	tracer, closer := config.Init("jaeger-http-userServices")
	defer closer.Close()
	http.HandleFunc("/getUserById", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("getUserById", ext.RPCServerOption(spanCtx))
		defer span.Finish()
		//模拟处理耗时
		time.Sleep(time.Second/2)
		helloStr := "{'userId:1','nick':'autohome'}"
		println(helloStr)
		w.Write([]byte(helloStr))
	})

	panic(http.ListenAndServe(":10008", nil))


}



//"调用其他平台的API接口"
//with tracing.tracer.start_span('test gettest', child_of=span) as scope:
//url = 'http://simu.openapi.autohome.com.cn/flowcontrol/test/gettest'
//scope.set_tag('http.url', url)
//scope.set_operation_name('flowcontrol/gettest')
//r = requests.get(url)
//scope.set_tag('http.code', r.status_code)
