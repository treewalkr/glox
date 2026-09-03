# Bytecode Specification — Chunk format and instruction set

**Normative.** This document pins the bytecode container, operand encoding,
and the complete instruction set. It is the **single source of truth** shared
by the VM (`internal/vm`), the assembler (Phase 7), the disassembler
(Phase 8), and the Glox compiler (Phase 9). Opcode numbers, once shipped,
are never renumbered; new opcodes are appended.

## 1. The Chunk

A compiled program is a `Chunk`:

```go
type Chunk struct {
    Code       []byte        // flat bytecode stream
    Constants  []value.Value // read-only literal pool
    NumGlobals int           // declared global slots (Phase 4+)
}
```

- `Code` is a flat byte stream; instructions are variable length
  (opcode + operands).
- `Constants` holds literals referenced by index. Deduplication is the
  producer's choice; the pool is read-only at runtime.
- Execution begins at `Code[0]`; the program result is the value stack top
  at `HALT` (see [vm.md §2](vm.md#2-machine-model)).

## 2. Operand encoding

Fixed rules, from Phase 1 onward — decided up front so hand-assembly is
stable:

1. **Endianness**: all multi-byte operands are **little-endian**
   (`binary.LittleEndian`).
2. **Index/address operands** (constant indices, call targets, global and
   local slots) are **unsigned `u16`** — 64 KiB of code, 64 Ki constants,
   64 Ki globals/locals per frame.
3. **Jump offsets** are **signed `i16`, relative to the address of the next
   instruction** (i.e. relative to the byte just after the offset's own two
   bytes). `target = pc_after_operands + offset`. Range: −32768…32767.
4. **Wide variants**: if a `u16` operand ever proves too small, a wide
   opcode variant is appended (e.g. `ConstantW` with `u32`); existing
   opcodes never change width.
5. Operand decoding **must** go through helpers in `internal/bytecode`
   (`ReadU16`, later `ReadI16`) so the encoding lives in exactly one place.
   The VM never slices `Code` directly.

### Worked encoding examples

`CONSTANT 1` (constants[1] = 4):

```text
01 01 00        ; opcode 0x01, then u16 index 1 little-endian
```

The `3 + 4` program (`CONSTANT 0; CONSTANT 1; ADD; HALT`):

```text
01 00 00        ; CONSTANT 0     push 3
01 01 00        ; CONSTANT 1     push 4
02              ; ADD            3 + 4
00              ; HALT
```

A backward jump. `JUMP :loop` sits at address 5, so its operand bytes occupy
6–7 and the next instruction is at 8. If `loop` is at address 2, the offset
is `2 − 8 = −6`, which is `0xFFFA` in two's-complement `i16`:

```text
15 FA FF        ; opcode 0x15, offset −6 little-endian
```

`CALL 0x0010 2` — call the function at address 16 with 2 arguments:

```text
19 10 00 02 00  ; opcode 0x19, target u16 = 0x0010, nArgs u16 = 2
```

## 3. Instruction set reference

The table below is **normative**. "Phase" is the roadmap phase that
introduces the opcode. Stack effects use bottom-to-top notation
(`vm.md §1`).

| Opcode   | Byte | Operands         | Stack effect          | Phase |
|----------|------|------------------|-----------------------|-------|
| `HALT`   | 0x00 | —                | `( -- )`              | 0/1   |
| `CONSTANT` | 0x01 | `idx u16`      | `( -- const[idx] )`   | 1     |
| `ADD`    | 0x02 | —                | `( a b -- a+b )`      | 1     |
| `SUB`    | 0x03 | —                | `( a b -- a-b )`      | 2     |
| `MUL`    | 0x04 | —                | `( a b -- a*b )`      | 2     |
| `DIV`    | 0x05 | —                | `( a b -- a/b )`      | 2     |
| `MOD`    | 0x06 | —                | `( a b -- a%b )`      | 2     |
| `NEG`    | 0x07 | —                | `( a -- -a )`         | 2     |
| `POP`    | 0x08 | —                | `( a -- )`            | 2     |
| `DUP`    | 0x09 | —                | `( a -- a a )`        | 2     |
| `SWAP`   | 0x0A | —                | `( a b -- b a )`      | 2     |
| `OVER`   | 0x0B | —                | `( a b -- a b a )`    | 2     |
| `ROT`    | 0x0C | —                | `( a b c -- b c a )`  | 2     |
| `PRINT`  | 0x0D | —                | `( a -- )`            | 2     |
| `EQ`     | 0x0E | —                | `( a b -- bool )`     | 3     |
| `NE`     | 0x0F | —                | `( a b -- bool )`     | 3     |
| `LT`     | 0x10 | —                | `( a b -- bool )`     | 3     |
| `GT`     | 0x11 | —                | `( a b -- bool )`     | 3     |
| `LE`     | 0x12 | —                | `( a b -- bool )`     | 3     |
| `GE`     | 0x13 | —                | `( a b -- bool )`     | 3     |
| `NOT`    | 0x14 | —                | `( a -- bool )`       | 3     |
| `JUMP`   | 0x15 | `off i16`        | `( -- )`              | 3     |
| `JUMPIFFALSE` | 0x16 | `off i16`   | `( cond -- )`         | 3     |
| `LOAD`   | 0x17 | `slot u16`       | `( -- globals[slot] )`| 4     |
| `STORE`  | 0x18 | `slot u16`       | `( v -- )`            | 4     |
| `CALL`   | 0x19 | `target u16, nArgs u16` | `( args -- ret )` | 5 |
| `RETURN` | 0x1A | —                | `( ret -- ret )`      | 5     |
| `LOADLOCAL` | 0x1B | `idx u16`    | `( -- locals[idx] )`  | 5     |
| `STORELOCAL` | 0x1C | `idx u16`   | `( v -- )`            | 5     |

Unassigned bytes (0x1D–0xFE, and 0xFF in particular) execute as
`ErrUnknownOpcode`. `0xFF` is deliberately never assigned and serves as the
canonical "unknown opcode" in tests.

Common errors (full table in [vm.md §8](vm.md#8-error-model)): truncated
operand reads, constant/global/local index out of range, stack underflow on
any pop, and type mismatches where noted below.

### Semantics, opcode by opcode

#### `HALT` — stop
Cease execution. The program result is the current value-stack top (if any).
No operands.

#### `CONSTANT idx` — push literal
Push `Constants[idx]`. Errors: `ErrTruncatedOperand`, `ErrConstantIndex`.

#### `ADD` — add / concatenate
Numbers: promote and add (`vm.md §3`). Strings: both operands strings →
concatenate (`a + b`, a is the deeper/left string). Any other combination:
`ErrTypeMismatch`.

#### `SUB`, `MUL` — arithmetic
Numbers only, promoted per `vm.md §3`. Otherwise `ErrTypeMismatch`.
`SUB ( a b -- a-b )` — `a` is the deeper value, `b` the top.

#### `DIV`, `MOD` — division
Numbers only. Integer `/` truncates toward zero; integer `%` follows Go
(sign of dividend). Zero divisor on integers: `ErrDivideByZero`. Float
division by zero yields IEEE ±Inf/NaN (not an error). Non-numbers:
`ErrTypeMismatch`.

#### `NEG` — negate
`( a -- -a )`. Numbers only; `ErrTypeMismatch` otherwise.

#### `POP` — drop
Discard the top value.

#### `DUP`, `SWAP`, `OVER`, `ROT` — Forth stack shuffles
```text
DUP   ( a     -- a a      )   copy the top
SWAP  ( a b   -- b a      )   exchange top two
OVER  ( a b   -- a b a    )   copy the second item to the top
ROT   ( a b c -- b c a    )   rotate third item to the top
```
All four require the implied number of values (`ErrStackUnderflow`).

#### `PRINT` — output
Pop one value and print its canonical text form ([vm.md §3](vm.md#3-value-model))
followed by a newline, to stdout.

#### `EQ`, `NE` — equality
Pop `b`, `a`; push `bool`. Equality is defined in [vm.md §3](vm.md#3-value-model):
numeric comparison across int/float, same-kind payload comparison otherwise,
different kinds simply unequal (never an error). `NE` pushes the negation.

#### `LT`, `GT`, `LE`, `GE` — ordering
Pop `b`, `a`; push `a < b`, `a > b`, `a <= b`, `a >= b` respectively.
Numbers only (int/float mix allowed, promoted); anything else is
`ErrTypeMismatch`.

#### `NOT` — logical not
`( a -- bool )`. Push `true` iff `a` is falsy (`false`, `nil`). Any value
kind is accepted.

#### `JUMP off` — unconditional relative jump
Set `pc = pc_after_operands + off` (signed `i16`, little-endian). A target
outside `[0, len(Code))` is `ErrBadJumpTarget`.

#### `JUMPIFFALSE off` — conditional relative jump
Pop `cond`. The operand **must** be a bool (`ErrTypeMismatch` otherwise —
this is a type-checked conditional branch, not a C-style zero test). If
`cond == false`, jump exactly as `JUMP` does; otherwise continue.

#### `LOAD slot` / `STORE slot` — globals
Push/pop `globals[slot]`. Requires `slot < NumGlobals`, else
`ErrGlobalIndex`. Globals are the flat, pre-sized array described in
[vm.md §6](vm.md#6-globals).

#### `CALL target nArgs` — call function
Follows the calling convention of [vm.md §7](vm.md#7-calling-convention):
pop `nArgs` values into a fresh frame (`locals[i] = argument i`, argument 0
deepest), push the frame with `ReturnPc` after this instruction, set
`pc = target` (absolute). Errors: `ErrStackUnderflow`, `ErrCallStackOverflow`.

#### `RETURN` — return from function
The return value is already on top of the value stack; leave it there. Pop
the call frame and resume at its `ReturnPc`. Only the script frame on the
call stack → `ErrNoCallFrame`.

#### `LOADLOCAL idx` / `STORELOCAL idx` — frame locals
Push/pop the current frame's `locals[idx]`. `STORELOCAL` grows the frame's
locals array as needed (bounds: `u16` index space). `LOADLOCAL` of a slot
never stored in this frame is `ErrLocalIndex`.

## 4. Entry and exit conventions

- Execution starts at `pc = 0` with an empty value stack, an empty call
  stack plus the script frame, and zero-valued globals.
- A well-formed chunk ends its top-level path in `HALT`. Running past the
  final byte without `HALT` is `ErrTruncatedOperand`.
- The program result is the value-stack top at `HALT`.
