# Assembly, Disassembly, and Debugging Specification

**Normative.** This document defines **SMS** ("stack-machine source"), the
text assembly language for the Glox VM (Phase 7), plus the disassembler and
trace output formats (Phase 8). SMS is a thin, line-oriented skin over the
bytecode of [bytecode.md](bytecode.md) — every mnemonic maps 1:1 to an
opcode.

Files use the `.sms` extension.

## 1. Design goals

- Human-writable: labels instead of numeric addresses, literals instead of
  pool indices.
- Human-readable round-trip: `-dump` output is valid SMS input (minus
  comments) so disassembly can be re-assembled.
- Two-pass: labels may be referenced before they are defined (forward
  references), enabling "main first, functions after" layouts.

## 2. Lexical structure

An SMS file is a sequence of lines. Each line is a sequence of
whitespace-separated tokens.

- **Comments**: `;` starts a comment running to end of line. Comments may
  appear anywhere, including after instructions.
- **Case**: mnemonics and directives are case-insensitive (`JUMP` = `jump`);
  labels are case-sensitive.
- **Numbers**: decimal (`42`, `-5`), hex (`0x1F`). Must fit the operand
  width of their instruction.
- **Strings**: double-quoted, with the same escapes as Glox string literals
  (`\\ \" \n \t \r \0`).
- **Identifiers**: same rules as Glox (`[A-Za-z_][A-Za-z0-9_]*`).

## 3. Lines

Each line is one of:

```text
label-definition   ::= IDENTIFIER ":"
instruction        ::= MNEMONIC ( operand ( "," operand )* )?
directive          ::= "." DIRECTIVE argument
comment/blank      ::= (ignored)
```

A label definition may share a line with an instruction:

```text
loop:  LOAD 0        ; label 'loop' points at the LOAD
```

### Labels

`name:` binds `name` to the code address of the next emitted byte. Duplicate
label definitions are an assembly error. Labels may be referenced before
their definition. Referencing an undefined label is an assembly error.

Label references use `:name` in jump/call operand position (bare `name` is
also accepted where unambiguous; `:name` is canonical).

## 4. Directives

| Directive   | Argument  | Meaning                                          |
|-------------|-----------|--------------------------------------------------|
| `.globals`  | `N`       | Set the chunk's `NumGlobals = N` (default 0)     |

`.globals` may appear at most once, anywhere in the file. Every `LOAD` /
`STORE` slot must be `< N` or assembly (or runtime) fails.

## 5. Instructions

Mnemonics and operands correspond exactly to the
[instruction set reference](bytecode.md#3-instruction-set-reference).
Operands in SMS:

| Instruction   | SMS operands                            | Notes |
|---------------|-----------------------------------------|-------|
| `HALT`        | —                                       |       |
| `CONSTANT`    | a **literal** (or `@idx`)               | see below |
| `ADD` `SUB` `MUL` `DIV` `MOD` `NEG` | —                  |       |
| `POP` `DUP` `SWAP` `OVER` `ROT` `PRINT` | —               |       |
| `EQ` `NE` `LT` `GT` `LE` `GE` `NOT`  | —                  |       |
| `JUMP`        | `:label` or integer offset              | label = absolute target |
| `JUMPIFFALSE` | `:label` or integer offset              | label = absolute target |
| `LOAD` `STORE`| slot number                             |       |
| `CALL`        | `:label` (or address) `,` nArgs         | two operands |
| `RETURN`      | —                                       |       |
| `LOADLOCAL` `STORELOCAL` | local index                     |       |

### The `CONSTANT` operand

`CONSTANT 42` interns the literal `42` into the constant pool (deduplicated
against earlier constants) and emits `CONSTANT <pool-index>`. Accepted
literal forms:

- integers: `42`, `-7`
- floats: `3.14` (once Phase 6 lands)
- strings: `"hi\n"` (once Phase 6 lands)
- `true`, `false`, `nil` (once Phase 6 lands)
- `@idx` — reference pool index `idx` explicitly instead of interning

### Jump operands

At the SMS level you write the **absolute target** (a label or an address);
the assembler computes the `i16` offset relative to the next instruction
and errors if it does not fit. Raw integer jump operands are interpreted as
absolute addresses too — offsets are a bytecode-level concern only.

## 6. Assembly algorithm

**Informative.** Two passes:

1. **Layout pass**: walk the lines, compute each instruction's encoded size
   from the [opcode table](bytecode.md#3-instruction-set-reference), assign
   addresses to labels. Constant interning starts here.
2. **Emit pass**: re-walk, encode instructions, resolving label references
   to addresses (and jump offsets). 

Assembly errors (all reported with file line numbers):

- unknown mnemonic or directive
- wrong number/kind of operands
- undefined or duplicate label
- jump offset out of `i16` range
- `LOAD`/`STORE` slot `>= .globals` value
- `@idx` constant index out of pool range

## 7. Examples

### Countdown

Prints `0` through `4`, one per line:

```text
; countdown.sms — print 0..4
.globals 1                 ; slot 0 = i

        CONSTANT 0
        STORE 0            ; i = 0
loop:
        LOAD 0
        CONSTANT 5
        LT                 ; i < 5 ?
        JUMPIFFALSE :done
        LOAD 0
        PRINT
        LOAD 0
        CONSTANT 1
        ADD
        STORE 0            ; i = i + 1
        JUMP :loop
done:
        HALT
```

### Factorial (functions, locals, recursion)

Prints `120`:

```text
; factorial.sms — factorial(5), recursive
; frame locals of factorial: [0] = n

        JUMP :main         ; skip the function body

factorial:
        LOADLOCAL 0        ; n
        CONSTANT 1
        LE                 ; n <= 1 ?
        JUMPIFFALSE :recurse
        CONSTANT 1
        RETURN             ; base case: return 1

recurse:
        LOADLOCAL 0
        CONSTANT 1
        SUB                ; -- (n-1)
        CALL :factorial, 1 ; -- (n-1)!
        LOADLOCAL 0        ; -- (n-1)! n
        MUL                ; -- n!
        RETURN

main:
        CONSTANT 5
        CALL :factorial, 1
        PRINT              ; 120
        HALT
```

Note the entry convention: execution starts at address 0, so the program
jumps over function bodies to its top-level code. (`CALL` targets are
absolute addresses; forward references are fine.)

## 8. Disassembler (`-dump`)

`glox -dump file.sms` (or `-dump file.glox`) prints the disassembly and
exits. Format:

```text
constants:
  0000: 3
  0001: 4
globals: 1
code:
  0000  CONSTANT 0000    ; 3        ( -- const )
  0003  CONSTANT 0001    ; 4        ( -- const )
  0006  ADD                         ( a b -- a+b )
  0007  HALT
```

- Addresses are 4-digit lowercase hex.
- Operands print as 4-digit hex for `u16`/`i16` (with a `-` sign for
  negative offsets), decimal for slot counts.
- The comment column shows, when applicable: the constant's value for
  `CONSTANT`; the absolute target address for `JUMP`/`JUMPIFFALSE`;
  the callee address for `CALL`; and the opcode's stack effect.
- Output is valid SMS input (the comment column is comments), so
  round-tripping works.

## 9. Tracer (`-trace`)

`glox -trace file.sms` prints one line per executed instruction:

```text
0000  CONSTANT 0000    stack: [3]
0003  CONSTANT 0001    stack: [3 4]
0006  ADD              stack: [7]
0007  HALT             stack: [7]
result: 7
```

- Each line shows the instruction's address, its disassembled form, and the
  value stack **after** the instruction executes.
- Stack prints bottom-first, top-last (rightmost = top), in canonical text
  form; floats, bools, nil, and strings appear once those kinds exist.
- On `HALT`, a final `result:` line prints the top of stack (or `result:
  (empty)`).
- Runtime errors print the failing instruction then the error.

## 10. CLI summary

```text
glox file.glox            compile + run
glox file.sms             assemble + run
glox -dump FILE           disassemble (bytecode source or SMS) and exit
glox -trace FILE          run with per-instruction tracing
glox -gc-stats FILE       run with GC statistics (Phase 10, optional)
glox                      REPL (Phase 11, stretch)
```

Exit status: 0 on success, 1 on compile/assembly error or runtime error
(message on stderr).
