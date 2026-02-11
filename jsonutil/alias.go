package jsonutil

import (
	"io"

	ij "github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
)

type RawMessage = ij.RawMessage
type Encoder = ij.Encoder
type Decoder = ij.Decoder

func Marshal(v any) ([]byte, error) { return ij.Marshal(v) }
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return ij.MarshalIndent(v, prefix, indent)
}
func Unmarshal(data []byte, v any) error  { return ij.Unmarshal(data, v) }
func NewEncoder(w io.Writer) *Encoder     { return ij.NewEncoder(w) }
func NewDecoder(r io.Reader) *Decoder     { return ij.NewDecoder(r) }
