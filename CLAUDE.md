# CLAUDE.md (Go implementation)

Guidance for Claude Code (and any AI coding agent) working in the **Go**
implementation of the educational stack machine.

This is the Go sibling of the Rust version in `../rs/`. Both implement the same
VM and follow the same phased plan, but this one uses Go idioms.

## Project Overview

An **educational stack machine** virtual machine (VM) implemented in Go.

The goal is to learn how VMs, bytecode interpreters, and instruction set
architectures (ISAs) work by building one from scratch — not to be the fastest
or most production-ready runtime. Clarity and teachability beat cleverness.

- **Language:** Go
- **Type:** Bytecode-interpreted stack-based VM
- **Audience:** Learners. Optimize for readable code and clear explanations over
  raw performance. Comment the *why*, especially for non-obvious VM internals.

## What Is a Stack Machine?

A stack machine is an ISA where operands for operations are implicitly taken
from the top of a stack, and results are pushed back onto the stack.
Instructions therefore need **zero explicit operands** for arithmetic — the
stack *is* the operand source.

Example — computing `3 + 4`:

```
CONSTANT 0   ; push constants[0]=3   ->  stack: [3]
CONSTANT 1   ; push constants[1]=4   ->  stack: [3, 4]
ADD          ; pops 3 and 4, pushes 7  ->  stack: [7]
```

Contrast this with a register machine: `ADD R1, R2, R3` (three explicit operands).

## Design Conventions

### VM architecture
- A **value stack** holds operands. Operations pop inputs and push outputs.
- A **call stack** (separate from the value stack) holds return addresses and
  frames if/when function calls are supported.
- A **program counter (PC)** points at the next instruction in bytecode.
- Bytecode is a flat `[]byte` (with a constants slice). Execution is a
  fetch-decode-execute loop.

### Go conventions
- Represent instructions as `type Opcode byte` with `iota` constants so each
  serializes to a single byte.
- Start with `type Value int64` (integers only); expand to a tagged `struct`
  with a `Kind` field (or an `interface`) once floats/bools/strings appear.
- Use a newtype for clarity where it helps: `type Pc int`.
- Error handling: return `error` from every operation that can fail. **No
  `panic`/`recover` in library code** — panics are fine only for truly
  unrecoverable programmer bugs in `main`.
- Define sentinel errors as package-level `var ErrXxx = errors.New(...)`,
  checked with `errors.Is`.
- No `unsafe` and no `reflect` unless there is a documented, justified reason.

### Code style
- `gofmt` (or `goimports`) before committing.
- `go vet ./...` clean.
- `golangci-lint run` clean if available.
- Group code into packages by concern, under `internal/`.

## Suggested Project Layout

```
go/
  go.mod
  main.go                  # package main: CLI entry point / REPL
  internal/
    opcode/opcode.go       # Opcode type + iota constants
    value/value.go         # the Value type
    bytecode/chunk.go      # Chunk container + reading helpers
    vm/vm.go               # the interpreter: fetch-decode-execute loop
    errors/errors.go       # sentinel VM errors
    compiler/compiler.go   # (later) source -> bytecode
    asm/asm.go             # (later) text assembly -> Chunk
```

> Tip: it's fine to start with everything in `package main` flat at the root and
> only split into `internal/` packages once a file gets crowded. `internal/`
> prevents outside importers from reaching into your implementation details.

Build the VM bottom-up: `opcode` -> `value` -> `bytecode` -> `vm`. Add a
compiler/assembler only once hand-assembled bytecode runs correctly.

## Build & Test Commands

```bash
go run .                   # build and run
go build ./...             # build everything
go test ./...              # run all tests
go test ./internal/vm/     # run tests in a specific package
go vet ./...               # lint
gofmt -w .                 # format in place
```

When adding an opcode, add a table-driven test in the `vm` package that
assembles a few bytes, runs the VM, and asserts the resulting stack.

## Testing Approach

- **Bytecode-level tests** are the core: assemble small programs, run them,
  assert the final stack/return value. This keeps tests fast and deterministic.
- Use Go's table-driven tests (`tests := []struct{...}`) with `t.Run`.
- Keep an opcode reference table (in comments or docs) mapping each opcode to
  its stack effect, e.g. `ADD  ( a b -- a+b )`.

## Out of Scope (for now)

- JIT compilation — we are interpreter-only for educational clarity.
- Full garbage collection — Go's runtime already GCs heap values, so the
  *language* doesn't force you to build a GC. (This is a key difference from the
  Rust version, which has a Phase 10 GC milestone.) If you later model an
  explicit object heap to mirror the Rust version, that's an optional exercise.
- Goroutines / concurrency — single-threaded by design.

## References

- Koopman, *Stack Computers: The New Wave* — taxonomy of stack machine designs.
- Nystrom, *Crafting Interpreters* — practical bytecode VM (port the ideas to Go).
- Thorsten Ball, *Writing An Interpreter In Go* — Go-native VM walkthrough.
