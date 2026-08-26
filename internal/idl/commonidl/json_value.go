package commonidl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func (value JsonValue) MarshalJSON() ([]byte, error) {
	if value.CountSetFieldsJsonValue() != 1 {
		return nil, fmt.Errorf("JsonValue must contain exactly one value")
	}
	switch {
	case value.NullValue != nil:
		if !*value.NullValue {
			return nil, fmt.Errorf("JsonValue null_value must be true")
		}
		return []byte("null"), nil
	case value.StringValue != nil:
		return json.Marshal(*value.StringValue)
	case value.BoolValue != nil:
		return json.Marshal(*value.BoolValue)
	case value.IntegerValue != nil:
		return json.Marshal(*value.IntegerValue)
	case value.NumberValue != nil:
		return json.Marshal(*value.NumberValue)
	case value.ListValue != nil:
		return json.Marshal(value.ListValue)
	case value.ObjectValue != nil:
		return json.Marshal(value.ObjectValue)
	default:
		return nil, fmt.Errorf("JsonValue must contain exactly one value")
	}
}

func (value *JsonValue) UnmarshalJSON(content []byte) error {
	if value == nil {
		return fmt.Errorf("JsonValue is nil")
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	decoded, err := FromAny(raw)
	if err != nil {
		return err
	}
	*value = *decoded
	return nil
}

func FromJSON(content []byte) (*JsonValue, error) {
	var value JsonValue
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

func FromAny(raw any) (*JsonValue, error) {
	switch typed := raw.(type) {
	case nil:
		value := true
		return &JsonValue{NullValue: &value}, nil
	case string:
		return &JsonValue{StringValue: &typed}, nil
	case bool:
		return &JsonValue{BoolValue: &typed}, nil
	case json.Number:
		if integer, err := strconv.ParseInt(string(typed), 10, 64); err == nil {
			return &JsonValue{IntegerValue: &integer}, nil
		}
		number, err := strconv.ParseFloat(string(typed), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON number: %w", err)
		}
		return &JsonValue{NumberValue: &number}, nil
	case float64:
		return &JsonValue{NumberValue: &typed}, nil
	case []any:
		result := make([]*JsonValue, len(typed))
		for index, item := range typed {
			value, err := FromAny(item)
			if err != nil {
				return nil, fmt.Errorf("JsonValue list item %d: %w", index, err)
			}
			result[index] = value
		}
		return &JsonValue{ListValue: result}, nil
	case map[string]any:
		result := make(map[string]*JsonValue, len(typed))
		for key, item := range typed {
			value, err := FromAny(item)
			if err != nil {
				return nil, fmt.Errorf("JsonValue object key %q: %w", key, err)
			}
			result[key] = value
		}
		return &JsonValue{ObjectValue: result}, nil
	default:
		content, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("JsonValue does not support %T", raw)
		}
		return FromJSON(content)
	}
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("JsonValue contains trailing JSON")
}
