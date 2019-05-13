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

