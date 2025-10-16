package main

import (
	"net/http"
	"net/url"
	"sync"
)

type defaultHTTPClient struct {
	*http.Client
	baseURL string
	isUp    bool
	mutex   sync.RWMutex
}

func (c *defaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	req.URL = baseURL.ResolveReference(req.URL)
	return c.Client.Do(req)
}

func (c *defaultHTTPClient) IsUp() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isUp
}

func (c *defaultHTTPClient) SetUp(isUp bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.isUp = isUp
}
