# The Glox Language Specification

**Normative.** Glox ("**Go + Lox**") is a small, dynamically typed language
in the Lox lineage, designed as the source language for the Glox VM. It is
deliberately minimal: the floor feature set is arithmetic, `let` bindings,
`if`/`else`, `while`, functions, and `print`. Everything else is an
explicitly marked extension.

Glox compiles to the bytecode defined in [bytecode.md](bytecode.md) and runs
on the machine defined in [vm.md](vm.md). Source files use the `.glox`
extension.

## 1. Design philosophy

- **Small enough to fully understand.** One afternoon should be enough to
  read the whole grammar.
- **Dynamic typing, static scoping.** Types are checked at runtime; names
  are resolved lexically at compile time.
- **No implicit conversions except numeric promotion.** `1 + "a"` is an
  error, not a surprise.
- **Floor features only.** Extensions (§12) exist on paper so the core stays
  honest.

## 2. Lexical structure

Source files are UTF-8 text. Lines are significant only for error reporting;
Glox is otherwise free-form.

### Whitespace and comments

Space, tab, carriage return, and newline separate tokens and are otherwise
ignored. `//` begins a comment that runs to end of line. There are no block
comments in the floor language.

### Identifiers and keywords

```text
identifier  ::=  LETTER ( LETTER | DIGIT | "_" )*
LETTER      ::=  "a"…"z" | "A"…"Z" | "_"
```

Keywords are reserved and may not be used as identifiers:

```text
let  fn  if  else  while  return  true  false  nil  print  and  or
```

### Literals

```text
INTEGER  ::=  DIGIT+                        42, 0, 909
FLOAT    ::=  DIGIT+ "." DIGIT+             3.14, 0.5
STRING   ::=  '"' ( CHAR | ESCAPE )* '"'
ESCAPE   ::=  "\\" | "\"" | "n" | "t" | "r" | "0"    (prefixed by a backslash)
```

- Integer literals must fit in `int64`; otherwise a compile error.
- Float literals require digits on both sides of the dot: `1.` and `.5` are
  errors.
- String escapes: `\\`, `\"`, `\n`, `\t`, `\r`, `\0`. A raw newline inside a
  string literal, or an unterminated string, is a compile error. Strings are
  immutable byte sequences.
- No exponent notation, hex literals, or numeric separators in the floor.

### Operators and punctuation

```text
+  -  *  /  %        arithmetic
== != <  <= >  >=    comparison
=  !                 assignment / logical not
(  )  {  }  ,  ;     grouping, blocks, separators
```

`and` and `or` are word keywords, not symbols.

## 3. Types and values

Glox has exactly five value kinds in the floor language:

| Kind    | Literals          | Notes                                    |
|---------|-------------------|------------------------------------------|
| int     | `42`              | signed 64-bit, wraps on overflow         |
| float   | `3.14`            | IEEE-754 binary64                        |
| bool    | `true`, `false`   |                                          |
| nil     | `nil`             | the absence of a value; a singleton      |
| string  | `"hi"`            | immutable                                |

There are no implicit conversions between kinds **except** numeric
promotion in arithmetic and comparison: if either operand of
`+ - * / % < <= > >=` is a float, the other becomes float and the result is
float (`7 / 2` is `3`, `7.0 / 2` is `3.5`).

### Truthiness

`false` and `nil` are falsy. Everything else — including `0`, `0.0`, and
`""` — is truthy.

### Equality

`==` / `!=` accept any operands. Numbers compare numerically across
int/float; same-kind values compare by value (strings by content);
different kinds are simply unequal. Comparing different kinds is not an
error.

## 4. Expressions

Precedence, lowest to highest (all binary operators are left-associative
except assignment, which is right-associative):

| Level | Operators        | Kind            |
|-------|------------------|-----------------|
| 1     | `=`              | assignment      |
| 2     | `or`             | logical         |
| 3     | `and`            | logical         |
| 4     | `==` `!=`        | equality        |
| 5     | `<` `<=` `>` `>=`| comparison      |
| 6     | `+` `-`          | term            |
| 7     | `*` `/` `%`      | factor          |
| 8     | `!` `-` (prefix) | unary           |
| 9     | `f(args)`        | call            |
| 10    | literals, names, `( )` | primary    |

### Arithmetic operators

`+ - * / %` require numbers, with promotion (§3). Additionally, `+`
accepts two strings, concatenating them (`a + b`, left operand first).
Any other operand kinds are a runtime `ErrTypeMismatch`. Integer division
truncates toward zero; integer division or modulo by zero is a runtime
`ErrDivideByZero`; float division by zero yields infinity/NaN.

### Comparison operators

`< <= > >=` require numbers (promoted); otherwise runtime
`ErrTypeMismatch`. They yield a bool. String ordering is an extension (§12).

### Logical operators

`and` and `or` **short-circuit** and **return one of their operands**:

- `a and b`: evaluate `a`; if falsy, yield `a`; else yield `b` (`b` is not
  evaluated when `a` is falsy).
- `a or b`: evaluate `a`; if truthy, yield `a`; else yield `b`.

`!a` yields a bool: `true` iff `a` is falsy.

```text
print 1 and 2;          // 2    (both truthy → right operand)
print nil or "backup";  // backup
print !0;               // false  (0 is truthy!)
```

### Assignment

`x = expr` stores into the nearest enclosing binding of `x`. The left side
must be a bare identifier — anything else (`1 = x`, `f() = 2`) is a compile
error. Assignment is an expression whose value is the assigned value.

### Calls

`f(a, b)` evaluates `f`'s name, then arguments left to right, then calls.
The callee must be a function declared with `fn`; arity is checked at
compile time. Calling with the wrong number of arguments is a compile
error, not a runtime one.

### Primary expressions

Integer, float, string, `true`, `false`, `nil`, identifiers, and
parenthesized expressions. An identifier that names no binding in scope is
a compile error (§7).

## 5. Statements

Statements are separated (terminated) by `;` where the grammar shows it.
Blocks `{ ... }` group statements and introduce scopes.

### Expression statement

```text
exprStmt ::= expression ";"
```

Any expression followed by `;`. Pure side-effect-free expressions (e.g.
`3 + 4;`) are legal; the compiler may warn, never reject.

### `let` — variable declaration

```text
letDecl ::= "let" IDENTIFIER ( "=" expression )? ";"
```

Declares a variable in the current scope. `let x;` initializes `x` to
`nil`. Redeclaring the same name in the same scope is a compile error.

### `if` / `else`

```text
ifStmt ::= "if" "(" expression ")" statement ( "else" statement )?
```

The condition may be any value; truthiness decides (§3). The parentheses
are required. The body is a single statement — usually a block. `else`
binds to the nearest `if`.

### `while`

```text
whileStmt ::= "while" "(" expression ")" statement
```

Evaluate the condition; if truthy, run the body and repeat. `break`/
`continue` do not exist in the floor language (§12).

### `return`

```text
returnStmt ::= "return" expression? ";"
```

Exits the enclosing function with the expression's value; `return;` and
falling off the end of a function body both yield `nil`. `return` at top
level (outside any function) is a compile error.

### `print`

```text
printStmt ::= "print" expression ";"
```

Evaluates the expression and prints its canonical text form
([vm.md §3](vm.md#3-value-model)) followed by a newline, to stdout.
`print` is a statement keyword in the floor language, not a function.

### Blocks

```text
block ::= "{" declaration* "}"
```

A block is a statement and a scope. Declarations inside it vanish when it
ends.

## 6. Functions

```text
fnDecl    ::= "fn" IDENTIFIER "(" parameters? ")" block
parameters::= IDENTIFIER ( "," IDENTIFIER )*
```

- `fn name(p0, p1) { ... }` declares a function named `name`.
- Parameters bind as locals of the function's frame, in order.
- Calls follow the VM calling convention ([vm.md §7](vm.md#7-calling-convention)):
  arguments are evaluated left to right, the callee receives them as
  locals, and a call yields the callee's return value.
- **Recursion works** (see lowering in §10); mutual recursion between
  top-level functions works because `fn` declarations are hoisted (§7).
- In the floor language: function declarations appear at **top level
  only**, functions are **not values** (they cannot be passed, stored, or
  returned), and **closures do not exist** — a function body may reference
  globals and its own parameters/locals, but referencing an enclosing
  function's locals is a compile error. See §12 for the extension path.

```text
fn add(a, b) { return a + b; }
print add(2, 3);          // 5

fn fib(n) {
  if (n < 2) { return n; }
  return fib(n - 1) + fib(n - 2);
}
print fib(10);            // 55
```

## 7. Declarations, scoping, and names

Glox is **lexically scoped**. Scopes nest: the global (script) scope, then
each function body, then each block within.

- A `let` (or parameter) binds its name in the innermost scope containing
  the declaration. Inner declarations may **shadow** outer names and
  globals.
- **Redeclaration in the same scope** is a compile error.
- Reads and assignments resolve to the **nearest** enclosing binding.
- Using a name that no declaration anywhere in scope introduces is a
  compile error: `undefined variable 'x'`.

Global (top-level) bindings get special treatment:

- All top-level `fn` and `let` declarations are **hoisted** as far as
  *slots* go: the compiler assigns every top-level name a unique global
  slot before generating code, so functions may be called before their
  textual declaration (mutual recursion) and a global may be referenced
  from a function declared above its `let`.
- A `let`'s *initialization* still happens in program order at runtime.
  Reading a global whose `let` statement has not executed yet is a runtime
  error: `ErrUndefinedVariable`. (Implementation note: the VM tracks a
  defined-bit per global slot; slots start undefined, and `let` stores on
  execution.)
- Function declarations also execute in order — the *body* only runs when
  called — so hoisting `fn` is observable only through early calls, which
  see the function fully compiled.

```text
print fib(5);             // legal: fn is hoisted (prints 5)
fn fib(n) { if (n < 2) { return n; } return fib(n-1) + fib(n-2); }

print later;              // runtime error: 'later' used before its let
let later = 1;
```

## 8. Grammar (EBNF)

The complete floor grammar:

```ebnf
program        ::= declaration* EOF ;

declaration    ::= fnDecl | statement ;

fnDecl         ::= "fn" IDENTIFIER "(" parameters? ")" block ;
parameters     ::= IDENTIFIER ( "," IDENTIFIER )* ;

statement      ::= letDecl
                 | ifStmt
                 | whileStmt
                 | returnStmt
                 | printStmt
                 | block
                 | exprStmt ;

letDecl        ::= "let" IDENTIFIER ( "=" expression )? ";" ;
ifStmt         ::= "if" "(" expression ")" statement ( "else" statement )? ;
whileStmt      ::= "while" "(" expression ")" statement ;
returnStmt     ::= "return" expression? ";" ;
printStmt      ::= "print" expression ";" ;
block          ::= "{" declaration* "}" ;
exprStmt       ::= expression ";" ;

expression     ::= assignment ;

assignment     ::= logicOr ( "=" assignment )? ;    // LHS must be IDENTIFIER

logicOr        ::= logicAnd ( "or" logicAnd )* ;
logicAnd       ::= equality ( "and" equality )* ;
equality       ::= comparison ( ( "==" | "!=" ) comparison )* ;
comparison     ::= term ( ( "<" | "<=" | ">" | ">=" ) term )* ;
term           ::= factor ( ( "+" | "-" ) factor )* ;
factor         ::= unary ( ( "*" | "/" | "%" ) unary )* ;
unary          ::= ( "!" | "-" ) unary | call ;
call           ::= primary ( "(" arguments? ")" )* ;
arguments      ::= expression ( "," expression )* ;

primary        ::= INTEGER | FLOAT | STRING
                 | "true" | "false" | "nil"
                 | IDENTIFIER
                 | "(" expression ")" ;
```

Note: `block` and `fnDecl` are both reachable from `statement` /
`declaration` respectively; a block is a statement, a `fn` is a
declaration. `fn` inside a block is a compile error in the floor language.

## 9. Program structure and evaluation

A program is a sequence of top-level declarations executed in order.
Function bodies execute only when called. The top-level code runs in the
script frame; when it finishes, the VM halts. A Glox program has no value —
output happens through `print` (and host-side errors).

## 10. Compilation model

**Informative.** The compiler pipeline is:

```text
source ─▶ lexer ─▶ tokens ─▶ recursive-descent parser ─▶ AST ─▶ compiler ─▶ Chunk
```

Single pass, no optimization passes, no intermediate representation beyond
the AST. Each construct lowers to bytecode per the patterns below
(`L*` are generated labels; addresses are illustrative only).

| Source                   | Lowering                                            |
|--------------------------|-----------------------------------------------------|
| `let x = e;` (global)    | `e; STORE slot(x)`                                  |
| `let x = e;` (local)     | `e; STORELOCAL idx(x)`                              |
| `x = e;`                 | `e; STORE/STORELOCAL` (nearest binding)             |
| `x` (read)               | `LOAD / LOADLOCAL`                                  |
| `if (c) A else B`        | `c; JUMPIFFALSE Lelse; A; JUMP Lend; Lelse: B; Lend:` |
| `if (c) A`               | `c; JUMPIFFALSE Lend; A; Lend:`                     |
| `while (c) B`            | `Lstart: c; JUMPIFFALSE Lend; B; JUMP Lstart; Lend:`|
| `a and b`                | `a; DUP; JUMPIFFALSE Lend; POP; b; Lend:`           |
| `a or b`                 | `a; DUP; JUMPIFFALSE Lb; JUMP Lend; Lb: POP; b; Lend:` |
| `!a`                     | `a; NOT`                                            |
| `f(a1, a2)`              | `a1; a2; CALL addr(f) 2`                            |
| `return e;`              | `e; RETURN`                                         |
| `return;`                | `CONSTANT nil; RETURN`                              |
| end of function body     | `CONSTANT nil; RETURN` (implicit)                   |
| `print e;`               | `e; PRINT`                                          |
| end of top-level code    | `HALT`                                              |

Local slot assignment: parameters first, then each block-scoped `let` gets
a fresh slot in its enclosing function frame (shadowing = distinct slots;
the compiler remaps names, the VM never renames).

## 11. Errors

### Compile-time errors

Reported as `line:col: message` and abort code generation:

- syntax errors (unexpected token, unterminated string, missing `;`, …)
- undefined variable (no binding in scope, §7)
- redeclaration in the same scope
- `return` outside a function
- invalid assignment target
- `fn` declared inside a function or block (floor restriction)
- integer literal out of `int64` range

### Runtime errors

Reported with the failing operation and `pc` where available; the program
stops, no unwinding, no recovery:

- `ErrTypeMismatch` — operand kinds invalid for the operator
- `ErrDivideByZero` — integer `/` or `%` by zero
- `ErrUndefinedVariable` — global read before its `let` executed
- `ErrStackOverflow` / `ErrCallStackOverflow` — limits exceeded (e.g.
  runaway recursion)

Malformed bytecode errors (`ErrTruncatedOperand`, `ErrUnknownOpcode`, …)
are impossible in compiler output; they indicate a compiler or assembler
bug.

## 12. Extensions (not in the floor)

Each is specified only far enough to reserve design space:

- **`for` loops** — `for (init; cond; step) body`, desugaring to
  `{ init; while (cond) { body; step; } }` with `init`'s scope wrapped in a
  block.
- **`break` / `continue`** — compile to jumps to the loop's exit/increment
  labels; need a loop-label stack in the compiler.
- **Closures & nested functions** — require upvalues or heap-allocated
  locals; function bodies could then capture enclosing bindings.
- **First-class functions** — function values in a constant pool or heap,
  `CALLINDIRECT`, runtime `ErrNotCallable` / arity checks.
- **Arrays and maps** — new kinds on the VM heap (would pair naturally with
  the optional Phase 10 object heap).
- **String comparison** — lexicographic `< <= > >=` on strings.
- **`++`/`--`, compound assignment** — sugar for `x = x + 1`-style stores.

## 13. Glox by example

```text
// arithmetic, precedence, promotion
print 2 + 3 * 4;          // 14
print (2 + 3) * 4;        // 20
print 7 / 2;              // 3      integer division
print 7.0 / 2;            // 3.5    promotion
print 7 % 3;              // 1
print -5 - -3;            // -2

// strings
print "hello" + ", " + "glox";   // hello, glox

// truthiness, short-circuiting
print !true;              // false
print 1 and 2;            // 2
print nil or "backup";    // backup
print !0;                 // false   (0 is truthy)

// variables and assignment
let x = 10;
x = x * 2;
print x;                  // 20

// branches
if (x > 15) {
  print "big";
} else {
  print "small";
}                          // big

// loops
let i = 0;
while (i < 3) {
  print i;                 // 0 1 2
  i = i + 1;
}

// functions and recursion
fn add(a, b) {
  return a + b;
}
print add(2, 3);           // 5

fn isEven(n) {
  return (n % 2) == 0;
}
print isEven(4);           // true

fn fib(n) {
  if (n < 2) {
    return n;
  }
  return fib(n - 1) + fib(n - 2);
}
print fib(10);             // 55

// shadowing
let shadow = 1;
{
  let shadow = 2;
  print shadow;            // 2
}
print shadow;              // 1
```
