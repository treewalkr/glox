package main

import (
	"encoding/binary"
	"fmt"
)

func main() {
	a := binary.LittleEndian.Uint16([]byte{0xff, 0x01})
	fmt.Printf("%#x\n", a)
}
