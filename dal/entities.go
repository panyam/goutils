package dal

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/panyam/goutils/utils"
)

type JsonField map[string]interface{}

func (j *JsonField) Map() map[string]interface{} {
	return *j
}

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
