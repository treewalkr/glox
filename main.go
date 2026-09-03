package main

import (
	"fmt"
	"os"

	"glox/internal/bytecode"
	"glox/internal/opcode"
	"glox/internal/value"
	"glox/internal/vm"
)

func main() {
	// Hand-assembled: (10 - 2) * 3 + 4 / 2 — the Phase 2 milestone program.
	//   CONSTANT 0   ; push constants[0] = 10
	//   CONSTANT 1   ; push constants[1] = 2
	//   SUB          ; 10 - 2 = 8
	//   CONSTANT 2   ; push constants[2] = 3
	//   MUL          ; 8 * 3 = 24
	//   CONSTANT 3   ; push constants[3] = 4
	//   CONSTANT 4   ; push constants[4] = 2
	//   DIV          ; 4 / 2 = 2
	//   ADD          ; 24 + 2 = 26
	//   HALT
	chunk := bytecode.Chunk{
		Code: []byte{
			byte(opcode.Constant), 0x00, 0x00, // constants[0]
			byte(opcode.Constant), 0x01, 0x00, // constants[1]
			byte(opcode.Sub),
			byte(opcode.Constant), 0x02, 0x00, // constants[2]
			byte(opcode.Mul),
			byte(opcode.Constant), 0x03, 0x00, // constants[3]
			byte(opcode.Constant), 0x04, 0x00, // constants[4]
			byte(opcode.Div),
			byte(opcode.Add),
			byte(opcode.Halt),
		},
		Constants: []value.Value{10, 2, 3, 4, 2},
	}

	machine := vm.New(chunk)
	if err := machine.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "vm error:", err)
		os.Exit(1)
	}

	stack := machine.Stack()
	if len(stack) == 0 {
		fmt.Println("(empty stack)")
		return
	}
	fmt.Println(stack[len(stack)-1])
}
