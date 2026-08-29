package vm

import "glox/internal/value"

type Vm struct {
	stack []value.Value
	pc    int
}
