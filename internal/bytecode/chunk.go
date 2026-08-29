package bytecode

import "glox/internal/value"

type Chunk struct {
	Code      []byte
	Constants []value.Value
}
