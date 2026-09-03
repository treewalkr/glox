package vm

import (
	"errors"
	"slices"
	"testing"

	"glox/internal/bytecode"
	vmerrors "glox/internal/errors"
	"glox/internal/opcode"
	"glox/internal/value"
)

// --- tiny test assembler ----------------------------------------------------
// Tests assemble programs from fragments instead of hand-writing hex, so
// each test reads like the assembly listing it produces.

// op assembles a single opcode byte.
func op(o opcode.Opcode) []byte {
	return []byte{byte(o)}
}

// constant assembles `CONSTANT idx` with its little-endian u16 operand.
func constant(idx uint16) []byte {
	return []byte{byte(opcode.Constant), byte(idx), byte(idx >> 8)}
}

// prog joins instruction fragments into one code stream.
func prog(parts ...[]byte) []byte {
	var code []byte
	for _, part := range parts {
		code = append(code, part...)
	}
	return code
}

// assertStack fails the test unless the stack equals want, bottom-first.
func assertStack(t *testing.T, got, want []value.Value) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("stack = %v, want %v", got, want)
	}
}

func TestConstantAddHalt(t *testing.T) {
	tests := []struct {
		name      string
		code      []byte
		constants []value.Value
		want      value.Value
	}{
		{
			name:      "3 + 4",
			code:      prog(constant(0), constant(1), op(opcode.Add), op(opcode.Halt)),
			constants: []value.Value{3, 4},
			want:      7,
		},
		{
			name:      "single constant",
			code:      prog(constant(0), op(opcode.Halt)),
			constants: []value.Value{42},
			want:      42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machine := New(bytecode.Chunk{Code: tt.code, Constants: tt.constants})
			if err := machine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			assertStack(t, machine.Stack(), []value.Value{tt.want})
		})
	}
}

// TestArithmetic covers every binary arithmetic opcode from Phase 2,
// including operand order and the truncation/sign rules of Div and Mod.
func TestArithmetic(t *testing.T) {
	tests := []struct {
		name string
		a, b value.Value // pushed in this order: a deep, b on top
		op   opcode.Opcode
		want value.Value
	}{
		{"add", 7, 5, opcode.Add, 12},
		{"add negatives", -7, -5, opcode.Add, -12},
		{"sub", 7, 5, opcode.Sub, 2},
		{"sub to negative", 5, 7, opcode.Sub, -2},
		{"mul", 7, 5, opcode.Mul, 35},
		{"mul by zero", 7, 0, opcode.Mul, 0},
		{"mul negative", -3, 4, opcode.Mul, -12},
		{"div", 7, 5, opcode.Div, 1},
		{"div exact", 10, 5, opcode.Div, 2},
		{"div truncates toward zero", -7, 2, opcode.Div, -3},
		{"div negative divisor", 7, -2, opcode.Div, -3},
		{"mod", 7, 5, opcode.Mod, 2},
		{"mod exact", 10, 5, opcode.Mod, 0},
		{"mod sign follows dividend", -7, 3, opcode.Mod, -1},
		{"mod negative divisor", 7, -3, opcode.Mod, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := prog(constant(0), constant(1), op(tt.op), op(opcode.Halt))
			machine := New(bytecode.Chunk{Code: code, Constants: []value.Value{tt.a, tt.b}})
			if err := machine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			assertStack(t, machine.Stack(), []value.Value{tt.want})
		})
	}
}

func TestNeg(t *testing.T) {
	tests := []struct {
		name string
		in   value.Value
		want value.Value
	}{
		{"positive", 5, -5},
		{"negative", -5, 5},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := prog(constant(0), op(opcode.Neg), op(opcode.Halt))
			machine := New(bytecode.Chunk{Code: code, Constants: []value.Value{tt.in}})
			if err := machine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			assertStack(t, machine.Stack(), []value.Value{tt.want})
		})
	}
}

func TestDivideByZero(t *testing.T) {
	tests := []struct {
		name string
		op   opcode.Opcode
	}{
		{"div", opcode.Div},
		{"mod", opcode.Mod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := prog(constant(0), constant(1), op(tt.op), op(opcode.Halt))
			machine := New(bytecode.Chunk{Code: code, Constants: []value.Value{7, 0}})
			if err := machine.Run(); !errors.Is(err, vmerrors.ErrDivideByZero) {
				t.Fatalf("Run() error = %v, want ErrDivideByZero", err)
			}
		})
	}
}

// TestPhase2DefinitionOfDone is the Phase 2 milestone from PLAN.md: a
// program computing (10 - 2) * 3 + 4 / 2 must leave 26 on the stack.
func TestPhase2DefinitionOfDone(t *testing.T) {
	code := prog(
		constant(0), constant(1), op(opcode.Sub), // 10 - 2 -> 8
		constant(2), op(opcode.Mul), // 8 * 3 -> 24
		constant(3), constant(4), op(opcode.Div), // 4 / 2 -> 2
		op(opcode.Add), // 24 + 2 -> 26
		op(opcode.Halt),
	)
	machine := New(bytecode.Chunk{Code: code, Constants: []value.Value{10, 2, 3, 4, 2}})
	if err := machine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertStack(t, machine.Stack(), []value.Value{26})
}

func TestStackUnderflow(t *testing.T) {
	machine := New(bytecode.Chunk{Code: prog(op(opcode.Add), op(opcode.Halt))})
	if err := machine.Run(); !errors.Is(err, vmerrors.ErrStackUnderflow) {
		t.Fatalf("Run() error = %v, want ErrStackUnderflow", err)
	}
}

func TestUnknownOpcode(t *testing.T) {
	machine := New(bytecode.Chunk{Code: []byte{0xFF}})
	if err := machine.Run(); err == nil {
		t.Fatal("Run() = nil, want unknown-opcode error")
	}
}
