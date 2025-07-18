package utils

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

func JsonDecodeBytes(bytes []byte) (any, error) {
	var output any = nil
	err := json.Unmarshal(bytes, &output)
	if err != nil {
		log.Fatal(err)
	}
	return output, err
}

func JsonReadFile(path string) (any, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	var json_data map[string]any
	err = decoder.Decode(&json_data)
	if err != nil {
		log.Fatal("Invalid error decoding json: ", err)
	}
	return json_data, err
}

func JsonWriteFile(data any, path string, perm os.FileMode) error {
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

func ToJson(v any) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "invalid", err
	}
	return string(jsonBytes), nil
}
