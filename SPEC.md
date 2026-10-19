# F11 Language Specification
## Draft 0.3 — Keyboard Core / Bootstrap Law

**Source:** `.f11`  
**Compiler:** `f11c`  
**Core calculus:** `JK∨`  
**Native contract:** F11 ABI  
**Initial backend:** LLVM

> A process should not automatically inherit the universe. It should receive a world.

## 1. Language stance

F11 is a native systems language for bounded-world computation. A computation receives an operative world, binds relevance inside it, transforms under that binding, tests invariants, and escapes when the world is no longer sufficient.

```text
F11 -> K -> J -> N -> cond
                    \-> || esc
```

The source language is designed for terminal natives: DOS users, shell users, game scripters, BBS-era programmers, exploit writers, batch-file survivors, and people who type faster than they decorate.

ASCII is canonical. Unicode is optional pretty-printing.

> **No transformation may silently acquire a larger world than the one under which it was justified.**

## 2. Canonical keyboard form

```f11
func parsMod(src:&source, arena:!arena) -> result<modID,parsErr> {
    F11 pars [src arena] -> result<modID,parsErr> {
        K stream = tok(src);
        J ast = parsItems(stream, arena);

        N bound = ast.end <= src.len;
        N whole = ast.mod != none;

        bound && whole || esc .pars;

        ok(ast.mod);
    }
    esc e {
        err(e);
    };
}
```

The grammar should feel familiar before its semantics become unusual.

## 3. Keyboard law

Canonical source uses:

```text
{}      executable blocks
[]      admitted world
()      calls / grouping / payloads
=       immutable binding
<-      mutation
&&      short-circuit AND
||      short-circuit OR
!       logical NOT / unique type prefix by position
&       shared borrow type prefix
*       raw representation pointer by position
.name   tag
;       statement terminator
#       line comment
//      line comment
```

Pretty/document notation may render:

```text
=   ->  ≡      binding
<-  ->  ←      mutation
&&  ->  ∧
||  ->  ∨
[]  ->  ⟪⟫     world admission
{}  ->  ⟨⟩     execution
<=  ->  ≤
>=  ->  ≥
!=  ->  ≠
```

Pretty notation never carries semantics unavailable in ASCII.

Keywords are case-insensitive:

```f11
func FUNC Func
f11 F11
esc ESC
k K
j J
n N
```

Identifiers remain case-sensitive.

## 4. Short-name pressure

Syntax already carries category, so names should not narrate it twice.

Prefer:

```text
pars    not parser/parsing
recov   not recovery
cond    not condition
bound   not bounded
whole   not completeness
tok     not tokenStream
cfg     not controlFlowGraph
cap     not capability when context is clear
```

This is style, not a lexer restriction.

## 5. Functions and blocks

```f11
func inc(x:n22) -> n22 {
    x + 1;
}
```

Blocks are expressions. Their final expression is their value. A trailing semicolon on the final expression is accepted.

```f11
func max(a:z44, b:z44) -> z44 {
    if a > b {
        a;
    } else {
        b;
    };
}
```

F11/0 may retain the old compact `func f(...) -> T = expr;` form as a bootstrap compatibility form, but it is not canonical.

## 6. Worlds

```f11
F11 name [admitted names] -> T {
    ...
}
esc e {
    recov;
}
```

A world is lexically closed. Inside it, code may use only:

- names admitted in `[]`
- names created inside the world
- compile-time language items
- authority reachable through admitted capability values

Commas in world admission are optional:

```f11
F11 a [src arena fs] -> T { ... }
F11 a [src, arena, fs] -> T { ... }
```

A function called inside a world does not smuggle ambient authority into it. Effective authority is transitive over values and capabilities admitted to the current world.

```f11
secret = load(fs);

F11 audit [log] -> bit {
    J x = inspect(secret);   # error: secret is outside this world
    true;
};
```

Nested worlds do not inherit outer authority:

```f11
F11 outer [net state] -> T {
    F11 inner [state] -> U {
        net.send(x);         # error
    };
};
```

A world does not claim metaphysical completeness. It claims that its admitted context is sufficient for the current computation.

## 7. K / J / N

`K`, `J`, and `N` are semantic bindings, not decorative keywords.

```f11
K toks = lex(src);
J ast = pars(toks, arena);
N whole = ast.mod != none;
```

`K x = e` binds `e` as relevant in the current world.

`J x = e` binds the result of a transformation performed under the current world.

`N x = e` binds invariant evidence and requires `e : bit`.

For bootstrap, K/J/N values use ordinary runtime representation. Their world provenance and role survive into JK∨ IR and may erase during lowering.

## 8. Cond / esc / recov

A **cond** is any `bit` expression governing continuation.

```f11
bound && whole || esc .pars;
```

Because `||` short-circuits and `esc : never`, this means:

```text
continue when bound && whole
otherwise reject the current world as sufficient
```

`esc` is not an exception, panic, return, or break.

```f11
F11 build [src a] -> result<modID,err> {
    ...
    N sane = verify(ir);
    sane || esc .ir;
    ok(mod);
}
esc e {
    err(e);
};
```

An escape:

1. evaluates its payload
2. destroys owned world-local values in reverse construction order
3. releases the world region
4. transfers directly to the world's `recov` arm

No hidden stack exception mechanism is required.

Nested escape targets the nearest world unless named:

```f11
esc .bad;
esc outer .reframe;
```

## 9. Binding and mutation

Immutable binding:

```f11
x = 4;
K toks = lex(src);
```

Explicit mutable local:

```f11
var i:n22 = 0;
i <- i + 1;
```

Repeated `=` does not mean mutation.

Mutation requires a writable local, unique borrow, or explicitly admitted mutable capability.

## 10. Scalar law

F11 does not use C's integer vocabulary as its native numeric ontology.

```text
bit          truth, exactly two values
byte         exactly 8 data bits
f11          element of GF(11), values 0..10

n11 z11
n22 z22
n44 z44
n88 z88

idx          target-native nonnegative index magnitude
off          target-native signed offset
```

`n` denotes natural/bounded unsigned integers.  
`z` denotes signed integers.

The width ladder is semantic:

```text
n11  : 11 value bits
n22  : 22 value bits
n44  : 44 value bits
n88  : 88 value bits
```

Physical carriers may be wider. An `n22` remains an `n22` even if a target uses a 32-bit register.

LLVM may therefore lower directly through `i11`, `i22`, `i44`, and `i88` before target legalization.

### f11

`f11` is genuinely the finite field GF(11):

```f11
a:f11 = 8;
b:f11 = 7;

c = a + b;   # 4
d = a * b;   # 1
```

Its arithmetic is field arithmetic, not machine wraparound.

### integer arithmetic

Default `n*/z*` arithmetic is checked.

Provisional explicit variants:

```f11
a + b     # checked
a +% b    # wrapping
a +^ b    # saturating
```

Exact operator spellings remain bootstrap-freeze material, but silent C-style overflow is forbidden.

## 11. Representation is not value

A semantic width does not dictate packed memory layout.

Example:

```text
n22 value domain    22 bits
F11 ABI carrier     profile-defined, commonly 32 bits
pack<n22>           explicit dense representation
```

Endianness, alignment, aggregate layout, padding, and enum representation are defined by the F11 ABI profile, never inferred from a C compiler.

## 12. Core data needed for self-hosting

The bootstrap wick is:

```text
byte / slice / vec
struct
enum
arena
integer ID
match
while
var + <-
F11 / K / J / N / esc
```

Compiler graphs use arenas and IDs rather than recursive ownership.

```f11
type tokID  = n22;
type exprID = n22;
type typeID = n22;

struct span {
    lo:n44;
    hi:n44;
}

enum expr {
    int(z44);
    bit(bit);
    name(symID);
    call(symID, vec<exprID>);
    binary(op, exprID, exprID);
    world(worldID);
    escape(exprID);
}

struct ast {
    exprs:arena<expr>;
    funcs:arena<func>;
    worlds:arena<world>;
}
```

This intentionally avoids requiring a sophisticated recursive lifetime solver before F11 can compile itself.

## 13. Vec / slice / str

```f11
struct vec<T> {
    ptr:*T;
    len:idx;
    cap:idx;
}

struct str {
    bytes:[]byte;
}
```

UTF-8 source uses byte offsets for spans. Unicode decoding belongs to lexing.

`vec<T>` is part of F11/0 because the compiler needs a growable contiguous collection.

## 14. Ownership and regions

F11 requires no tracing GC.

```text
T     owned value
&T    shared borrow
!T    unique writable borrow
*T    raw representation pointer
```

Rules:

- non-copy owned values move
- `&T` may alias but cannot mutate T
- `!T` is unique for its live extent
- borrows cannot outlive their referent
- raw pointer dereference requires unsafe authority
- each F11 world may own a region
- references into a world region cannot survive world exit

General heap allocation requires an admitted allocator capability.

## 15. Capabilities

External authority is a typed value.

The self-host compiler initially needs only:

```text
alloc
file
proc
```

It does not receive network, clock, entropy, environment, or arbitrary process authority unless explicitly admitted.

Imports grant names, never powers.

## 16. Control required by F11/0

```f11
if cond { a; } else { b; }
unless cond { body; }

while cond {
    ...
}

for x in xs {
    ...
}

match x {
    .a(v) { ... }
    .b(x y) { ... }
    _       { ... }
}

break;
continue;
return expr;
esc expr;
```

Patterns omit commas by default:

```f11
.binary(op l r)
```

Calls retain commas:

```f11
emitBin(op, l, r, ast, out);
```

`match` must be exhaustive unless `_` appears.

## 17. Modules

Modules are namespaces and compilation units, not authority boundaries.

```f11
mod pars;
use lex.tok;
use ir.{block,value};
```

For the first self-host, one large `f11c.f11` is acceptable. Modules may be implemented immediately after the fixed point if omitting them materially simplifies Stage 0.

## 18. F11 ABI

F11 speaks F11 natively.

```text
F11 source
    |
F11 semantic types
    |
F11 ABI
    |
target-profile lowering
    |
machine code
```

There is no C lingua franca in the native path.

The F11 ABI defines:

- canonical symbol identity / mangling
- scalar representation
- aggregate layout
- alignment and padding
- argument/result classes
- ownership transfer across calls
- capability handle representation
- world-bound value restrictions
- escape boundary behavior
- ABI versioning
- target profile legalization

Initial target profiles:

```text
linux-x86_64
linux-aarch64
android-aarch64
```

A profile maps F11 semantic values to legal physical carriers. Example:

```text
n11 -> low 11 semantic bits in legal target carrier
n22 -> low 22 semantic bits in legal target carrier
n44 -> low 44 semantic bits in legal target carrier
n88 -> canonical multi-register / stack representation
f11 -> canonical normalized 0..10 representation
```

The precise physical carrier is profile law, not source-language law.

## 19. Foreign boundaries

Foreign interfaces are canonical translations from F11 ABI directly into the foreign target contract.

```text
F11 ABI <-> SysV64
F11 ABI <-> AAPCS64
F11 ABI <-> JNI
F11 ABI <-> WASM
F11 ABI <-> platform binary interface
```

No C source wrapper is required.

Foreign representation descriptors live only at the boundary and do not become ordinary F11 types.

A foreign C library is treated as a binary interface obeying a platform calling convention plus C layout rules; F11 generates that translation directly.

## 20. Native compilation

```text
.f11
  |
lex / pars
  |
AST
  |
resolve / type
  |
world / cap / borrow
  |
JK∨ IR
  |
SSA
  |
LLVM IR
  |
object
  |
link
  |
native executable
```

JK∨ IR preserves at least:

```text
WORLD.IN
WORLD.OUT
WORLD.ESC
K.BIND
J.APPLY
N.CHECK
CAP.USE
REGION.NEW
REGION.DROP
```

These are semantic IR operations, not mandatory runtime calls.

Stage 0 may invoke Clang as an LLVM/link driver. Clang does not define F11 semantics or ABI. The intended native chain is eventually LLVM object output plus the platform linker/lld.

## 21. Bootstrap

```text
f11c0  Go seed
  |
  | grow only the wick
  v
f11c1  compiler written in F11
  |
  | self-compile
  v
f11c2  compiler produced by F11
```

The parent learns only what the child requires.

Bootstrap succeeds when:

```text
f11c0(f11c.f11) -> f11c1
f11c1(f11c.f11) -> f11c2
normJK(f11c1 output) == normJK(f11c2 output)
```

Native binary identity is desirable but not required initially because linker metadata may vary.

After `f11c1` exists, no Go-only semantic feature may remain necessary.

## 22. Compiler-world discipline

The self-host compiler itself must demonstrate F11's thesis.

Each phase receives a world no larger than required:

```text
lex       [src arena]
pars      [tok arena]
resolve   [ast names arena]
type      [ast types arena]
worldchk  [typed worldinfo arena]
lower     [checked ir arena]
emit      [ir out]
```

No pass receives generic compiler-global authority.

## 23. Diagnostic voice

Diagnostics should be short, spatial, and actionable.

```text
F11:E041 pars: `outside` is outside the world

  F11 pars [src arena]
           ^^^^^^^^^^^

  J ast uses `outside`
             ^^^^^^^

admit it, derive it, or esc
```

The compiler should speak F11 semantics rather than collapsing world violations into generic undefined-variable messages.

## 24. F11/0 stop line

Stage 0 may grow only until `f11c.f11` compiles.

Required before stop:

```text
struct
enum
match
var / <-
while
[]T
vec<T>
arena<T>
indexing
result<T,E>
opt<T>
byte/string ops
alloc/file/proc capabilities
unsafe representation operations
F11 ABI lowering
```

Not required before self-hosting:

```text
threads
async
closures
traits
operator overloading
macros
reflection
exceptions
GC
SMT
shell/refinement types
package management
direct machine-code backend
```

## 25. Semantic summary

```text
F11 W [ctx]
    totalize ctx as the operative world

K x = e
    bind e as relevant in W

J y = f(x)
    bind an explicit transformation in W

N n = p(y)
    bind invariant evidence

n || esc r
    continue if n holds
    otherwise reject W as sufficient
```

A simulated world is a bounded execution world that admits its heuristic.

F11 makes that admission executable.
