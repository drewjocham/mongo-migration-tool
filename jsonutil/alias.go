package jsonutil

import (
	"io"

	ij "github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	jsoniter "github.com/json-iterator/go"
)

var JSON = ij.JSON

type RawMessage = ij.RawMessage

func Marshal(v any) ([]byte, error) { return ij.Marshal(v) }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return ij.MarshalIndent(v, prefix, indent)
}
func Unmarshal(data []byte, v any) error       { return ij.Unmarshal(data, v) }
func NewEncoder(w io.Writer) *jsoniter.Encoder { return ij.NewEncoder(w) }
func NewDecoder(r io.Reader) *jsoniter.Decoder { return ij.NewDecoder(r) }
