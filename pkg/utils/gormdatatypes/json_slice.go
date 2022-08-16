package gormdatatypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type JSONSlice []string

// Value return json value, implement driver.Valuer interface
func (m JSONSlice) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := m.MarshalJSON()
	return string(ba), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *JSONSlice) Scan(val interface{}) error {
	if val == nil {
		*m = make(JSONSlice, 0)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := []string{}
	err := json.Unmarshal(ba, &t)
	*m = JSONSlice(t)
	return err
}

// MarshalJSON to output non base64 encoded []byte
func (m JSONSlice) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	t := ([]string)(m)
	return json.Marshal(t)
}

// UnmarshalJSON to deserialize []byte
func (m *JSONSlice) UnmarshalJSON(b []byte) error {
	t := []string{}
	err := json.Unmarshal(b, &t)
	*m = JSONSlice(t)
	return err
}

// GormDataType gorm common data type
func (m JSONSlice) GormDataType() string {
	return "jsonslice"
}

// GormDBDataType gorm db data type
func (JSONSlice) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}
