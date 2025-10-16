package main

import "net/http"

type defaultHTTPClient struct {
	*http.Client
}

func (c *defaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.Client.Do(req)
}
