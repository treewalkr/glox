# Glox Specification

Glox is an **educational stack-machine virtual machine** and a small
**dynamically typed language** of the same name ("**Go + Lox**"), implemented
in Go. The goal is to learn how VMs, bytecode interpreters, and instruction
set architectures work by building one from scratch — clarity and
teachability beat performance everywhere in this design.

There are three layers, each specified in its own document:

```
glox source (.glox)  ──compile──▶  bytecode (Chunk)  ──run──▶  results
        ▲                              ▲
   glox.md                       bytecode.md + vm.md
                                        ▲
        text assembly (.sms)  ──assemble──┘   (asm.md)
```

## Document map

| Document          | Contents                                                        | Role       |
|-------------------|-----------------------------------------------------------------|------------|
| [vm.md](vm.md)     | Machine model, value model, limits, calling convention, errors | Normative  |
| [bytecode.md](bytecode.md) | Chunk container, operand encoding, full opcode reference | Normative |
| [asm.md](asm.md)   | The `.sms` text assembly language, disassembler, tracing        | Normative  |
| [glox.md](glox.md) | The Glox source language: lexical structure, grammar, semantics | Normative  |

Supporting context lives outside `docs/`:

- `PLAN.md` — the phased implementation roadmap (governs *sequencing* of work).
- `CLAUDE.md` — Go conventions and project ground rules.

## How to read these specs

- Sections marked **Normative** define required behavior. The key words
  **must**, **must not**, and **may** are used as in RFC 2119 (lightly).
- Sections marked **Informative** explain *how* the reference implementation
  is expected to work; other implementations are free to differ.
- Unless stated otherwise, the specs describe the **complete machine**
  (Phases 0–9). Features land phase-by-phase per `PLAN.md`; every opcode row
  lists the phase in which it appears.
- If a spec here conflicts with `PLAN.md` on encoding or behavior,
  **this spec set wins**. If it conflicts with the code, the code shows what
  exists *today* and should be brought up to spec.

## Design principles

1. **Clarity over cleverness.** The VM is a teaching artifact; a slow,
   obvious `switch` loop is correct.
2. **Errors, never panics.** Every failure is a sentinel `error` value
   checkable with `errors.Is`; no `panic`/`recover` in library code.
3. **One source of truth for encoding.** Operand widths and opcode numbers
   are pinned once (in [bytecode.md](bytecode.md)) and never change
   mid-flight; the assembler, disassembler, and VM all share that table.
4. **Bottom-up construction.** Data structures first, then a minimal loop,
   then features — each phase ends runnable and tested.
5. **Bounded and observable.** Explicit limits (stack depth, code size) and
   explicit tooling (`-trace`, `-dump`) make the machine inspectable.

## Implementation status

Current state (Phase 0–2 complete, everything else planned):

| Phase | Milestone                                   | Status   |
|-------|---------------------------------------------|----------|
| 0     | Foundation: opcode/value/chunk/errors types | done     |
| 1     | Minimal VM: `Constant`, `Add`, `Halt`       | done     |
| 2     | Arithmetic & stack manipulation             | done     |
| 3     | Control flow (Turing complete)              | planned  |
| 4     | Global variables                            | planned  |
| 5     | Functions & call stack                      | planned  |
| 6     | Richer data types (float/nil/string)        | planned  |
| 7     | Assembler (`.sms` text → bytecode)          | planned  |
| 8     | Disassembler & debugging (`-trace`, `-dump`)| planned  |
| 9     | Glox source language & compiler             | planned  |
| 10    | Explicit object heap + mark-sweep GC        | optional |
| 11    | REPL                                        | stretch  |

## Tooling & CLI (target interface)

```text
go run .                    # no args: REPL (Phase 11, stretch)
go run . program.glox       # compile + run Glox source (Phase 9)
go run . program.sms        # assemble + run text assembly (Phase 7)

go run . -trace program.sms # print every fetch-decode-execute step
go run . -dump program.sms  # disassemble and exit (see asm.md)
go run . -gc-stats prog.glox # GC statistics (Phase 10, optional)
```

Development commands (from the repo root):

```text
go build ./...    go test ./...    go vet ./...    gofmt -w .
```
