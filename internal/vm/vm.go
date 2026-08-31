package vm

import (
	"fmt"

	"glox/internal/bytecode"
	"glox/internal/errors"
	"glox/internal/opcode"
	"glox/internal/value"
)

// Vm is the stack machine interpreter. pc is the program counter: an index
// into chunk.Code pointing at the next instruction byte to fetch.
type Vm struct {
	chunk bytecode.Chunk
	stack []value.Value
	pc    int
}

func New(chunk bytecode.Chunk) *Vm {
	return &Vm{chunk: chunk}
}

// Run executes the fetch-decode-execute loop until Halt or error.
func (v *Vm) Run() error {
	for {
		if v.pc >= len(v.chunk.Code) {
			return errors.ErrTruncatedOperand
		}
		op := opcode.Opcode(v.chunk.Code[v.pc])
		v.pc++ // advance past the opcode byte; operand reads continue from pc

		switch op {
		case opcode.Halt:
			return nil

		case opcode.Constant:
			idx, err := bytecode.ReadU16(v.chunk.Code, v.pc)
			if err != nil {
				return err
			}
			v.pc += 2
			if int(idx) >= len(v.chunk.Constants) {
				return errors.ErrConstantIndex
			}
			v.push(v.chunk.Constants[idx])

		case opcode.Add: // ( a b -- a+b )
			b, err := v.pop()
			if err != nil {
				return err
			}
			a, err := v.pop()
			if err != nil {
				return err
			}
			v.push(a + b)

		default:
			return fmt.Errorf("%w: 0x%02x at pc %d", errors.ErrUnknownOpcode, byte(op), v.pc-1)
		}
	}
}

// Stack returns a copy of the current value stack (for tests/inspection).
func (v *Vm) Stack() []value.Value {
	out := make([]value.Value, len(v.stack))
	copy(out, v.stack)
	return out
}

func (v *Vm) push(val value.Value) {
	v.stack = append(v.stack, val)
}

func (v *Vm) pop() (value.Value, error) {
	if len(v.stack) == 0 {
		return 0, errors.ErrStackUnderflow
	}
	top := v.stack[len(v.stack)-1]
	v.stack = v.stack[:len(v.stack)-1]
	return top, nil
}
