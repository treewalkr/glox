// Package opcode defines the instruction set: one Opcode constant per
// instruction byte. Opcode values are pinned by docs/bytecode.md — append
// new opcodes only; never renumber or reuse existing values.
package opcode

// Opcode is a single instruction byte in the bytecode stream.
type Opcode byte

const (
	Halt     Opcode = iota // stop execution
	Constant               // ( -- const[idx] ) push the constant pool entry
	Add                    // ( a b -- a+b )
	Sub                    // ( a b -- a-b )
	Mul                    // ( a b -- a*b )
	Div                    // ( a b -- a/b ) integer division, truncates toward zero
	Mod                    // ( a b -- a%b ) remainder, sign follows the dividend
	Neg                    // ( a -- -a )
)
