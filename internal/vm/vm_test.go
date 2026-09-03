package vm

import (
	"bytes"
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

// TestStackOps covers every Phase 2 stack-shuffle opcode; push lists the
// stack before the op (bottom-first), want the stack after it.
func TestStackOps(t *testing.T) {
	tests := []struct {
		name string
		push []value.Value // pushed bottom-first before the op runs
		op   opcode.Opcode
		want []value.Value // expected stack after the op (bottom-first)
	}{
		{"pop drops the top", []value.Value{7, 8}, opcode.Pop, []value.Value{7}},
		{"dup copies the top", []value.Value{5}, opcode.Dup, []value.Value{5, 5}},
		{"swap exchanges two", []value.Value{3, 4}, opcode.Swap, []value.Value{4, 3}},
		{"over copies the second", []value.Value{3, 4}, opcode.Over, []value.Value{3, 4, 3}},
		{"rot rotates three", []value.Value{1, 2, 3}, opcode.Rot, []value.Value{2, 3, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var code []byte
			for i := range tt.push {
				code = append(code, constant(uint16(i))...)
			}
			code = append(code, op(tt.op)...)
			code = append(code, op(opcode.Halt)...)
			machine := New(bytecode.Chunk{Code: code, Constants: tt.push})
			if err := machine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			assertStack(t, machine.Stack(), tt.want)
		})
	}
}

// TestStackOpUnderflow checks each shuffle fails cleanly when the stack is
// one value short of what it needs.
func TestStackOpUnderflow(t *testing.T) {
	tests := []struct {
		name  string
		op    opcode.Opcode
		stack []value.Value // fewer values than the op needs
	}{
		{"pop on empty", opcode.Pop, nil},
		{"dup on empty", opcode.Dup, nil},
		{"swap on one", opcode.Swap, []value.Value{1}},
		{"over on one", opcode.Over, []value.Value{1}},
		{"rot on two", opcode.Rot, []value.Value{1, 2}},
		{"neg on empty", opcode.Neg, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var code []byte
			for i := range tt.stack {
				code = append(code, constant(uint16(i))...)
			}
			code = append(code, op(tt.op)...)
			code = append(code, op(opcode.Halt)...)
			machine := New(bytecode.Chunk{Code: code, Constants: tt.stack})
			if err := machine.Run(); !errors.Is(err, vmerrors.ErrStackUnderflow) {
				t.Fatalf("Run() error = %v, want ErrStackUnderflow", err)
			}
		})
	}
}

// TestPrint checks output text and that Print pops each value it prints.
func TestPrint(t *testing.T) {
	tests := []struct {
		name      string
		constants []value.Value
		code      []byte
		want      string
	}{
		{
			name:      "single value",
			constants: []value.Value{42},
			code:      prog(constant(0), op(opcode.Print), op(opcode.Halt)),
			want:      "42\n",
		},
		{
			name:      "negative value",
			constants: []value.Value{-7},
			code:      prog(constant(0), op(opcode.Print), op(opcode.Halt)),
			want:      "-7\n",
		},
		{
			name:      "sequence pops each value",
			constants: []value.Value{1, 2, 3},
			code: prog(
				constant(0), op(opcode.Print),
				constant(1), op(opcode.Print),
				constant(2), op(opcode.Print),
				op(opcode.Halt),
			),
			want: "1\n2\n3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			machine := New(bytecode.Chunk{Code: tt.code, Constants: tt.constants})
			machine.SetOutput(&buf)
			if err := machine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
			if stack := machine.Stack(); len(stack) != 0 {
				t.Errorf("stack after prints = %v, want empty", stack)
			}
		})
	}
}

// TestStackOverflow pushes one more constant than the depth limit allows;
// the (maxStack+1)th push must fail with ErrStackOverflow, leaving exactly
// maxStack values on the stack.
func TestStackOverflow(t *testing.T) {
	code := make([]byte, 0, (maxStack+1)*3+1)
	for i := 0; i <= maxStack; i++ {
		code = append(code, constant(0)...)
	}
	code = append(code, op(opcode.Halt)...)

	machine := New(bytecode.Chunk{Code: code, Constants: []value.Value{1}})
	if err := machine.Run(); !errors.Is(err, vmerrors.ErrStackOverflow) {
		t.Fatalf("Run() error = %v, want ErrStackOverflow", err)
	}
	if got := len(machine.Stack()); got != maxStack {
		t.Errorf("stack depth = %d, want %d", got, maxStack)
	}
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
