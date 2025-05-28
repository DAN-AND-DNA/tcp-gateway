package encoding

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"sync"

	"errors"
	//"github.com/pkg/errors"
)

// 目前仅支持以下原生类型
// bool
// byte
// int16
// uint16
// int32
// uint32
// float32
// float64
// string
// []byte
// 支持他们的数组（注意：这里的数组指的是 Slice 而不是 Array）
// 扩展类型必须是由原生类型组成的结构体
// 只有结构体或者数组支持指针，原生类型不支持指针，也不支持结构体或者数组的指针的指针
// 结构体即使是空也要把每个字段填入 0，数组即使是空至少也会写入一个长度 0
// 结构体中出现循环引用会报错

type encodeState struct {
	bytes.Buffer // accumulated output
	scratch      [8]byte

	// Keep track of what pointers we've seen in the current recursive call
	// path, to avoid cycles that could lead to a stack overflow. Only do
	// the relatively expensive map operations if ptrLevel is larger than
	// startDetectingCyclesAfter, so that we skip the work if we're within a
	// reasonable amount of nested pointers deep.
	ptrLevel uint
	ptrSeen  map[interface{}]struct{}
}

const startDetectingCyclesAfter = 100

var encodeStatePool sync.Pool

func newEncodeState() *encodeState {
	if v := encodeStatePool.Get(); v != nil {
		e := v.(*encodeState)
		e.Reset()
		if len(e.ptrSeen) > 0 {
			panic("ptrEncoder.encode should have emptied ptrSeen via defers")
		}
		e.ptrLevel = 0
		return e
	}
	return &encodeState{ptrSeen: make(map[interface{}]struct{})}
}

func (e *encodeState) encode(v interface{}) error {
	switch vv := v.(type) {
	case bool:
		if vv {
			_ = e.WriteByte(1)
		} else {
			_ = e.WriteByte(0)
		}
		return nil
	case byte:
		_ = e.WriteByte(vv)
		return nil
	case int16:
		binary.LittleEndian.PutUint16(e.scratch[:2], uint16(vv))
		_, _ = e.Write(e.scratch[:2])
		return nil
	case uint16:
		binary.LittleEndian.PutUint16(e.scratch[:2], vv)
		_, _ = e.Write(e.scratch[:2])
		return nil
	case int32:
		binary.LittleEndian.PutUint32(e.scratch[:4], uint32(vv))
		_, _ = e.Write(e.scratch[:4])
		return nil
	case uint32:
		binary.LittleEndian.PutUint32(e.scratch[:4], vv)
		_, _ = e.Write(e.scratch[:4])
		return nil
	case float32:
		binary.LittleEndian.PutUint32(e.scratch[:4], math.Float32bits(vv))
		_, _ = e.Write(e.scratch[:4])
		return nil
	case float64:
		binary.LittleEndian.PutUint64(e.scratch[:8], math.Float64bits(vv))
		_, _ = e.Write(e.scratch[:8])
		return nil
	case string:
		index := strings.IndexByte(vv, 0)
		if index >= 0 {
			return fmt.Errorf("string index %d bytes is \\0", index)
		}
		_, _ = e.Write([]byte(vv))
		_ = e.WriteByte(0)
		return nil
	case []byte:
		if len(vv) > math.MaxUint16 {
			return fmt.Errorf("byte array too long: %d bytes", len(vv))
		}
		binary.LittleEndian.PutUint16(e.scratch[:2], uint16(len(vv)))
		_, _ = e.Write(e.scratch[:2])
		_, _ = e.Write(vv)
		return nil
	}
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Ptr {
		if vv.IsNil() {
			return errors.New("encode nil")
		}
		e.ptrLevel++
		defer func() {
			e.ptrLevel--
		}()
		if e.ptrLevel > startDetectingCyclesAfter {
			// We're a large number of nested ptrEncoder.encode calls deep;
			// start checking if we've run into a pointer cycle.
			ptr := vv.Pointer()
			if _, ok := e.ptrSeen[ptr]; ok {
				return fmt.Errorf("encode encountered a cycle via %s", vv.Type())
			}
			e.ptrSeen[ptr] = struct{}{}
			defer delete(e.ptrSeen, ptr)
		}

		vv = vv.Elem()
	}
	switch vv.Kind() {
	case reflect.Struct:
		err := e.encodeStruct(vv)
		if err == nil {
			return nil
		}
		return fmt.Errorf("encode type %T error: %v", v, err)
	case reflect.Slice:
		err := e.encodeArray(vv)
		if err == nil {
			return nil
		}
		return fmt.Errorf("encode type %T error: %v", v, err)
	default:
		panic(fmt.Sprintf("Unknown type %#v", v))
	}
}

func (e *encodeState) encodeStruct(v reflect.Value) error {
	l := v.NumField()
	t := v.Type()
	for i := 0; i < l; i++ {
		fieldName := t.Field(i).Name
		c := fieldName[0] // Field name first char
		if 'a' <= c && c <= 'z' {
			continue
		}
		err := e.encode(v.Field(i).Interface())
		if err != nil {
			return fmt.Errorf("encode field %s error: %v", fieldName, err)
		}
	}
	return nil
}

func (e *encodeState) encodeArray(v reflect.Value) error {
	l := v.Len()
	if l > math.MaxUint16 {
		return fmt.Errorf("array too long: %d elements", l)
	}
	binary.LittleEndian.PutUint16(e.scratch[:2], uint16(l))
	_, _ = e.Write(e.scratch[:2])
	for i := 0; i < l; i++ {
		err := e.encode(v.Index(i).Interface())
		if err != nil {
			return fmt.Errorf("encode array[%d] error: %v", i, err)
		}
	}
	return nil
}

func Marshal(v interface{}) ([]byte, error) {
	e := newEncodeState()
	err := e.encode(v)
	if err != nil {
		return nil, fmt.Errorf("encode error: %v", err)
	}
	buf := append([]byte(nil), e.Bytes()...)
	encodeStatePool.Put(e) // 出错的 encodeState 不放回去
	return buf, nil
}

func MarshalToWriter(v interface{}, w io.Writer) (int, error) {
	e := newEncodeState()
	err := e.encode(v)
	if err != nil {
		return 0, fmt.Errorf("encode error: %v", err)
	}
	n, err := w.Write(e.Bytes())
	if err != nil {
		return n, fmt.Errorf("write error: %v", err)
	}
	encodeStatePool.Put(e) // 出错的 encodeState 不放回去
	return n, nil
}
