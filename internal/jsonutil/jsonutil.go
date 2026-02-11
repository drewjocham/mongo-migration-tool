package jsonutil

import (
	"io"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/decoder"
	"github.com/bytedance/sonic/encoder"
)

var json = sonic.ConfigFastest

type RawMessage = []byte

func Marshal(v any) ([]byte, error)                              { return json.Marshal(v) }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) { return json.MarshalIndent(v, prefix, indent) }
func Unmarshal(data []byte, v any) error                         { return json.Unmarshal(data, v) }

// Encoder wraps sonic's streaming encoder with SetIndent support.
type Encoder struct {
	w      io.Writer
	prefix string
	indent string
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) SetIndent(prefix, indent string) {
	e.prefix = prefix
	e.indent = indent
}

func (e *Encoder) Encode(v any) error {
	var data []byte
	var err error
	if e.indent != "" || e.prefix != "" {
		data, err = encoder.EncodeIndented(v, e.prefix, e.indent, 0)
	} else {
		data, err = encoder.Encode(v, 0)
	}
	if err != nil {
		return err
	}
	_, err = e.w.Write(data)
	if err != nil {
		return err
	}
	_, err = e.w.Write([]byte("\n"))
	return err
}

// Decoder wraps sonic's streaming decoder.
type Decoder = decoder.StreamDecoder

func NewDecoder(r io.Reader) *Decoder {
	return decoder.NewStreamDecoder(r)
}
