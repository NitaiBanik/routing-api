package health

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	IsUp() bool
	SetUp(isUp bool)
	GetBaseURL() string
}

type DefaultHTTPClient struct {
	*http.Client
	BaseURL string
	Up      bool
	mutex   sync.RWMutex
}

func NewDefaultHTTPClient(baseURL string, requestTimeout, connectTimeout time.Duration) *DefaultHTTPClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: connectTimeout,
		}).DialContext,
		ResponseHeaderTimeout: requestTimeout,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	}

	return &DefaultHTTPClient{
		Client:  client,
		BaseURL: baseURL,
		Up:      true,
	}
}

func (c *DefaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	baseURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	req.URL = baseURL.ResolveReference(req.URL)
	req.RequestURI = ""

	return c.Client.Do(req)
}

func (c *DefaultHTTPClient) IsUp() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Up
}

func (c *DefaultHTTPClient) SetUp(isUp bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Up = isUp
}

func (c *DefaultHTTPClient) GetBaseURL() string {
	return c.BaseURL
}
