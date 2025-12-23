package db

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type StringArray []string

// Value implements the driver.Valuer interface.
// This method converts the StringArray to a format PostgreSQL can understand for text[].
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	// Format: {"value1","value2","value3"}
	var b strings.Builder
	b.WriteString("{")
	for i, s := range a {
		if i > 0 {
			b.WriteString(",")
		}
		// Escape quotes and backslashes
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		fmt.Fprintf(&b, `"%s"`, escaped)
	}
	b.WriteString("}")
	return b.String(), nil
}

// Scan implements the sql.Scanner interface.
// This method scans a value from the database and converts it to a StringArray.
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan type %T into StringArray", value)
	}

	if str == "{}" {
		*a = []string{}
		return nil
	}

	// Convert Postgres array format into Go slice manually
	str = strings.Trim(str, "{}") // remove braces
	if str == "" {
		*a = []string{}
		return nil
	}

	// Split on commas (naïve, but works for simple values)
	parts := strings.Split(str, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		// Strip quotes if present
		p = strings.Trim(p, `"`)
		result[i] = p
	}
	*a = result
	return nil
}

// Int64Array represents a PostgreSQL bigint[] array type
type Int64Array []int64

// Value implements the driver.Valuer interface for bigint[]
func (a Int64Array) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	var b strings.Builder
	b.WriteString("{")
	for i, n := range a {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%d", n)
	}
	b.WriteString("}")
	return b.String(), nil
}

// Scan implements the sql.Scanner interface for bigint[]
func (a *Int64Array) Scan(value interface{}) error {
	if value == nil {
		*a = []int64{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan type %T into Int64Array", value)
	}

	if str == "{}" || str == "" {
		*a = []int64{}
		return nil
	}

	// Convert Postgres array format into Go slice
	str = strings.Trim(str, "{}") // remove braces
	if str == "" {
		*a = []int64{}
		return nil
	}

	parts := strings.Split(str, ",")
	result := make([]int64, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		var n int64
		_, err := fmt.Sscanf(p, "%d", &n)
		if err != nil {
			return fmt.Errorf("failed to parse int64 from '%s': %w", p, err)
		}
		result[i] = n
	}
	*a = result
	return nil
}
