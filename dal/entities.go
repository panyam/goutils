package dal

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/panyam/goutils/utils"
)

var InvalidJsonKeyError = errors.New("InvalidJsonKeyError")

type Json struct {
	ValueJson     string
	LastUpdatedAt int64 `gorm:"autoUpdateTime:milli"`
	value         interface{}
}

func NewJson(value interface{}) *Json {
	j, _ := json.Marshal(value)
	return &Json{ValueJson: string(j), value: value}
}

func (jr *Json) HasValue() bool {
	if jr == nil {
		return false
	}
	var err error
	if jr.value == nil && jr.ValueJson != "" {
		jr.value, err = utils.JsonDecodeStr(jr.ValueJson)
	}
	return err == nil && jr.value != nil
}

func (jr *Json) Value() (result interface{}, err error) {
	if jr == nil {
		return nil, nil
	}
	if jr.value == nil && jr.ValueJson != "" {
		jr.value, err = utils.JsonDecodeStr(jr.ValueJson)
	}
	return jr.value, err
}

type JsonField map[string]interface{}

func (j *JsonField) Scan(value interface{}) error {
	// Scan scan value into Jsonb, implements sql.Scanner interface
	if value != nil && value != "null" {
		bytes, ok := value.(string)
		if !ok {
			return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
		}

		res, err := utils.JsonDecodeStr(bytes)
		if err != nil {
			return err
		}
		*j = JsonField(res.(map[string]interface{}))
	}
	return nil
}

// Value return json value, implement driver.Valuer interface
func (j JsonField) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	js, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(js), nil
}
