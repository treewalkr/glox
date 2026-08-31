// Package errors defines the sentinel errors returned by the VM and its
// helpers. All VM failures are reported as errors (never panics), so callers
// can distinguish failure modes with errors.Is.
package errors

import "errors"

// Stack errors.
var (
	// ErrStackUnderflow is returned when an instruction pops more values
	// than the stack holds.
	ErrStackUnderflow = errors.New("stack underflow")

	// ErrStackOverflow is returned when the value stack exceeds its
	// configured limit.
	ErrStackOverflow = errors.New("stack overflow")
)

// Arithmetic errors.
var (
	// ErrDivideByZero is returned by Div and Mod when the divisor is zero.
	ErrDivideByZero = errors.New("integer divide by zero")
)

// Code / decode errors.
var (
	// ErrUnknownOpcode is returned when the fetch-decode loop reads an
	// opcode byte that has no corresponding instruction.
	ErrUnknownOpcode = errors.New("unknown opcode")

	// ErrTruncatedOperand is returned when an instruction's operand bytes
	// run past the end of the code array (e.g. a Constant whose u16 index
	// is cut off mid-read).
	ErrTruncatedOperand = errors.New("truncated instruction operand")

	// ErrConstantIndex is returned when a Constant operand indexes outside
	// the constant pool.
	ErrConstantIndex = errors.New("constant index out of range")
)

// Call-stack errors (Phase 5).
var (
	// ErrCallStackOverflow is returned when Call nests deeper than the
	// call-frame limit.
	ErrCallStackOverflow = errors.New("call stack overflow")

	// ErrNoCallFrame is returned by Return when the call stack is empty.
	ErrNoCallFrame = errors.New("return with no call frame")
)
