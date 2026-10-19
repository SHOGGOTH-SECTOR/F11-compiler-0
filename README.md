# F11

Experimental native systems language for bounded-world computation.

> A process should not automatically inherit the universe. It should receive a world.

This repository contains the first executable Stage-0 seed. It parses a deliberately narrow F11 subset, enforces lexical world admission, type-checks `K/J/N` bindings and `esc`, emits textual LLVM IR, and invokes Clang for native code.

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

## Implemented now

- `func`
- scalar `i32`, `i64`, `bool`, `tag`
- function calls
- arithmetic/comparison/boolean expressions
- Unicode syntax and bootstrap ASCII aliases
- `F11 name ⟪ admits ⟫ -> T ⟨ ... ⟩`
- `K`, `J`, `N`
- `cond ∨ esc reason`
- recovery arm `esc e ⟨ ... ⟩`
- lexical world closure
- native LLVM/Clang AOT output

## Deliberately not implemented yet

The compiler is not self-hosting yet. Structs/enums, result/opt, ownership/borrows, mutation, match, modules, capabilities, regions, and the self-hosted compiler source are the next bootstrap layers. The current seed exists to lock the unusual execution semantics before expanding the ordinary systems-language surface.
