# F11 Language Specification
## Draft 0.2 — Bootstrap Core

**Source:** `.f11` · **Compiler:** `f11c` · **Core calculus:** `JK∨` · **Backend:** native AOT, LLVM first

> A process should not automatically inherit the universe. It should receive a world.

## 1. Purpose
F11 is a native systems language for bounded-world computation. A process receives an operative world, binds relevance inside it, transforms state under that binding, evaluates invariants, and escapes when the world is no longer sufficient.

```text
F11 → K → J → N → cond
                  └─∨ esc
```

`F11` totalizes operative context. `K` binds relevance. `J` marks transformation. `N` binds invariant evidence. A `cond` continues or short-circuits through `∨ esc`. `esc` rejects the sufficiency of the current world and enters its `recov` path.

> **No transformation may silently acquire a larger world than the one under which it was justified.**

## 2. Canonical Form
```f11
func parsMod (src: &source, arena: !arena)
    -> result<modID, parsErr>
=
F11 pars ⟪ src, arena ⟫ -> result<modID, parsErr> ⟨
    K stream ≡ tok(src);
    J ast ≡ parsItems(stream, arena);
    N bound ≡ ast.end ≤ src.len;
    N whole ≡ ast.mod ≠ ∅;
    bound ∧ whole ∨ esc .pars;
    ok(ast.mod)
⟩
esc e ⟨ err(e) ⟩
```

`=` defines. `≡` binds. `←` mutates. `⟪…⟫` encloses admitted context. `⟨…⟩` encloses execution. Blocks yield their final expression.

## 3. Core Notation
| Form | Meaning |
|---|---|
| `func f (…) -> T = e` | function definition |
| `x ≡ e` | immutable binding |
| `x ← e` | mutation |
| `F11 w ⟪…⟫ -> T ⟨…⟩` | bounded operative world |
| `K x ≡ e` | relevance binding |
| `J x ≡ e` | transformational binding |
| `N x ≡ e` | invariant evidence; `e : bool` |
| `a ∧ b`, `a ∨ b` | short-circuit boolean ops |
| `esc e` | escape current world |
| `.name` | tag literal |
| `&T` | shared borrow |
| `!T` | unique writable borrow |
| `*T` | raw pointer; unsafe |
| `∅` | empty / none where admitted by type |

Canonical source is UTF-8. F11/0 must also accept ASCII aliases: `<< >>` for `⟪ ⟫`, `<{ }>` for `⟨ ⟩`, `:=` for `≡`, `<-` for `←`, `&&` for `∧`, `||` for `∨`, `<=` for `≤`, `!=` for `≠`.

## 4. Worlds
`F11` creates a lexically closed operative world. Code inside it may use only admitted names, names created inside it, compile-time types/constants, and authority reachable through admitted capabilities.

```f11
secret ≡ loadSecret(io);
F11 audit ⟪ log ⟫ -> bool ⟨
    secret ≠ ∅     // error: not admitted
⟩
```

Admission follows type. `T` is owned and moves unless `T : copy`; `&T` is shared; `!T` is unique writable. Capability values carry external authority. Worlds do not restate those modes.

```f11
F11 outer ⟪ net, state ⟫ -> T ⟨
    F11 inner ⟪ state ⟫ -> U ⟨
        net.send(x)    // error: inner did not admit net
    ⟩
⟩
```

A world does not claim to contain every fact. It declares its admitted context sufficient for the current computation. When that declaration fails, `esc` or construct another world.

## 5. K — Relevance
`K` binds a value as relevant inside the current world.

```f11
K stream ≡ tok(src);
K dead ≡ insn | redundant(_, dom);
```

A `K` binding carries world provenance and cannot outlive its world by reference. `K` is not magical attention: the expression may be a projection, predicate, mask, slice, index set, graph neighborhood, query, model output, or any typed selector.

Conceptually: `K : (world W, T) → focus<T,W>`. The compiler may lower a focus to an ordinary value, slice, bitmask, index set, pointer range, SIMD mask, or another native representation.

## 6. J — Transformation
`J` marks a state transition whose dependencies are explicit in the current world.

```f11
J ast ≡ parsItems(stream, arena);
J next ≡ rewrite(block, dead);
```

`J` does not itself imply mutation. Mutation remains explicit through writable values and `←`. `J` records transformational meaning for checks, diagnostics, JK∨ IR, and audit tooling.

```f11
J func rewrite (b: !block, k: &cand) -> block = ⟨ ... ⟩
```

A `J` expression may use only values visible in its world and effects reachable through admitted capabilities.

## 7. N — Invariant Evidence
`N` binds boolean evidence about a state or transformation.

```f11
N bound ≡ ast.end ≤ src.len;
N whole ≡ ast.mod ≠ ∅;
N safe ≡ policy.permits(next);
```

`N name ≡ e` requires `e : bool`. If the compiler proves `e`, the runtime check may vanish. Otherwise it evaluates normally. Names should stay short because `N` already carries the category: `bound`, `whole`, `safe`, `typed`, `live`, `same`.

Named refinement/shell types are deferred beyond F11/0.

## 8. Cond, ∨, esc, recov
A **cond** is any boolean expression governing continuation.

```f11
bound ∧ whole ∨ esc .pars;
```

Because `∨` short-circuits and `esc : never`, this means continue when the cond holds, otherwise escape. `esc` is not `break`, `return`, an ordinary error, or panic; it means the current world is insufficient for justified continuation.

```f11
F11 w ⟪ ctx ⟫ -> result<T,E> ⟨
    ...
    cond ∨ esc e;
    ok(v)
⟩
esc e ⟨ err(e) ⟩
```

The world body and recov arm must type to the declared result type. Nested escape may target a named world: `esc outer .reframe;`. On escape, owned values in the exited world are dropped, its region is released, then control enters recov.

## 9. Bindings and Mutation
```f11
x ≡ 4;
x: i64 ≡ 4;
var i: usize ≡ 0;
i ← i + 1;
```

Mutation requires a writable local, unique borrow, or explicit mutable capability. There is no implicit global mutable state.

## 10. Types
Bootstrap scalars:

```text
bool
u8 u16 u32 u64 usize
i8 i16 i32 i64 isize
f32 f64
char unit never
```

Core compound forms:

```text
[T; n]       fixed array
[]T          slice
(T,U,…)      tuple
struct       product
enum         sum
func(…) -> T function
result<T,E>
opt<T>
&T  !T  *T
```

Generics are monomorphized in F11/0.

```f11
struct pair<T,U> ⟨ a: T; b: U; ⟩
```

`result<T,E>` has `ok(T)` and `err(E)`. `opt<T>` has `some(T)` and `none`.

## 11. Ownership and Memory
F11 requires no tracing GC. Owned non-`copy` values move. `&T` may alias but cannot mutate `T`; `!T` is unique for its lifetime and may mutate `T`; borrows cannot outlive their referent; dereferencing `*T` requires `unsafe`.

Each `F11` world may own a region. Region allocations die on normal completion or `esc`, and references into that region cannot escape it.

```f11
node ≡ region.alloc<node>();
```

General heap allocation is provided by an admitted allocator capability, not ambient authority.

## 12. Capabilities
External authority is represented by typed values whose methods perform effects. Hosted core capabilities include `fileR<P>`, `fileW<P>`, `proc`, `clock`, `entropy`, `net`, and `alloc`. Importing a module grants no authority.

```f11
func emit (obj: &bytes, out: fileW<path>) -> ioRes = out.write(obj);
```

A function with no capability, writable borrow, unsafe op, clock, or entropy input is deterministic with respect to its arguments.

## 13. Control
F11/0 requires:

```f11
if cond ⟨ a ⟩ else ⟨ b ⟩
unless cond ⟨ body ⟩
while cond ⟨ body ⟩
for x in xs ⟨ body ⟩
match x ⟨ pat => expr; ... ⟩
break;
continue;
return expr;
esc expr;
```

`unless` remains ordinary negative conditional syntax; canonical world guards use `cond ∨ esc e`. Blocks are expressions and yield their final expression unless control transfers.

## 14. Data and Match
```f11
enum expr ⟨
    lit(i64);
    name(sym);
    add(exprID, exprID);
⟩

func eval (e: &expr) -> i64 =
match e ⟨
    .lit(n)   => n;
    .add(a,b) => eval(a) + eval(b);
    .name(_)  => 0;
⟩;
```

`match` must be exhaustive unless `_` is present.

## 15. Modules, Unsafe, FFI
```f11
mod pars;
use lex.tok;
use ir.{block, value};
```

Modules are namespaces/compilation units, not authority boundaries.

```f11
unsafe ⟨ x ≡ ptr.*; ⟩
extern "C" func write (fd: i32, p: *u8, n: usize) -> isize;
```

F11/0 supports the platform C ABI. Foreign calls are unsafe unless wrapped by a typed capability. `@repr(C)` fixes interop layout.

## 16. Native Compilation
```text
.f11
 ↓ lex / pars
AST
 ↓ name + type
world + cap + borrow
 ↓
JK∨ IR
 ↓
SSA
 ↓
LLVM IR
 ↓
object → link → native executable
```

JK∨ IR preserves at least `WORLD.IN`, `WORLD.OUT`, `WORLD.ESC`, `K.BIND`, `J.APPLY`, `N.CHECK`, `CAP.USE`, `REGION.NEW`, `REGION.DROP`. These are semantic IR ops, not runtime calls; lowering may erase them.

Initial targets: `x86_64-linux`, `aarch64-linux`, `aarch64-android`. The first compiler may emit textual LLVM IR and invoke LLVM/Clang through an admitted `proc` capability. LLVM is a backend, not an F11 runtime. Freestanding F11 may compile with no language runtime beyond platform startup and explicitly linked facilities.

## 17. F11/0 Bootstrap Subset
F11/0 contains only what is needed to implement `f11c`: scalars; arrays/slices; struct/enum; result/opt; monomorphized generics; funcs/recursion; immutable binding plus explicit mutation; if/unless/while/for/match; modules; ownership plus `&`/`!`; world regions; capabilities; `F11/K/J/N/esc`; unsafe/C ABI; textual LLVM IR emission.

It excludes threads, async, closures, traits, overloadable operators, macros, reflection, exceptions, GC, shell/refinement types, SMT proving, package management, and a direct machine-code backend. Exclusion means “not required to bootstrap,” not “promised later.”

## 18. Bootstrap
Stage 0 is a disposable host-language compiler whose only job is to compile the first self-hosted compiler. The current seed is Go.

```text
f11c₀ (Go)
   ↓
f11c₁ (F11)
   ↓ self-compile
f11c₂ (F11)
```

Bootstrap succeeds when `f11c₁` and `f11c₂` agree on normalized semantic output for the compiler source. Byte-identical native binaries are desirable but not initially required because backend/linker metadata may vary. No Stage-0-only semantic feature may remain necessary after Stage 1.

The self-hosted compiler remains conventional beneath F11 semantics:

```text
source → lex → pars → AST → resolve → type → world/cap/borrow → JK∨ IR → SSA → LLVM text
```

```f11
func optBlock (b: !block, dom: &domInfo)
    -> result<block,optErr>
=
F11 opt ⟪ b, dom ⟫ -> result<block,optErr> ⟨
    K dead ≡ b.insn | redundant(_, dom);
    J next ≡ elim(b, dead);
    N cfg ≡ validCfg(next);
    N typed ≡ validType(next);
    cfg ∧ typed ∨ esc .rewrite;
    ok(next)
⟩
esc e ⟨ err(e) ⟩
```

## 19. Grammar Spine
```ebnf
prog   = { item } ;
item   = func | struct | enum | mod | use | extern ;
func   = [ "J" ] "func" id "(" params ")" "->" type "=" expr ;
world  = "F11" id "⟪" admits "⟫" "->" type block [ "esc" id block ] ;
block  = "⟨" { stmt } [ expr ] "⟩" ;
stmt   = bind ";" | kbind ";" | jbind ";" | nbind ";" | assign ";" | control ;
bind   = id [ ":" type ] "≡" expr ;
kbind  = "K" id "≡" expr ;
jbind  = "J" id "≡" expr ;
nbind  = "N" id "≡" expr ;
assign = place "←" expr ;
escape = "esc" [ id ] [ expr ] ;
```

This fixes the bootstrap spine only. Full precedence, patterns, attributes, type grammar, and module paths belong in the implementation grammar.

## 20. Naming Pressure
Where syntax already supplies category, use the shortest clear root: `pars` not `parser/parsing`; `recov` not `recovery`; `cond` not `condition`; `bound` not `bounded`; `whole` not `completeness`; `tok` not `tokenStream`; `cfg` not `controlFlowGraph`; `cap` not `capability` when context is clear. This is style, not a lexer rule.

## 21. Diagnostics
Diagnostics should speak F11 semantics, not collapse violations into generic scope errors.

```text
E041  J `next` depends on `policy` outside F11 `decide`.

      next
        ↑ J
      evidence
        ↑ K
      decide ⟪ state, obs ⟫

      `policy` is not admitted.
      Admit it, derive it from admitted context, or esc.
```

World provenance remains compiler-visible even when erased from machine code.

## 22. Non-goals
F11 is not a prompt language, agent framework, transformer DSL, theorem prover, VM, or universal replacement for C/Rust/OCaml/Python. It is for software where hidden context, ambient authority, unchecked transformation, or unstated adequacy assumptions are themselves part of correctness.

## 23. Semantic Summary
```text
F11 W ⟪Γ⟫
    totalize Γ as the operative world
K x ≡ e
    bind e as relevant in W
J y ≡ f(x,…)
    bind the result of an explicit transformation in W
N n ≡ p(y)
    bind invariant evidence
n ∨ esc r
    continue if the cond holds; otherwise reject W as sufficient
```

A simulated world is a bounded execution world that admits its heuristic. F11 makes that admission executable.
