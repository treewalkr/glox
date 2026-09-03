# Stack Machine VM — Implementation Plan (Go)

A phased roadmap for building the educational stack-machine VM bottom-up, in Go.
Each phase ends in a **runnable, testable milestone** — do not start the next
phase until the current one's "Definition of Done" passes.

> Convention: stack effects are written as `( before -- after )`.
> Example: `ADD ( a b -- a+b )` pops `a` then `b`, pushes the sum.

### Operand encoding (fixed from Phase 1 onward)

Hand-assembling bytecode is impossible without pinning this down, so decide it
up front and never change it mid-plan:

- Operands are **little-endian**, `u16` for indices/offsets/targets
  (`binary.LittleEndian.Uint16`), giving 64 KiB of code and 64 Ki constants —
  plenty for an educational VM.
- If a `u16` operand ever proves too small, add a wide variant opcode
  (e.g. `ConstantW`) rather than changing the width of an existing opcode.
- Jump offsets are **signed i16 relative to the address of the next
  instruction** (i.e. after the operand bytes). Document this in `opcode.go`.
- Record each opcode's operand count and widths in the opcode table below so
  the assembler (Phase 7) and disassembler (Phase 8) share one source of truth.

This mirrors `../rs/PLAN.md` phase-for-phase; only the build commands, package
layout, and a few language-specific notes differ.

---

## Phase 0 — Foundation (data structures, no execution yet)

**Goal:** define the vocabulary of the VM with no logic that runs.

- [ ] `internal/opcode/opcode.go` — `type Opcode byte` with `iota`; just `Halt` for now.
- [ ] `internal/value/value.go` — `type Value int64` (start integers-only).
- [ ] `internal/bytecode/chunk.go` — `Chunk` struct `{ Code []byte; Constants []value.Value }`
      with a reader that yields the next byte and tracks position.
- [ ] `internal/errors/errors.go` — sentinel `var ErrXxx = errors.New(...)`.
- [ ] `main.go` imports nothing yet beyond stdlib.

**Definition of Done:** `go build ./...` is green; a `Chunk` can be constructed
in a unit test and read byte-by-byte. No VM exists yet.

---

## Phase 1 — Minimal running VM (the "hello world")

**Goal:** prove the fetch-decode-execute loop end-to-end.

- [ ] Add opcodes: `Constant` (read index, push from constant pool), `Add`, `Halt`.
- [ ] `internal/vm/vm.go` — `type Vm struct { stack []value.Value; pc int }` with a
      `Run()` loop: fetch byte → `switch` opcode → execute → advance `pc`.
- [ ] `Add` pops two, pushes one. `Halt` stops the loop.
- [ ] Every pop checks for **stack underflow** and returns `ErrStackUnderflow`
      (no panics, from day one).
- [ ] Operand decoding helpers live in `bytecode` (`ReadU16`), so the VM never
      slices `Code` directly.
- [ ] `main.go` hand-assembles `3 + 4` → bytecode → run → print stack top.

**Definition of Done:**
```text
CONSTANT 0   ; constants = [3]
CONSTANT 1   ; constants = [3, 4]
ADD
HALT
```
Unit test asserts the stack top is `7` after `Run()`.

---

## Phase 2 — Arithmetic & stack manipulation

**Goal:** full integer arithmetic plus the classic Forth stack ops.

- [x] Arithmetic: `Sub`, `Mul`, `Div`, `Mod`, `Neg`.
- [x] Division-by-zero returns a sentinel error (no panic).
- [x] Stack ops: `Pop`, `Dup`, `Swap`, `Over`, `Rot` (optional).
- [x] Add a `Print` opcode that pops and prints.

**Definition of Done:** a test computes `(10 - 2) * 3 + 4 / 2` and asserts `26`.

---

## Phase 3 — Control flow  🎯 (VM becomes Turing complete here)

**Goal:** make the VM able to branch and loop.

- [ ] Comparison opcodes that push a bool: `Eq`, `Ne`, `Lt`, `Gt`, `Le`, `Ge`.
- [ ] Minimal bool support: extend `Value` to a tagged struct with `Integer`
      and `Bool` kinds only (floats/strings/nil wait for Phase 6).
- [ ] `Jump(offset)` — relative jump per the encoding rules above.
- [ ] `JumpIfFalse(offset)` — pop top; jump if false.

**Definition of Done:** hand-assemble a loop that counts `0..5` and prints each
value; test asserts the printed/returned sequence.

---

## Phase 4 — Variables

**Goal:** named storage beyond the stack.

- [ ] Global slots: `Load(slot)` / `Store(slot)` backed by a `[]value.Value`.
- [ ] `Halt` returns the top-of-stack as the program's result value.
- [ ] Optional: `LoadConst`/`StoreConst` shortcuts.

**Definition of Done:** compute and store a value, then `Load` it back and use
it in another expression. Test verifies persistence across operations.

---

## Phase 5 — Functions & call stack

**Goal:** first-class functions with proper scoping.

- [ ] Separate **call stack** of `Frame { returnPc int; locals []value.Value }`
      (do not reuse the value stack for return addresses).
- [ ] Opcodes: `Call(target, nArgs)` (so the callee knows how many args to pop
      into its locals) and `Return`.
- [ ] Per-frame locals: `LoadLocal(idx)` / `StoreLocal(idx)`.
- [ ] Define a calling convention (e.g. arguments pushed left-to-right, callee
      pops them into locals) and document it.

**Definition of Done:** a "factorial" or "fibonacci" function written in
hand-assembled bytecode returns the correct value.

---

## Phase 6 — Richer data types

**Goal:** stop being integers-only.

- [ ] Extend `Value` with the remaining kinds: `Float(float64)`, `Nil`,
      `Object *Obj` (the tagged struct itself arrives in Phase 3).
- [ ] Make arithmetic opcodes type-aware (int + int → int; promote on mixed).
- [ ] String literal support via the constant pool.

**Definition of Done:** concatenate or compare strings; mix int/float arithmetic
and assert correct promotion.

---

## Phase 7 — Assembler (text → bytecode)

**Goal:** stop hand-writing byte slices by hand.

- [ ] A tiny line-oriented assembly language, e.g.:
      ```text
      CONSTANT 3
      CONSTANT 4
      ADD
      HALT
      ```
- [ ] `internal/asm/asm.go` parses lines → `Chunk`. Support labels and label
      resolution for jumps (`JUMP :loop`).
- [ ] `main.go` loads an `.sms` (or `.asm`) file and runs it.

**Definition of Done:** a file like `examples/fib.sms` runs end-to-end from CLI.

---

## Phase 8 — Disassembler & debugging

**Goal:** make the VM observable.

- [ ] `internal/asm/disasm.go` pretty-prints a `Chunk`: address, opcode, operands,
      stack-effect hint.
- [ ] `-trace` flag prints each instruction + the stack after every step.
- [ ] `-dump` flag prints the disassembled bytecode and exits.

**Definition of Done:** `go run . -trace examples/fib.sms` shows every
fetch-decode-execute step with the stack state.

---

## Phase 9 — Source language & compiler (stretch)

**Goal:** graduate from assembly to a real high-level language.

- [ ] `internal/lexer/lexer.go` — tokenize a small language.
- [ ] `internal/parser/parser.go` — recursive-descent parser → AST.
- [ ] `internal/compiler/compiler.go` — AST → `Chunk` (single pass, no optimization).
- [ ] Language floor: arithmetic, `let`, `if/else`, `while`, functions, `print`.

**Definition of Done:** run a real program like:
```text
fn fib(n) { if n < 2 { return n } return fib(n-1) + fib(n-2) }
print fib(10)
```

---

## Phase 10 — Explicit object heap (stretch, optional for Go)

**Goal:** mirror the Rust version's heap/GC milestone as a learning exercise.

- [ ] Allocate objects on a VM-owned heap; `Value` holds an index/handle into it.
- [ ] Implement mark-and-sweep rooted at the value stack, call frames, globals.
- [ ] `-gc-stats` flag reports allocations and collections.

> Note: Go already garbage-collects heap allocations, so this is **optional** —
> useful only if you want to learn how a GC works by building one. Otherwise skip
> straight to Phase 11.

**Definition of Done:** a tight loop allocating strings runs forever without
unbounded memory growth *in your own heap*.

---

## Phase 11 — REPL & ergonomics (stretch)

**Goal:** interactive usage.

- [ ] `internal/repl/repl.go` — read a line, compile, run, print result, loop.
- [ ] Use a line-reading lib (e.g. `github.com/chzyer/readline`) for history/editing.
- [ ] Helpful errors: opcode name + pc offset + surrounding bytecode context.

**Definition of Done:** launch with no args, type expressions, get results live.

---

## Opcode reference (grow this as you implement)

| Opcode | Stack effect | Phase |
|--------|--------------|-------|
| `Halt` | `( -- )` | 0/1 |
| `Constant(i)` | `( -- const[i] )` | 1 |
| `Add` | `( a b -- a+b )` | 1 |
| `Sub` | `( a b -- a-b )` | 2 |
| `Mul` | `( a b -- a*b )` | 2 |
| `Div` | `( a b -- a/b )` | 2 |
| `Mod` | `( a b -- a%b )` | 2 |
| `Neg` | `( a -- -a )` | 2 |
| `Pop` | `( a -- )` | 2 |
| `Dup` | `( a -- a a )` | 2 |
| `Swap` | `( a b -- b a )` | 2 |
| `Over` | `( a b -- a b a )` | 2 |
| `Print` | `( a -- )` | 2 |
| `Eq/Ne/Lt/Gt/Le/Ge` | `( a b -- bool )` | 3 |
| `Jump(off)` | `( -- )` | 3 |
| `JumpIfFalse(off)` | `( bool -- )` | 3 |
| `Load(slot)` / `Store(slot)` | `( -- v ) / ( v -- )` | 4 |
| `Call(target, nArgs)` / `Return` | `( args -- ret )` | 5 |
| `LoadLocal(i)` / `StoreLocal(i)` | `( -- v ) / ( v -- )` | 5 |

---

## Cross-cutting rules (apply to every phase)

- `gofmt` + `go vet ./...` clean before marking a phase done.
- Every new opcode gets a bytecode-level, table-driven test in the `vm` package.
- No `panic` in library code — return `error`.
- No `unsafe`, no `reflect` without a documented justification in a comment.
- One concept per commit; commit message follows Conventional Commits.
