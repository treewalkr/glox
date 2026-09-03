# VM Specification — the Glox stack machine

**Normative.** This document defines the runtime architecture of the Glox
virtual machine: the machine model, the value model, limits, the calling
convention, and the error model. The bytecode encoding and opcode reference
live in [bytecode.md](bytecode.md).

## 1. Overview

Glox VM is a **bytecode-interpreted stack machine**. Operands for every
operation are taken implicitly from the top of a **value stack**, and results
are pushed back onto it. Instructions therefore need **zero explicit
register operands** for arithmetic — the stack *is* the operand source.

Computing `3 + 4`:

```text
CONSTANT 0   ; push constants[0] = 3    stack: [3]
CONSTANT 1   ; push constants[1] = 4    stack: [3, 4]
ADD          ; pop 4, pop 3, push 7     stack: [7]
HALT
```

Contrast with a register machine: `ADD R1, R2, R3` (three explicit operands).

### Stack-effect notation

Throughout the specs, stack effects are written `( before -- after )`. The
symbols list values **bottom-to-top, left to right**: the rightmost symbol is
the **top** of the stack.

```text
ADD  ( a b -- a+b )     ; pops b (top) first, then a; pushes a+b
SUB  ( a b -- a-b )     ; pops b, then a; pushes a-b (a is the deeper value)
```

So `CONSTANT 3; CONSTANT 4; SUB` computes `3 - 4 = -1`.

## 2. Machine model

The machine consists of five parts:

```text
            ┌────────────────────────────────────────────┐
            │  Chunk                                     │
            │   Code      []byte      bytecode           │
            │   Constants []Value     constant pool      │
            │   NumGlobals int        declared globals   │
            └────────────────────────────────────────────┘
                        ▲                 ▲
            ┌───────────┴─────────┐  ┌────┴──────────────┐
            │  pc  (int)          │  │  globals []Value  │
            │  index into Code of │  │  slot-indexed     │
            │  the next opcode    │  │  named storage    │
            └───────────┬─────────┘  └───────────────────┘
                        │
        ┌───────────────┴───────────────┐
        │  value stack  []Value         │  operands & results
        │  call stack   []Frame         │  function activation records
        └───────────────────────────────┘
```

- **Value stack** — holds operands. Operations pop inputs and push outputs.
- **Call stack** — a *separate* stack of `Frame` records holding return
  addresses and locals. Return addresses must never share the value stack.
- **Program counter (pc)** — index into `Code` of the next opcode byte to
  fetch.
- **Globals** — slot-indexed named storage (`[]value.Value`), sized from the
  chunk's declared `NumGlobals` (see §6).
- **Constant pool** — read-only literals shared by all frames.

### Fetch–decode–execute loop

Execution starts at `pc = 0` and proceeds:

1. **Fetch**: read the opcode byte at `pc`; advance `pc` by one.
2. **Decode**: if the opcode has operands, read them (little-endian; widths
   per the [opcode table](bytecode.md#3-instruction-set-reference)) and
   advance `pc` past them. Operand reads **must** go through the `bytecode`
   package helpers (`ReadU16`, and later `ReadI16`); the VM never slices
   `Code` directly.
3. **Execute**: apply the opcode's semantics, popping/pushing as its stack
   effect states.
4. Repeat until `HALT` or an error.

Falling off the end of `Code` without executing `HALT` is a runtime error.
The program's **result** is the value on top of the value stack when `HALT`
executes (an empty stack means no result).

### Frame

```go
type Frame struct {
    ReturnPc int      // code address to resume at after Return
    Locals   []value.Value
}
```

The VM starts with a single **script frame** (depth 0) that owns the
top-level code. `Return` executed while only the script frame remains is the
error `ErrNoCallFrame`.

## 3. Value model

The value model grows in three steps; each step is a superset of the last.

| Phase | Go type                                  | Kinds                        |
|-------|------------------------------------------|------------------------------|
| 0–2   | `type Value int64`                       | int                          |
| 3–5   | tagged struct                            | int, bool                    |
| 6+    | tagged struct                            | int, float, bool, nil, string|
| 10 (optional) | tagged struct + VM heap         | + object handles             |

Reference design for the tagged struct:

```go
type Kind uint8

const (
    KindInt Kind = iota
    KindFloat
    KindBool
    KindNil
    KindString
    KindObject // optional Phase 10
)

type Value struct {
    Kind  Kind
    Int   int64
    Float float64
    Bool  bool
    Str   string
    Obj   uint32 // handle into the VM-owned heap (Phase 10, optional)
}
```

An `interface`-based representation is a permitted alternative; the *kinds*
and their semantics below are normative either way.

### Numeric semantics

- Integers are **signed 64-bit**. Arithmetic wraps silently on overflow
  (documented behavior, not an error).
- Floats are IEEE-754 **binary64**.
- **Promotion**: for `+ - * / %`, if either operand is a float the other is
  converted to float and the result is a float; `int ⊕ int → int` otherwise.
- **Integer division** truncates toward zero; integer `/` or `%` with a zero
  divisor is the error `ErrDivideByZero`.
- **Float division** by zero produces IEEE infinity/NaN — not an error.
- `%` on floats follows Go's `math.Mod` (sign follows the dividend).

### Truthiness

Exactly two values are **falsy**: `false` and `nil`. Everything else —
including `0`, `0.0`, and `""` — is truthy.

### Canonical text forms

Used by `Print`, the disassembler, the tracer, and the REPL:

| Kind   | Text form                                   |
|--------|---------------------------------------------|
| int    | decimal (`strconv.FormatInt`)               |
| float  | shortest form that round-trips (Go `%v`)    |
| bool   | `true` / `false`                            |
| nil    | `nil`                                       |
| string | the string's contents, unquoted, unescaped  |

### Equality

`Eq` / `Ne` accept any two values. Two values are equal iff:

- both are numbers (int or float, mixed allowed) and numerically equal, or
- they have the same kind and equal payloads (strings compare by content,
  `nil == nil` is true, bools by truth).

Values of different kinds (outside the numeric case) are never equal;
comparing them is **not** an error — it just yields `false` for `Eq`.

## 4. Limits

| Resource            | Default limit | Enforced by          |
|---------------------|---------------|----------------------|
| Code size           | 65,536 bytes  | `u16`/`i16` operands |
| Constant pool       | 65,536 values | `u16` index          |
| Globals             | 65,536 slots  | `u16` slot index     |
| Frame locals        | 65,536 slots  | `u16` index          |
| Value stack depth   | 1,024 values  | `ErrStackOverflow`   |
| Call stack depth    | 256 frames    | `ErrCallStackOverflow` |

Value-stack and call-stack limits may be raised or lowered through VM
options; the rest are structural (fixed by operand width). Jump offsets are
`i16`, so a single jump spans at most 32,767 bytes forward — forward jumps
in a 64 KiB program longer than that must be routed through an intermediate
trampoline label.

If an opcode ever needs a wider operand, a **wide variant** opcode is
appended (e.g. `ConstantW` with a `u32` operand); existing opcode numbers and
operand widths never change.

## 5. Value stack discipline

- Every pop checks for underflow and returns `ErrStackUnderflow` — the VM
  never indexes into an empty slice.
- Every push checks the depth limit and returns `ErrStackOverflow`.
- The VM exposes a **copy** of the stack for inspection (`Stack()`); callers
  must not alias VM internals.

## 6. Globals

Global variables are a flat `[]value.Value` of `NumGlobals` slots, declared
by the chunk (the assembler's `.globals N` directive, or the compiler's
slot allocator). Slots are zero-valued until stored.

- `Load slot` / `Store slot` with `slot >= NumGlobals` is the error
  `ErrGlobalIndex`.
- There is no "grow the globals" opcode; the count is fixed per chunk.

The compiler assigns every top-level binding (from `let` and `fn`) a unique
global slot. See [glox.md §7](glox.md#7-declarations-scoping-and-names).

## 7. Calling convention

**Normative.** All function calls — from Glox source or hand-written
assembly — follow one convention:

1. The caller evaluates arguments **left to right**, pushing each onto the
   value stack. Argument 0 is the deepest.
2. The caller executes `Call target nArgs`. The VM:
   - pops `nArgs` values (underflow if fewer are present) into a **new
     frame's** locals such that `locals[i] = argument i`;
   - pushes the frame with `ReturnPc` = address just past the `Call`
     operands (call-stack overflow if at the frame limit);
   - sets `pc = target` (an absolute code address).
3. The callee runs. Its locals live in the frame: `LoadLocal idx` /
   `StoreLocal idx`, `u16`-indexed. `StoreLocal` grows the frame's local
   array on demand (up to 65,536 slots); `LoadLocal` of a slot that was
   never stored is the error `ErrLocalIndex`.
4. The callee leaves its **return value** on top of the value stack and
   executes `Return`. The VM pops the frame (error `ErrNoCallFrame` if only
   the script frame remains) and resumes at `ReturnPc`. The value stack is
   otherwise untouched, so the net effect of a call is
   `( arg0 .. argN-1 -- ret )`.

The invariant "exactly one return value sits on top at `Return`" is the
callee author's/compiler's responsibility; the VM does not verify it.
Glox-compiled functions always satisfy it: the compiler appends an implicit
`CONSTANT nil; RETURN` if control can fall off the end of a function body.

Arity is **compile-time checked** in Glox (call sites know the callee's
parameter count). Hand-written bytecode that calls with the wrong `nArgs`
simply misaligns the callee's locals — a documented sharp edge.

## 8. Error model

**Normative.** Every VM failure is a Go `error`, never a panic. Sentinel
errors are package-level values in `internal/errors`, comparable with
`errors.Is`. Lower layers may **wrap** sentinels with context (opcode, pc)
using `%w`.

| Sentinel                  | Raised by                                       | Phase |
|---------------------------|-------------------------------------------------|-------|
| `ErrStackUnderflow`       | any pop from an empty value stack               | 1     |
| `ErrStackOverflow`        | value stack past its limit                      | 2     |
| `ErrDivideByZero`         | integer `Div`/`Mod` with zero divisor           | 2     |
| `ErrUnknownOpcode`        | fetch of an unassigned opcode byte              | 1     |
| `ErrTruncatedOperand`     | operand bytes run past end of `Code`; also PC past end of code | 1 |
| `ErrConstantIndex`        | `Constant` index outside the pool               | 1     |
| `ErrTypeMismatch`         | an operand's kind is wrong for the opcode       | 3     |
| `ErrBadJumpTarget`        | jump target outside `[0, len(Code))`            | 3     |
| `ErrGlobalIndex`          | `Load`/`Store` slot `>= NumGlobals`             | 4     |
| `ErrCallStackOverflow`    | `Call` past the frame limit                     | 5     |
| `ErrNoCallFrame`          | `Return` with only the script frame             | 5     |
| `ErrLocalIndex`           | `LoadLocal` of a never-stored slot              | 5     |
| `ErrUndefinedVariable`    | Glox: global read before its `let` executed     | 9     |

## 9. Object heap and GC (Phase 10, optional)

Go already garbage-collects heap allocations, so this phase exists purely to
learn how a GC works. If implemented:

- The VM owns an explicit object heap; `Value.Obj` is a handle (index) into
  it, never a raw pointer.
- Collection is **mark-and-sweep**. Roots: the value stack, every frame's
  locals, and the globals array.
- `-gc-stats` reports allocation counts and collection counts.
- A tight loop allocating objects must run forever without unbounded growth
  of the *VM-owned* heap.

## 10. Go API surface

**Informative** — the reference API shape:

```go
chunk := bytecode.Chunk{
    Code:      []byte{...},
    Constants: []value.Value{3, 4},
    NumGlobals: 0,
}

machine := vm.New(chunk)           // future: vm.New(chunk, options...)
if err := machine.Run(); err != nil {
    // errors.Is(err, vmerrors.ErrStackUnderflow) etc.
}

top := machine.Stack()             // copy of the value stack, bottom-first
```

Rules: `Run()` is single-threaded (no concurrency, by design); a `Vm` runs
one chunk; re-running a finished VM restarts from `pc = 0` with cleared
stacks (not guaranteed across versions — prefer a fresh `Vm`).
