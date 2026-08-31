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
	// Hand-assembled: 3 + 4
	//   CONSTANT 0   ; push constants[0]=3
	//   CONSTANT 1   ; push constants[1]=4
	//   ADD          ; 3 + 4 = 7
	//   HALT
	chunk := bytecode.Chunk{
		Code: []byte{
			byte(opcode.Constant), 0x00, 0x00, // constants[0]
			byte(opcode.Constant), 0x01, 0x00, // constants[1]
			byte(opcode.Add),
			byte(opcode.Halt),
		},
		Constants: []value.Value{3, 4},
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
