package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func JsonToQueryString(json StringMap) string {
	out := ""
	for key, value := range json {
		if len(out) > 0 {
			out += "&"
		}
		item := fmt.Sprintf("%s=%v", url.PathEscape(key), value.(interface{}))
		out += item
	}
	return out
}

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
	body, err = ioutil.ReadAll(resp.Body)
	// err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("Error reading body: ", string(body), err)
	}
	result, err = JsonDecodeBytes(body)
	if err != nil {
		log.Println("Error decoding json: ", string(body), err)
	}
	return result, resp, err
}

func JsonDecodeBytes(bytes []byte) (interface{}, error) {
	var output interface{} = nil
	err := json.Unmarshal(bytes, &output)
	if err != nil {
		log.Fatal(err)
	}
	return output, err
}

func JsonDecodeStr(str string) (interface{}, error) {
	return JsonDecodeBytes([]byte(str))
}

func JsonDecodeFile(path string) (interface{}, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	var json_data map[string]interface{}
	err = decoder.Decode(&json_data)
	if err != nil {
		log.Fatal("Invalid error decoding json: ", err)
	}
	return json_data, err
}
