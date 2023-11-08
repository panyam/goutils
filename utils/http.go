package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	ErrCodeInvalidRequest     = 400
	ErrCodeInvalidCredentials = 401
	ErrCodeAuthorized         = 403
	ErrCodeEntityNotFound     = 404
)

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

type HTTPClient struct {
	Host       string
	LogRequest bool
	LogBody    bool
}

func NewClient(host string, apikey string) *HTTPClient {
	if strings.TrimSpace(host) == "" {
		host = os.Getenv("TYPESENSE_HOST")
		if strings.TrimSpace(host) == "" {
			host = "http://localhost:8108"
		}
	}
	if strings.TrimSpace(apikey) == "" {
		apikey = os.Getenv("TYPESENSE_API_KEY")
		if strings.TrimSpace(apikey) == "" {
			apikey = "test_api_key"
		}
	}
	return &HTTPClient{
		Host:       host,
		LogRequest: true,
		LogBody:    true,
	}
}

func (t *HTTPClient) MakeUrl(endpoint string, args string) (url string) {
	endpoint = strings.TrimPrefix(endpoint, "/")
	url = fmt.Sprintf("%s/%s", t.Host, endpoint)
	if args != "" {
		url += "?" + args
	}
	return url
}

func (t *HTTPClient) MakeJsonRequest(method string, endpoint string, body StringMap) (req *http.Request, err error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	if body != nil {
		marshalled, _ := json.MarshalIndent(body, "", "  ")
		log.Println("BODY: ", string(marshalled))
	}
	return t.MakeBytesRequest(method, endpoint, bodyBytes)
}

func (t *HTTPClient) MakeBytesRequest(method string, endpoint string, body []byte) (req *http.Request, err error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}
	return t.MakeRequest(method, endpoint, bodyReader)
}

func (t *HTTPClient) MakeRequest(method string, endpoint string, bodyReader io.Reader) (req *http.Request, err error) {
	url := t.MakeUrl(endpoint, "")
	req, err = http.NewRequest(method, url, bodyReader)
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		log.Printf("Request: '%s %s", method, url)
	}
	return
}

func (t *HTTPClient) JsonCall(req *http.Request) (response StringMap, err error) {
	out, err := t.Call(req)
	if err != nil {
		return nil, err
	}
	return out.(StringMap), err
}

func (t *HTTPClient) Call(req *http.Request) (response interface{}, err error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
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
