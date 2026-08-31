package vm

import (
	"errors"
	"testing"

	"glox/internal/bytecode"
	vmerrors "glox/internal/errors"
	"glox/internal/opcode"
	"glox/internal/value"
)

func TestConstantAddHalt(t *testing.T) {
	tests := []struct {
		name      string
		code      []byte
		constants []value.Value
		want      value.Value
	}{
		{
			name: "3 + 4",
			code: []byte{
				byte(opcode.Constant), 0x00, 0x00,
				byte(opcode.Constant), 0x01, 0x00,
				byte(opcode.Add),
				byte(opcode.Halt),
			},
			constants: []value.Value{3, 4},
			want:      7,
		},
		{
			name: "single constant",
			code: []byte{
				byte(opcode.Constant), 0x00, 0x00,
				byte(opcode.Halt),
			},
			constants: []value.Value{42},
			want:      42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			machine := New(bytecode.Chunk{Code: tt.code, Constants: tt.constants})
			if err := machine.Run(); err != nil {
				t.Fatalf("Run() error: %v", err)
			}
			stack := machine.Stack()
			if len(stack) != 1 {
				t.Fatalf("stack length = %d, want 1: %v", len(stack), stack)
			}
			if stack[0] != tt.want {
				t.Errorf("stack top = %d, want %d", stack[0], tt.want)
			}
		})
	}
}

func TestStackUnderflow(t *testing.T) {
	machine := New(bytecode.Chunk{Code: []byte{byte(opcode.Add), byte(opcode.Halt)}})
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
