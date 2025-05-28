package encoding

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"

	"errors"
	//"github.com/pkg/errors"
)

type decodeState struct {
	data []byte
	off  int
}

func (d *decodeState) init(data []byte) *decodeState {
	d.data = data
	d.off = 0
	return d
}

func (d *decodeState) decode(v interface{}) error {
	switch vv := v.(type) {
	case *bool:
		if d.off >= len(d.data) {
			return errors.New("read bool EOF")
		}
		*vv = d.data[d.off] != 0
		d.off += 1
		return nil
	case *byte:
		if d.off >= len(d.data) {
			return errors.New("read byte EOF")
		}
		*vv = d.data[d.off]
		d.off += 1
		return nil
	case *int16:
		if d.off+2 > len(d.data) {
			return errors.New("read int16 EOF")
		}
		*vv = int16(binary.LittleEndian.Uint16(d.data[d.off : d.off+2]))
		d.off += 2
		return nil
	case *uint16:
		if d.off+2 > len(d.data) {
			return errors.New("read uint16 EOF")
		}
		*vv = binary.LittleEndian.Uint16(d.data[d.off : d.off+2])
		d.off += 2
		return nil
	case *int32:
		if d.off+4 > len(d.data) {
			return errors.New("read int32 EOF")
		}
		*vv = int32(binary.LittleEndian.Uint32(d.data[d.off : d.off+4]))
		d.off += 4
		return nil
	case *uint32:
		if d.off+4 > len(d.data) {
			return errors.New("read uint32 EOF")
		}
		*vv = binary.LittleEndian.Uint32(d.data[d.off : d.off+4])
		d.off += 4
		return nil
	case *float32:
		if d.off+4 > len(d.data) {
			return errors.New("read float32 EOF")
		}
		*vv = math.Float32frombits(binary.LittleEndian.Uint32(d.data[d.off : d.off+4]))
		d.off += 4
		return nil
	case *float64:
		if d.off+8 > len(d.data) {
			return errors.New("read float64 EOF")
		}
		*vv = math.Float64frombits(binary.LittleEndian.Uint64(d.data[d.off : d.off+8]))
		d.off += 8
		return nil
	case *string:
		if d.off >= len(d.data) {
			return errors.New("read string EOF")
		}
		l := bytes.IndexByte(d.data[d.off:], 0)
		if l < 0 {
			return errors.New("read string no \\0")
		}
		*vv = string(d.data[d.off : d.off+l])
		d.off += l + 1
		return nil
	case *[]byte:
		if d.off+2 > len(d.data) {
			return errors.New("read byte array len EOF")
		}
		l := binary.LittleEndian.Uint16(d.data[d.off : d.off+2])
		d.off += 2
		ll := int(l)
		if d.off+ll > len(d.data) {
			return errors.New("read byte array EOF")
		}
		*vv = make([]byte, ll)
		copy(*vv, d.data[d.off:d.off+ll])
		d.off += ll
		return nil
	}
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}
	if vv.Kind() == reflect.Ptr {
		if vv.IsNil() {
			vv.Set(reflect.New(vv.Type().Elem()))
		}
		vv = vv.Elem()
	}
	switch vv.Kind() {
	case reflect.Struct:
		return d.decodeStruct(vv)
	case reflect.Slice:
		return d.decodeArray(vv)
	default:
		panic(fmt.Sprintf("Unknown type %#v", v))
	}
}

func (d *decodeState) decodeStruct(v reflect.Value) error {
	l := v.NumField()
	t := v.Type()
	for i := 0; i < l; i++ {
		fieldName := t.Field(i).Name
		c := fieldName[0] // Field name first char
		if 'a' <= c && c <= 'z' {
			continue
		}
		err := d.decode(v.Field(i).Addr().Interface())
		if err != nil {
			return fmt.Errorf("decode field %s error: %v", fieldName, err)
			//return errors.WithMessagef(err, "decode field %s error", fieldName)
		}
	}
	return nil
}

func (d *decodeState) decodeArray(v reflect.Value) error {
	if d.off+2 > len(d.data) {
		return errors.New("read array len EOF")
	}
	l := binary.LittleEndian.Uint16(d.data[d.off : d.off+2])
	d.off += 2
	ll := int(l)
	t := v.Type()
	vv := reflect.New(t.Elem())
	v.SetLen(0)
	for i := 0; i < ll; i++ {
		err := d.decode(vv.Interface())
		if err != nil {
			return fmt.Errorf("decode array[%d] error: %v", i, err)
			//return errors.WithMessagef(err, "decode array[%d] error", i)
		}
		vNew := reflect.Append(v, vv.Elem())
		v.Set(vNew)
	}
	return nil
}

func Unmarshal(p []byte, v interface{}) (int, error) {
	d := decodeState{}
	d.init(p)
	err := d.decode(v)
	return d.off, err
}
