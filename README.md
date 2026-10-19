# F11

Experimental native systems language for bounded-world computation.

> A process should not automatically inherit the universe. It should receive a world.

This repository contains the Stage-0 bootstrap compiler. The seed is intentionally small: it exists to lock F11's world semantics and grow only until it can compile the first compiler written in F11.

## Keyboard dialect

Canonical F11 source is ASCII-first and terminal-friendly:

```f11
func main() -> i32 {
    F11 run [] -> i32 {
        K base = 7;
        J out = base + 1;
        N sane = out == 8;
        sane || esc 42;
        out;
    }
    esc e {
        e;
    };
}
```

The mathematical notation remains available as a pretty/document dialect:

```text
&&  <=>  ∧
||  <=>  ∨
<-  <=>  ←
=   <=>  ≡  for bindings
[]  <=>  ⟪⟫ for world admission
{}  <=>  ⟨⟩ for blocks
```

Keywords are case-insensitive. Identifiers are case-sensitive. Both `//` and `#` start line comments.

## Build

```sh
go build -o f11c ./cmd/f11c
```

## Compile native F11

```sh
./f11c examples/hello.f11 -o hello
./hello
printf '%s\n' "$?"   # 8
```

Escape path:

```sh
./f11c examples/escape.f11 -o escape
./escape
printf '%s\n' "$?"   # 42
```

World closure is enforced:

```sh
./f11c --emit check examples/closure_error.f11
# error: name "outside" is not admitted or bound in this world
```

LLVM inspection:

```sh
./f11c --emit llvm -o hello.ll examples/hello.f11
```

## Implemented in the seed

- `func` and expression blocks
- scalar bootstrap `i32`, `i64`, `bool`, `tag`
- function calls
- arithmetic/comparison/boolean expressions
- `F11 name [admitted names] -> T { ... }`
- `K`, `J`, `N`
- `cond || esc payload`
- `esc e { recov }`
- lexical world closure
- native LLVM/Clang AOT output

The bootstrap scalars above are implementation scaffolding, not the final F11 numeric law. The language specification defines the intended `f11 / n11 / z11 / n22 / z22 / ...` families and the F11 ABI.

## Next bootstrap layer

Structs, enums, match, mutation, vectors, arenas, result/opt, byte/string operations, capabilities, and F11 ABI lowering. Stage 0 stops growing when it can compile `f11c.f11`.
