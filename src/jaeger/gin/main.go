package main

import (
	"net/http"
	"io/ioutil"
	"github.com/gin-gonic/gin"
	"jaeger/lib/config"
	"github.com/opentracing/opentracing-go"
	"jaeger/gin/ginhttp"
)

func main() {
	//初始化Jaeger
	tracer, closer := config.Init("jaeger-gin-http")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
	router := gin.Default()
	//使用Jaeger中间件
	router.Use(ginhttp.Middleware(tracer))
	{
		//simple tracer
		router.GET("/tracer", func(c *gin.Context) {
			tracer.StartSpan("tracer")
			//获取Span
			span := opentracing.SpanFromContext(c.Request.Context())
			for k, v := range c.Request.Header {
				//写入http--header里的数据
				span.SetTag(k, v)

			}
			c.String(http.StatusOK, "Hello World")
		})
		//需要调用其他站点的方法、写入child_tracer
		router.GET("/child_tracer", func(c *gin.Context) {
			span := opentracing.SpanFromContext(c.Request.Context())
			//写入tag
			span.SetTag("operation_name", "log")
			//写入日志
			span.LogEvent("Hello, world!")
			//需要访问的其他站点服务
			url := "http://simu.openapi.autohome.com.cn/flowcontrol/test/gettest"
			ctx := opentracing.ContextWithSpan(c.Request.Context(), span)
			//创建child_span 记录访问目标站点时、需要记录的信息
			child_span, _ := opentracing.StartSpanFromContext(ctx, "flowcontrol/gettest")
			child_span.SetTag("http.url", url)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				panic(err.Error())
			}
			client := http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err.Error())
			}
			//记录返回结果
			child_span.SetTag("http.code", resp.Status)
			child_span.Finish()
			body, err := ioutil.ReadAll(resp.Body)
			respStr := string(body)
			//输出返回内容
			c.String(http.StatusOK, respStr)

		})

	}
	//绑定站点端口
	router.Run(":8000")
}
