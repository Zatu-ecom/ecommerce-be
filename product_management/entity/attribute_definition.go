package entity

import (
	"database/sql/driver"
	"ecommerce-be/common/entity"
	"fmt"
	"strings"
)

type AttributeDefinition struct {
	entity.BaseEntity
	Key           string      `json:"key" binding:"required" gorm:"column:key;uniqueIndex"`
	Name          string      `json:"name" binding:"required" gorm:"column:name"`
	DataType      string      `json:"dataType" binding:"required" gorm:"column:data_type"` // string, number, boolean, array
	Unit          string      `json:"unit" gorm:"column:unit"`
	Description   string      `json:"description" gorm:"column:description"`
	AllowedValues StringArray `json:"allowedValues" gorm:"column:allowed_values;type:text[]"`

	// Relationships - use pointers to avoid N+1 queries
	CategoryAttributes []CategoryAttribute `json:"categoryAttributes,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	ProductAttributes  []ProductAttribute  `json:"productAttributes,omitempty" gorm:"foreignKey:attribute_definition_id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

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
