package main

import (
	"net/http"

	thrift "github.com/uber/jaeger-client-go/thrift-gen/baggage"
	"encoding/json"
)

func main() {

	// 初始化配置
	http.HandleFunc("/baggageRestrictions", func(w http.ResponseWriter, r *http.Request) {
		var array = []*thrift.BaggageRestriction{&thrift.BaggageRestriction{BaggageKey:"test", MaxValueLength:6}}
		bytes, _ := json.Marshal(array)
		w.Header().Add("Content-Type", "application/json")
		w.Write(bytes)
	})
	panic(http.ListenAndServe(":11000", nil))


}
