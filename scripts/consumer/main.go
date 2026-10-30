package main

import (
	"context"
	"fmt"
	"github.com/EASYSLIP-CO-LTD/sdk-go"
	"io"
	"net/http"
	"strings"
)

type offline struct{}

func (offline) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Header.Get("Authorization") != "Bearer synthetic-consumer-key" {
		panic("missing authorization")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))}, nil
}
func main() {
	c, err := easyslip.NewClient("synthetic-consumer-key", easyslip.WithHTTPClient(&http.Client{Transport: offline{}}))
	if err != nil {
		panic(err)
	}
	defer c.Close()
	if _, err = c.Banks.List(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println("Installed module offline request PASS")
}
