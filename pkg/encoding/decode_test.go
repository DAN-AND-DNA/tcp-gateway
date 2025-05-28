package encoding

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"
)

func BenchmarkUnmarshal(b *testing.B) {
	data, err := Marshal(getBenchmarkMarshalData())
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		v := &cmdMatchStart{}
		n, err := Unmarshal(data, v)
		if err != nil {
			panic(err)
		}
		if n != len(data) {
			panic("read wrong bytes")
		}
	}
}

func BenchmarkJsonUnmarshal(b *testing.B) {
	data, err := json.Marshal(getBenchmarkMarshalData())
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		v := &cmdMatchStart{}
		err := json.Unmarshal(data, v)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkGobUnmarshal(b *testing.B) {
	v := getBenchmarkMarshalData()
	data := &bytes.Buffer{}
	e := gob.NewEncoder(data)
	err := e.Encode(v)
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		data2 := bytes.NewBuffer(data.Bytes())
		d := gob.NewDecoder(data2)
		v2 := &cmdMatchStart{}
		err := d.Decode(v2)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkCopy(b *testing.B) {
	buf := make([]byte, 10)

	msg := []byte("dddddd")
	for i := 0; i < b.N; i++ {

		buf = buf[:10]
		copy(buf, msg)
		buf = buf[:0]
	}
}
