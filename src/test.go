package main

import (
	"tfe"
	"fmt"

	"net/http"
	"time"
)

func main() {
	h := &tfe.HttpService { &http.Transport {} }
	ht := tfe.NewServiceWithTimeout( h, 1000 * time.Millisecond )
	hht := tfe.NewServiceWithHistory ( ht, "http", nil )
	c := tfe.NewCluster([]*tfe.ServiceWithHistory{hht}, "test")

	req, _ := http.NewRequest("GET", "http://google.com/", nil)
	rsp, err := c.Serve(req)
	fmt.Printf("rsp:%v, err:%v", rsp, err)
}
