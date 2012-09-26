package main

import (
	"net/http"
	"io"
	"fmt"
)

type Server int
var (
	server Server
	client http.Client
)

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "http://www.google.com/", nil)
	fmt.Printf("%v", err)
	rsp, _ := client.Do(req)
	w.WriteHeader(200)
    io.Copy(w, rsp.Body)
}
func main() {
	http.Handle("/test", &server)
	http.ListenAndServe(":8888", nil)
}
