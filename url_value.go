package structs

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
)

const (
	toValuesTag = "vkey"
)

func ToUrlValues(v interface{}) (url.Values, error) {
	val := reflect.Indirect(reflect.ValueOf(v))
	if val.Kind() != reflect.Struct {
		return nil, errors.New("value must be struct type")
	}

	return structToValues(val)
}

func subToValues(v reflect.Value) (url.Values, error) {
	switch v.Kind() {
	case reflect.Struct:
		return structToValues(v)
	case reflect.Map:
		return mapToValues(v)
	default:
		return make(url.Values), nil
	}
}

func structToValues(v reflect.Value) (url.Values, error) {
	values := make(url.Values)
	typ := v.Type()

	for i, l := 0, v.NumField(); i < l; i++ {
		fieldInfo := typ.Field(i)
		if fieldInfo.PkgPath != "" {
			continue
		}
		field := v.Field(i)

		key, ok := fieldInfo.Tag.Lookup(toValuesTag)
		if !ok {
			key = fieldInfo.Name
		}

		if key == "-" {
			continue
		}

		switch field.Kind() {
		case reflect.Interface:
			err := insertInterfaceValues(&values, key, field)
			if err != nil {
				return nil, err
			}
		case reflect.Struct, reflect.Map:
			sub, err := subToValues(field)
			if err != nil {
				return nil, err
			}
			for k := range sub {
				values[k] = append(values[k], sub[k]...)
			}
		default:
			values[key] = append(values[key], toStrings(field)...)
		}
	}

	return values, nil
}

func mapToValues(v reflect.Value) (url.Values, error) {
	if v.Type().Key().Kind() != reflect.String {
		return nil, errors.New("map key type must be string")
	}

	values := make(url.Values)
	keys := v.MapKeys()
	for i := range keys {
		mapKey := keys[i]
		key := mapKey.String()
		val := v.MapIndex(mapKey)

		switch val.Kind() {
		case reflect.Interface:
			err := insertInterfaceValues(&values, key, val)
			if err != nil {
				return nil, err
			}
		case reflect.Struct, reflect.Map:
			sub, err := subToValues(val)
			if err != nil {
				return nil, err
			}
			for k := range sub {
				values[k] = append(values[k], sub[k]...)
			}
		default:
			values[key] = append(values[key], toStrings(val)...)
		}
	}

	return values, nil
}

func insertInterfaceValues(dst *url.Values, key string, v reflect.Value) error {
	if str, ok := v.Interface().(fmt.Stringer); ok {
		dst.Add(key, str.String())
		return nil
	}

	v = v.Elem()
	switch v.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Struct, reflect.Map:
		sub, err := subToValues(v)
		if err != nil {
			return err
		}
		for k := range sub {
			(*dst)[k] = append((*dst)[k], sub[k]...)
		}
	default:
		(*dst)[key] = append((*dst)[key], toStrings(v)...)
	}

	return nil
}

func toStrings(field reflect.Value) []string {
	switch field.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Ptr:
		return toStrings(field.Elem())
	case reflect.Array, reflect.Slice:
		var str []string
		for i, l := 0, field.Len(); i < l; i++ {
			str = append(str, toStrings(field.Index(i))...)
		}
		return str
	default:
		return []string{fmt.Sprint(field.Interface())}
	}
}