package main

import (
	"net/http"
	"net/url"
)

type defaultHTTPClient struct {
	*http.Client
	baseURL string
}

func (c *defaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	req.URL = baseURL.ResolveReference(req.URL)
	return c.Client.Do(req)
}
