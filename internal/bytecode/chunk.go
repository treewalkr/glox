package bytecode

import (
	"encoding/binary"
	"errors"

	"glox/internal/value"
)

type Chunk struct {
	Code      []byte
	Constants []value.Value
}

// ErrTruncatedOperand is returned when an operand read runs past the end
// of the code array.
var ErrTruncatedOperand = errors.New("truncated instruction operand")

// ReadU16 reads the little-endian u16 operand at code[offset]. This is the
// only sanctioned way to decode operands, so operand encoding stays in one
// place and the VM never slices Code directly.
func ReadU16(code []byte, offset int) (uint16, error) {
	if offset+2 > len(code) {
		return 0, ErrTruncatedOperand
	}
	return binary.LittleEndian.Uint16(code[offset : offset+2]), nil
}
