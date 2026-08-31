package opcode

type Opcode byte

const (
	Halt     Opcode = iota // stop execution
	Constant               // (idx -- const[idx])
	Add                    // (a b -- a+b)
)
