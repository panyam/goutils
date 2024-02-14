package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	ErrCodeInvalidRequest     = 400
	ErrCodeInvalidCredentials = 401
	ErrCodeAuthorized         = 403
	ErrCodeEntityNotFound     = 404
)

var DefaultHttpClient *http.Client
var LowQPSHttpClient *http.Client
var MediumQPSHttpClient *http.Client
var HighQPSHttpClient *http.Client

func init() {
	defaultTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	DefaultHttpClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: defaultTransport,
	}

	lowQPSTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 2,
	}
	LowQPSHttpClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: lowQPSTransport,
	}

	mediumQPSTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}
	MediumQPSHttpClient = &http.Client{
		Timeout:   20 * time.Second,
		Transport: mediumQPSTransport,
	}

	highQPSTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
	}
	HighQPSHttpClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: highQPSTransport,
	}
}

func HTTPErrorCode(err error) int {
	if err != nil {
		switch e := err.(type) {
		case *HTTPError:
			return e.Code
		default:
		}
	}
	return -1
}

type HTTPError struct {
	Code    int
	Message string
}

func (t *HTTPError) Error() string {
	return fmt.Sprintf("Status: %d, Message: %s", t.Code, t.Message)
}

func MakeUrl(host, path string, args string) (url string) {
	path = strings.TrimPrefix(path, "/")
	url = fmt.Sprintf("%s/%s", host, path)
	if args != "" {
		url += "?" + args
	}
	return url
}

func NewJsonRequest(method string, endpoint string, body map[string]any) (req *http.Request, err error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	if body != nil {
		json.MarshalIndent(body, "", "  ")
		// log.Println("BODY: ", string(marshalled))
	}
	return NewBytesRequest(method, endpoint, bodyBytes)
}

func NewBytesRequest(method string, endpoint string, body []byte) (req *http.Request, err error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}
	return NewRequest(method, endpoint, bodyReader)
}

func NewRequest(method string, endpoint string, bodyReader io.Reader) (req *http.Request, err error) {
	url := endpoint // t.MakeUrl(endpoint, "")
	req, err = http.NewRequest(method, url, bodyReader)
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		log.Printf("Request: '%s %s", method, url)
	}
	return
}

func Call(req *http.Request, client *http.Client) (response interface{}, err error) {
	if client == nil {
		client = DefaultHttpClient
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Println("client: error making http request: ", err)
		return nil, err
	}
	endTime := time.Now()
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Printf("Response: %d in %f seconds", resp.StatusCode, (endTime.Sub(startTime)).Seconds())
	if resp.StatusCode != 200 {
		log.Println("Response Message: ", string(respbody))
	}

	if resp.StatusCode >= 400 {
		return nil, &HTTPError{resp.StatusCode, string(respbody)}
	}

	content_type := resp.Header.Get("Content-Type")
	if strings.HasPrefix(content_type, "application/json") {
		err = json.Unmarshal(respbody, &response)
	} else {
		// send as is
		response = respbody
	}
	return response, err
}
