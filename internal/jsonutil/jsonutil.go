package jsonutil

import (
	"bytes"
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

var JSON = jsoniter.ConfigCompatibleWithStandardLibrary

type RawMessage = jsoniter.RawMessage

var bufferPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func Marshal(v any) ([]byte, error) { return encodeWithIndent(v, "", "") }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return encodeWithIndent(v, prefix, indent)
}
func Unmarshal(data []byte, v any) error       { return JSON.Unmarshal(data, v) }
func NewEncoder(w io.Writer) *jsoniter.Encoder { return JSON.NewEncoder(w) }
func NewDecoder(r io.Reader) *jsoniter.Decoder { return JSON.NewDecoder(r) }

func encodeWithIndent(v any, prefix, indent string) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	enc := JSON.NewEncoder(buf)
	if indent != "" || prefix != "" {
		enc.SetIndent(prefix, indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}
