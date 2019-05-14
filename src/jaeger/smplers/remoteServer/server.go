package main

import (
	"net/http"
)

func main() {



	http.HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {

	})
	panic(http.ListenAndServe(":11001", nil))

}

