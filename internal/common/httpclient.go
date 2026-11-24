package common

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var defaultTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var insecureTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
}

// NewDefaultClient returns an HTTP client with standard secure TLS and pooled connections.
func NewDefaultClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
	}
}

// NewInsecureClient returns an HTTP client with skipped TLS verification (e.g. for self-signed ROS/local router APIs).
func NewInsecureClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: insecureTransport,
		Timeout:   timeout,
	}
}

// NewWithCookieJar returns an HTTP client with an isolated CookieJar (e.g. for iKuai Web-RPC session isolation).
func NewWithCookieJar(timeout time.Duration, jar http.CookieJar) *http.Client {
	return &http.Client{
		Transport: defaultTransport,
		Timeout:   timeout,
		Jar:       jar,
	}
}
