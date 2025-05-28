package encoding

import (
	"fmt"
	"reflect"
	//"github.com/pkg/errors"
)

func CheckRegisterTypeInner(t reflect.Type) error {
	switch t.Kind() {
	case reflect.Bool:
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Float32:
	case reflect.Float64:
	case reflect.String:
	case reflect.Ptr:
		tt := t.Elem()
		if tt.Kind() != reflect.Struct {
			return fmt.Errorf("a pointer can only point to a struct: %s", t.String())
		}
		err := CheckRegisterTypeInner(tt)
		if err != nil {
			return fmt.Errorf("pointer to struct type error: %v", err)
		}
	case reflect.Slice:
		err := CheckRegisterTypeInner(t.Elem())
		if err != nil {
			return fmt.Errorf("slice element type error: %v", err)
		}
	case reflect.Struct:
		numField := t.NumField()
		for i := 0; i < numField; i++ {
			field := t.Field(i)
			fieldName := field.Name
			c := fieldName[0] // Field name first char
			if 'a' <= c && c <= 'z' {
				continue
			}
			err := CheckRegisterTypeInner(field.Type)
			if err != nil {
				return fmt.Errorf("struct field %s type error: %v", fieldName, err)
			}
		}
	default:
		return fmt.Errorf("cannot use type %s", t.String())
	}
	return nil
}

func CheckRegisterType(t reflect.Type) error {
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("register type must be a pointer: %s", t.String())
	}
	if t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("register type must be a pointer to struct: %s", t.String())
	}
	return CheckRegisterTypeInner(t)
}

func CheckRegisterTypeInterface(v interface{}) error {
	return CheckRegisterType(reflect.TypeOf(v))
}
