package utils

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

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

func JsonReadFile(path string) (interface{}, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
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

func JsonWriteFile(data interface{}, path string, perm os.FileMode) error {
	encdata, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, encdata, perm)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
