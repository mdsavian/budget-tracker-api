package main

import (
	"fmt"
	"net/http"
)

func hello(w http.ResponseWriter, req *http.Request){
	fmt.Fprintf(w,"Hello World\n")
}


//func headers (w http.ResponseWriter, req *http.Request){

//}

func main() {

	http.HandleFunc("/hello", hello)

	http.ListenAndServe(":8090", nil)
}
