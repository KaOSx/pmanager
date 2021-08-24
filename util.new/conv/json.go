package conv

import (
	"bytes"
	"encoding/json"
	"io"
)

func ReadJson(r io.Reader, v interface{}) error {
	d := json.NewDecoder(r)
	return d.Decode(v)
}

func WriteJson(w io.Writer, v interface{}, beautify bool) error {
	e := json.NewEncoder(w)
	if beautify {
		e.SetIndent("", "    ")
	}
	return e.Encode(v)
}

func ToJson(v interface{}, beautify bool) []byte {
	var w bytes.Buffer
	WriteJson(&w, v, beautify)
	return w.Bytes()
}

func ToMap(v interface{}) Map {
	m := make(Map)
	b, _ := json.Marshal(v)
	json.Unmarshal(b, &m)
	return m
}

func ToData(src, dest interface{}) (err error) {
	var buf bytes.Buffer
	if err = WriteJson(&buf, src, false); err == nil {
		err = ReadJson(&buf, dest)
	}
	return
}
