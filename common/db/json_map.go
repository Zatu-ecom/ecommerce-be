package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap represents a JSONB object column and supports database scan/value operations.
type JSONMap map[string]any

// Scan implements sql.Scanner.
func (jm *JSONMap) Scan(value any) error {
	if value == nil {
		*jm = make(JSONMap)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			*jm = make(JSONMap)
			return nil
		}
		return json.Unmarshal(v, jm)
	case string:
		if v == "" {
			*jm = make(JSONMap)
			return nil
		}
		return json.Unmarshal([]byte(v), jm)
	default:
		return fmt.Errorf("db.JSONMap: unsupported Scan type %T", value)
	}
}

// Value implements driver.Valuer.
func (jm JSONMap) Value() (driver.Value, error) {
	if jm == nil {
		return json.Marshal(make(map[string]any))
	}
	return json.Marshal(jm)
}
