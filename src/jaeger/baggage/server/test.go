package main

import (
	"net/http"
)

func main() {

	// 初始化配置
	http.HandleFunc("/baggageRestrictions", func(w http.ResponseWriter, r *http.Request) {


		w.Write([]byte("baggagerKey:test"))
	})

	panic(http.ListenAndServe(":11000", nil))

}