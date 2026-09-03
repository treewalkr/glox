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

		// Arithmetic: binary instructions pop b (the top of the stack)
		// first, then a (the deeper value). In the effect ( a b -- a-b )
		// the left operand a sits *below* the right operand b.

		case opcode.Add: // ( a b -- a+b )
			if err := v.binaryOp(add); err != nil {
				return err
			}

		case opcode.Sub: // ( a b -- a-b )
			if err := v.binaryOp(sub); err != nil {
				return err
			}

		case opcode.Mul: // ( a b -- a*b )
			if err := v.binaryOp(mul); err != nil {
				return err
			}

		case opcode.Div: // ( a b -- a/b ) truncates toward zero
			if err := v.binaryOp(div); err != nil {
				return err
			}

		case opcode.Mod: // ( a b -- a%b ) sign follows the dividend
			if err := v.binaryOp(mod); err != nil {
				return err
			}

		case opcode.Neg: // ( a -- -a )
			a, err := v.pop()
			if err != nil {
				return err
			}
			v.push(-a)

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

// binaryOp runs one binary arithmetic instruction: pop b (the top) then a
// (the deeper value), apply f, push the result. Operand order is the
// classic stack-machine bug source, so it lives in exactly one place.
func (v *Vm) binaryOp(f func(a, b value.Value) (value.Value, error)) error {
	b, err := v.pop()
	if err != nil {
		return err
	}
	a, err := v.pop()
	if err != nil {
		return err
	}
	result, err := f(a, b)
	if err != nil {
		return err
	}
	v.push(result)
	return nil
}

// Arithmetic kernels. Each takes (a, b) where a is the deeper stack value
// (the left operand). They stay separate named functions because Phase 6
// makes them type-aware (int/float promotion).

func add(a, b value.Value) (value.Value, error) { return a + b, nil }

func sub(a, b value.Value) (value.Value, error) { return a - b, nil }

func mul(a, b value.Value) (value.Value, error) { return a * b, nil }

// div implements integer division truncating toward zero (Go's /). A zero
// divisor returns ErrDivideByZero — never a panic.
func div(a, b value.Value) (value.Value, error) {
	if b == 0 {
		return 0, errors.ErrDivideByZero
	}
	return a / b, nil
}

// mod implements the remainder with the sign of the dividend (Go's %). A
// zero divisor returns ErrDivideByZero — never a panic.
func mod(a, b value.Value) (value.Value, error) {
	if b == 0 {
		return 0, errors.ErrDivideByZero
	}
	return a % b, nil
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
