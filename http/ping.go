package http

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"
)

var urlPingHttpClient *http.Client

func init() {
	if urlPingHttpClient == nil {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
		}
		urlPingHttpClient = &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		}
	}
}

type URLWaiter struct {
	Method             string
	Url                string
	Headers            map[string]string
	Payload            map[string]any
	DelayBetweenChecks time.Duration

	// Func versions of above so we can do something dynamcially on each iteration
	RequestFunc  func(iter int, prevError error) (*http.Request, error)
	ValidateFunc func(req *http.Request, resp *http.Response) error
}

func (u *URLWaiter) Run() (success bool, iter int, err error) {
	var req *http.Request
	var resp any
	if u.Method == "" {
		u.Method = "GET"
	}
	if u.DelayBetweenChecks <= 0 {
		u.DelayBetweenChecks = time.Second * 10
	}
	for iter = 0; ; iter += 1 {
		if u.RequestFunc != nil {
			req, err = u.RequestFunc(iter, err)
			if req == nil {
				log.Println("Could not create any requests, returning")
				return false, iter, err
			}
		} else {
			req, err = NewJsonRequest(u.Method, u.Url, u.Payload)
			if err != nil && u.Headers != nil {
				for k, v := range u.Headers {
					req.Header.Set(k, v)
				}
			}
		}
		if err != nil {
			log.Println("error creating request: ", err)
			return false, iter, err
		}
		resp, err = Call(req, urlPingHttpClient)
		if err != nil {
			log.Println("Error calling URl: ", u.Url, err)
		} else {
			log.Println("Success calling URL: ", u.Url, resp)
			return
		}
		time.Sleep(u.DelayBetweenChecks)
	}
}
