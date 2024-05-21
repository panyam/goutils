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

	gut "github.com/panyam/goutils/utils"
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
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
	}
	LowQPSHttpClient = &http.Client{
		Timeout:   10 * time.Second,
		Transport: lowQPSTransport,
	}

	mediumQPSTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 10,
	}
	MediumQPSHttpClient = &http.Client{
		Timeout:   20 * time.Second,
		Transport: mediumQPSTransport,
	}

	highQPSTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        40,
		MaxIdleConnsPerHost: 20,
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

// Representing HTTP specific errors
type HTTPError struct {
	Code    int
	Message string
}

func (t *HTTPError) Error() string {
	return fmt.Sprintf("Status: %d, Message: %s", t.Code, t.Message)
}

// Creates a URL on a host, path and with optional query parameters
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
	}
	return
}

func Call(req *http.Request, client *http.Client) (response interface{}, err error) {
	if client == nil {
		client = DefaultHttpClient
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("client: error making http request: ", err)
		return nil, err
	}
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

// A simple wrapper for performing JSON Get requests.
// The url is the full url once all query params have been added.
// The onReq callback allows customization of the http requests before it is sent.
func JsonGet(url string, onReq func(req *http.Request)) (interface{}, *http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
		return nil, nil, err
	}
	if onReq != nil {
		onReq(req)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
		return nil, resp, err
	}
	defer resp.Body.Close()
	var result interface{}
	var body []byte
	body, err = io.ReadAll(resp.Body)
	// err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("Error reading body: ", string(body), err)
	}
	result, err = gut.JsonDecodeBytes(body)
	if err != nil {
		log.Println("Error decoding json: ", string(body), err)
	}
	return result, resp, err
}
