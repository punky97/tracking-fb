package utils

import (
	"bytes"
	"github.com/punky97/go-codebase/core/logger"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HttpHelper struct {
	Headers map[string]string
}

var _ignoreHeader = []string{
	"cookie",
	"authorization",
}

func (h *HttpHelper) SetHeader(headers map[string]string, request *http.Request) *http.Request {
	for key, value := range headers {
		request.Header.Add(key, value)
	}

	return request
}

// SendRequest --
func (httpHelper *HttpHelper) SendRequest(method string, url string, params map[string]interface{}) (resp *http.Response, err error) {
	paramsFormatted, _ := json.Marshal(params)
	request, err := http.NewRequest(method, url, bytes.NewBuffer(paramsFormatted))

	if err != nil {
		logger.BkLog.Error("Error when get send request info from seoshop, detail: ", err)
		return
	}
	request = httpHelper.SetHeader(httpHelper.Headers, request)

	// Create client
	client := &http.Client{}

	// Fetch Request
	resp, err = client.Do(request)
	if err != nil {
		fmt.Println(err)
		return
	}

	return
}

func Execute(method string, postUrl string, params map[string]interface{}, headers map[string]string) (resultByte []byte, err error, raw *http.Response) {
	if nil == params {
		params = map[string]interface{}{}
	}

	if nil == headers {
		headers = map[string]string{}
	}

	contentType := ""
	bodyContent := ""
	client := &http.Client{
		Timeout: 30 * time.Second, // Set request time out
	}

	postForm := url.Values{}
	acceptedContentTypes := []string{
		"application/x-www-form-urlencoded",
		"application/json",
	}

	// set default content type
	contentType, ok := headers["Content-Type"]
	if !ok {
		contentType = "application/x-www-form-urlencoded"
	}

	if method != http.MethodGet {
		headers["Content-Type"] = "application/json"
	}
	// check content type is allowed
	if !IsStringSliceContains(acceptedContentTypes, contentType) {
		err = fmt.Errorf("Content type is not allowed, %v", contentType)
		return
	}

	// encode json body if content type is json
	if len(params) != 0 {
		if contentType == "application/json" {
			if bodyBytes, err := json.Marshal(params); err == nil {
				bodyContent = string(bodyBytes)
			}
		} else {
			for key, value := range params {
				postForm.Add(key, fmt.Sprint(value))
			}
			bodyContent = postForm.Encode()
		}
	}

	logger.BkLog.Info("Method: ", method, " url: ", postUrl, " params: ", bodyContent, " Headers: ", headers)

	var req *http.Request
	if len(bodyContent) > 0 {
		if method == http.MethodGet {
			req, err = http.NewRequest(method, fmt.Sprintf("%s?%s", postUrl, bodyContent), nil)
		} else {
			req, err = http.NewRequest(method, postUrl, strings.NewReader(bodyContent))
		}
	} else {
		req, err = http.NewRequest(method, postUrl, nil)
	}

	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when create request: %v", err), "url", postUrl)
		return
	}

	req.Header.Set("Content-Type", contentType)
	if len(headers) != 0 {
		for key, value := range headers {
			req.Header.Add(key, fmt.Sprint(value))
		}
	}

	raw, err = client.Do(req)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when send Get request from api, details: %v", err), "url", postUrl, "header", raw)
		return
	}

	resultByte, err = ioutil.ReadAll(raw.Body)
	defer raw.Body.Close()

	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when read response body: %v", err), "header", raw)
		return
	}

	return
}

func IsHttpErrorCode(code int) bool {
	if code >= 400 && code <= 599 {
		return true
	}

	return false
}

func DumpHeaderMap(request *http.Request, ignoreHeaders ...string) map[string][]string {
	h := make(map[string][]string)
	for k, v := range request.Header {
		if !IsStringSliceCaseInsensitiveContains(_ignoreHeader, k) && !IsStringSliceCaseInsensitiveContains(ignoreHeaders, k) {
			h[k] = v
		}
	}

	return h
}

func GetOriginClientIP(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if realIP := req.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	} else if realIP := req.Header.Get("X-Forwarded-For"); realIP != "" {
		remoteAddr = realIP
	}

	return remoteAddr
}

//
const (
	DefaultConnectTimeout        int64 = 5
	DefaultRequestTimeout        int64 = 15
	DefaultHandshakeTimeout      int64 = 5
	DefaultKeepalive             int64 = 100
	DefaultExpectContinueTimeout int64 = 1
	DefaultMaxIdleConnPerHost    int   = 500 // TODO: auto tuning this params to avoid keeping too many idle conn
)

// NewDefaultClient creates new clients
func NewDefaultClient() *NetClient {
	o := &NetClient{}
	return o.Build()
}

// NetClient --
type NetClient struct {
	*http.Client
	ConnectTimeout        int64 `json:"connect_timeout"`
	RequestTimeout        int64 `json:"request_timeout"`
	HandshakeTimeout      int64 `json:"handshake_timeout"`
	KeepAlive             int64 `json:"keep_alive"`
	EnableInsecureMode    bool  `json:"enable_insecure"`
	ExpectContinueTimeout int64 `json:"expect_continue_timeout,omitempty"`
	MaxIdleConnPerHost    int   `json:"max_idle_conn_per_host,omitempty"`
}

// WithConnectTimeout --
func (o *NetClient) WithConnectTimeout(t int64) *NetClient {
	o.ConnectTimeout = t
	return o
}

// WithRequestTimeout --
func (o *NetClient) WithRequestTimeout(t int64) *NetClient {
	o.RequestTimeout = t
	return o
}

// WithHandshakeTimeout --
func (o *NetClient) WithHandshakeTimeout(t int64) *NetClient {
	o.HandshakeTimeout = t
	return o
}

// WithKeepalive --
func (o *NetClient) WithKeepalive(t int64) *NetClient {
	o.KeepAlive = t
	return o
}

// WithInsecureMode --
func (o *NetClient) WithInsecureMode(b bool) *NetClient {
	o.EnableInsecureMode = b
	return o
}

// Build --
func (o *NetClient) Build() *NetClient {
	o.build()

	config := &tls.Config{
		InsecureSkipVerify: o.EnableInsecureMode,
	}
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   time.Duration(o.ConnectTimeout) * time.Second,
			KeepAlive: time.Duration(o.KeepAlive) * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   time.Duration(o.HandshakeTimeout) * time.Second,
		TLSClientConfig:       config,
		ExpectContinueTimeout: time.Duration(o.ExpectContinueTimeout) * time.Second,
		IdleConnTimeout:       time.Duration(o.KeepAlive) * time.Second,
		MaxIdleConnsPerHost:   o.MaxIdleConnPerHost,
	}
	o.Client = &http.Client{
		Timeout:   time.Duration(o.RequestTimeout) * time.Second,
		Transport: netTransport,
	}

	return o
}

func (o *NetClient) build() {
	if o.ConnectTimeout == 0 {
		o.ConnectTimeout = DefaultConnectTimeout
	}
	if o.RequestTimeout == 0 {
		o.RequestTimeout = DefaultRequestTimeout
	}
	if o.HandshakeTimeout == 0 {
		o.HandshakeTimeout = DefaultHandshakeTimeout
	}
	if o.KeepAlive == 0 {
		o.KeepAlive = DefaultKeepalive
	}
	if o.ExpectContinueTimeout == 0 {
		o.ExpectContinueTimeout = DefaultExpectContinueTimeout
	}
	if o.MaxIdleConnPerHost == 0 {
		o.MaxIdleConnPerHost = DefaultMaxIdleConnPerHost
	}
}
