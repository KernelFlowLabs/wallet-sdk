package algorand

import (
	"io"

	"github.com/algorand/go-codec/codec"
)

var CodecHandle *codec.MsgpackHandle

var LenientCodecHandle *codec.MsgpackHandle

func init() {
	CodecHandle = new(codec.MsgpackHandle)
	CodecHandle.ErrorIfNoField = true
	CodecHandle.ErrorIfNoArrayExpand = true
	CodecHandle.Canonical = true
	CodecHandle.RecursiveEmptyCheck = true
	CodecHandle.WriteExt = true
	CodecHandle.PositiveIntUnsigned = true

	LenientCodecHandle = new(codec.MsgpackHandle)

	LenientCodecHandle.ErrorIfNoField = false
	LenientCodecHandle.ErrorIfNoArrayExpand = true
	LenientCodecHandle.Canonical = true
	LenientCodecHandle.RecursiveEmptyCheck = true
	LenientCodecHandle.WriteExt = true
	LenientCodecHandle.PositiveIntUnsigned = true
}

func Encode(obj interface{}) []byte {
	var b []byte
	enc := codec.NewEncoderBytes(&b, CodecHandle)
	enc.MustEncode(obj)
	return b
}

func Decode(b []byte, objptr interface{}) error {
	dec := codec.NewDecoderBytes(b, CodecHandle)
	err := dec.Decode(objptr)
	if err != nil {
		return err
	}
	return nil
}

func NewDecoder(r io.Reader) *codec.Decoder {
	return codec.NewDecoder(r, CodecHandle)
}

func NewLenientDecoder(r io.Reader) *codec.Decoder {
	return codec.NewDecoder(r, LenientCodecHandle)
}
