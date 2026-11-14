package netutil

import (
	"io"
	"net/http"
	"time"
)

const maxDrainBytes = 64 * 1024 // 最多排空 64KB 残余数据，兼顾连接复用与防恶意大流阻塞

// DrainAndCloseBody 安全排空并关闭 HTTP 响应体，杜绝 Socket FD 泄漏与 TIME_WAIT 堆积
func DrainAndCloseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrainBytes))
	_ = resp.Body.Close()
}

// NewBoundedHTTPClient 创建带严格连接池上限与空闲超时的 HTTP 客户端，严防后台探针撑爆内存
func NewBoundedHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	transport := &http.Transport{
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   4,
		MaxConnsPerHost:       8,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   4 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
