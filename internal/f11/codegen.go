package f11

import (
	"fmt"
	"hash/fnv"
	"strings"
)

type cgVal struct {
	ty Type
	op string
}
type cgBind struct {
	ty  Type
	op  string
	ptr string
	mut bool
}
type cgEnv struct{ vals map[string]cgBind }

func newCGEnv() *cgEnv { return &cgEnv{vals: map[string]cgBind{}} }
func (e *cgEnv) clone() *cgEnv {
	n := newCGEnv()
	for k, v := range e.vals { n.vals[k] = v }
	return n
}

type blockBuf struct {
	label string
	lines []string
	term  bool
}

type cgFunc struct {
	g      *Codegen
	fn     *FuncDecl
	blocks []*blockBuf
	cur    *blockBuf
	tmp    int
	label  int
	worlds []*cgWorld
}
type cgWorld struct {
	name     string
	escBlock *blockBuf
	escSlot  string
	escTy    Type
}
type Codegen struct {
	funcs   map[string]*FuncDecl
	structs map[string]*StructDecl
}

func EmitLLVM(p *Program) (string, error) {
	g := &Codegen{
		funcs:   map[string]*FuncDecl{},
		structs: map[string]*StructDecl{},
	}
	for _, d := range p.Structs { g.structs[d.Name] = d }
	for _, f := range p.Funcs { g.funcs[f.Name] = f }
	var sb strings.Builder
	sb.WriteString("; F11 bootstrap LLVM IR\nsource_filename = \"f11\"\n\n")
	for _, d := range p.Structs {
		var fs []string
		for _, f := range d.Fields { fs = append(fs, llvmTy(f.Ty)) }
		sb.WriteString(fmt.Sprintf("%%%s = type { %s }\n", d.Name, strings.Join(fs, ", ")))
	}
	if len(p.Structs) != 0 { sb.WriteString("\n") }
	for _, f := range p.Funcs {
		x, err := g.emitFunc(f)
		if err != nil { return "", err }
		sb.WriteString(x)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func llvmTy(t Type) string {
	switch t.Name {
	case "i32", "tag":
		return "i32"
	case "i64":
		return "i64"
	case "bool":
		return "i1"
	case "never":
		return "void"
	}
	return "%" + t.Name
}

func tagConst(name string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%d", h.Sum32()&0x7fffffff)
}

func (g *Codegen) emitFunc(f *FuncDecl) (string, error) {
	c := &cgFunc{g: g, fn: f}
	entry := c.newBlock("entry")
	c.cur = entry
	e := newCGEnv()
	var ps []string
	for _, p := range f.Params {
		op := "%" + p.Name
		ps = append(ps, fmt.Sprintf("%s %s", llvmTy(p.Ty), op))
		e.vals[p.Name] = cgBind{ty: p.Ty, op: op}
	}
	v, err := c.expr(f.Body, e)
	if err != nil { return "", err }
	if !c.cur.term {
		c.emit(fmt.Sprintf("ret %s %s", llvmTy(v.ty), v.op))
		c.cur.term = true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("define %s @%s(%s) {\n", llvmTy(f.Ret), f.Name, strings.Join(ps, ", ")))
	for _, b := range c.blocks {
		sb.WriteString(b.label + ":\n")
		for _, ln := range b.lines { sb.WriteString("  " + ln + "\n") }
	}
	sb.WriteString("}\n")
	return sb.String(), nil
}

func (c *cgFunc) newBlock(prefix string) *blockBuf {
	c.label++
	b := &blockBuf{label: fmt.Sprintf("%s.%d", prefix, c.label)}
	c.blocks = append(c.blocks, b)
	return b
}
func (c *cgFunc) newTmp() string { c.tmp++; return fmt.Sprintf("%%v%d", c.tmp) }
func (c *cgFunc) emit(s string)  { c.cur.lines = append(c.cur.lines, s) }
func (c *cgFunc) br(b *blockBuf) {
	if !c.cur.term {
		c.emit("br label %" + b.label)
		c.cur.term = true
	}
}
func (c *cgFunc) read(b cgBind) cgVal {
	if b.ptr == "" { return cgVal{ty: b.ty, op: b.op} }
	t := c.newTmp()
	c.emit(fmt.Sprintf("%s = load %s, ptr %s", t, llvmTy(b.ty), b.ptr))
	return cgVal{ty: b.ty, op: t}
}

func (c *cgFunc) block(b *Block, e *cgEnv) (cgVal, error) {
	for _, st := range b.Stmts {
		switch q := st.(type) {
		case *BindStmt:
			v, err := c.expr(q.X, e)
			if err != nil { return cgVal{}, err }
			e.vals[q.Name] = cgBind{ty: v.ty, op: v.op}
		case *VarStmt:
			v, err := c.expr(q.X, e)
			if err != nil { return cgVal{}, err }
			ty := v.ty
			if q.Ty.Name != "" { ty = q.Ty }
			ptr := c.newTmp()
			c.emit(fmt.Sprintf("%s = alloca %s", ptr, llvmTy(ty)))
			c.emit(fmt.Sprintf("store %s %s, ptr %s", llvmTy(ty), v.op, ptr))
			e.vals[q.Name] = cgBind{ty: ty, ptr: ptr, mut: true}
		case *AssignStmt:
			b := e.vals[q.Name]
			v, err := c.expr(q.X, e)
			if err != nil { return cgVal{}, err }
			c.emit(fmt.Sprintf("store %s %s, ptr %s", llvmTy(b.ty), v.op, b.ptr))
		case *WhileStmt:
			if err := c.whileStmt(q, e); err != nil { return cgVal{}, err }
		case *ExprStmt:
			if _, err := c.expr(q.X, e); err != nil { return cgVal{}, err }
		}
		if c.cur.term { break }
	}
	if c.cur.term { return cgVal{ty: Type{Name: "never"}}, nil }
	if b.Tail == nil { return cgVal{ty: Type{Name: "i32"}, op: "0"}, nil }
	return c.expr(b.Tail, e)
}

func (c *cgFunc) whileStmt(q *WhileStmt, e *cgEnv) error {
	condB := c.newBlock("while.cond")
	bodyB := c.newBlock("while.body")
	doneB := c.newBlock("while.done")
	c.br(condB)

	c.cur = condB
	cv, err := c.expr(q.Cond, e)
	if err != nil { return err }
	c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", cv.op, bodyB.label, doneB.label))
	c.cur.term = true

	c.cur = bodyB
	if _, err = c.block(q.Body, e.clone()); err != nil { return err }
	if !c.cur.term { c.br(condB) }

	c.cur = doneB
	return nil
}

func (c *cgFunc) expr(x Expr, e *cgEnv) (cgVal, error) {
	switch q := x.(type) {
	case *BlockExpr:
		return c.block(q.Block, e.clone())
	case *IntExpr:
		return cgVal{ty: Type{Name: "i32"}, op: q.Value}, nil
	case *BoolExpr:
		if q.Value { return cgVal{ty: Type{Name: "bool"}, op: "1"}, nil }
		return cgVal{ty: Type{Name: "bool"}, op: "0"}, nil
	case *TagExpr:
		return cgVal{ty: Type{Name: "tag"}, op: tagConst(q.Name)}, nil
	case *FieldExpr:
		base, err := c.expr(q.Base, e)
		if err != nil { return cgVal{}, err }
		d := c.g.structs[base.ty.Name]
		idx := -1
		var ty Type
		for i, f := range d.Fields {
			if f.Name == q.Name { idx = i; ty = f.Ty; break }
		}
		if idx < 0 { return cgVal{}, fmt.Errorf("codegen: no field %s.%s", d.Name, q.Name) }
		t := c.newTmp()
		c.emit(fmt.Sprintf("%s = extractvalue %s %s, %d", t, llvmTy(base.ty), base.op, idx))
		return cgVal{ty: ty, op: t}, nil
	case *VarExpr:
		b, ok := e.vals[q.Name]
		if !ok { return cgVal{}, fmt.Errorf("codegen: unknown %s", q.Name) }
		return c.read(b), nil
	case *CallExpr:
		if f, ok := c.g.funcs[q.Name]; ok {
			var as []string
			for _, a := range q.Args {
				v, err := c.expr(a, e)
				if err != nil { return cgVal{}, err }
				as = append(as, fmt.Sprintf("%s %s", llvmTy(v.ty), v.op))
			}
			t := c.newTmp()
			c.emit(fmt.Sprintf("%s = call %s @%s(%s)", t, llvmTy(f.Ret), q.Name, strings.Join(as, ", ")))
			return cgVal{ty: f.Ret, op: t}, nil
		}
		if d, ok := c.g.structs[q.Name]; ok {
			cur := "undef"
			sty := Type{Name: d.Name}
			for i, a := range q.Args {
				v, err := c.expr(a, e)
				if err != nil { return cgVal{}, err }
				t := c.newTmp()
				c.emit(fmt.Sprintf("%s = insertvalue %s %s, %s %s, %d", t, llvmTy(sty), cur, llvmTy(v.ty), v.op, i))
				cur = t
			}
			return cgVal{ty: sty, op: cur}, nil
		}
		return cgVal{}, fmt.Errorf("codegen: unknown callable %s", q.Name)
	case *UnaryExpr:
		v, err := c.expr(q.X, e)
		if err != nil { return cgVal{}, err }
		t := c.newTmp()
		if q.Op == TokMinus {
			c.emit(fmt.Sprintf("%s = sub %s 0, %s", t, llvmTy(v.ty), v.op))
			return cgVal{ty: v.ty, op: t}, nil
		}
		c.emit(fmt.Sprintf("%s = xor i1 %s, true", t, v.op))
		return cgVal{ty: Type{Name: "bool"}, op: t}, nil
	case *BinaryExpr:
		if q.Op == TokOr {
			if _, ok := q.R.(*EscExpr); ok { return c.guardOr(q, e) }
			return c.shortBool(q, e, false)
		}
		if q.Op == TokAnd {
			if _, ok := q.R.(*EscExpr); ok { return c.guardAnd(q, e) }
			return c.shortBool(q, e, true)
		}
		lv, err := c.expr(q.L, e)
		if err != nil { return cgVal{}, err }
		rv, err := c.expr(q.R, e)
		if err != nil { return cgVal{}, err }
		t := c.newTmp()
		switch q.Op {
		case TokPlus:
			c.emit(fmt.Sprintf("%s = add %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{ty: lv.ty, op: t}, nil
		case TokMinus:
			c.emit(fmt.Sprintf("%s = sub %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{ty: lv.ty, op: t}, nil
		case TokStar:
			c.emit(fmt.Sprintf("%s = mul %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{ty: lv.ty, op: t}, nil
		case TokSlash:
			c.emit(fmt.Sprintf("%s = sdiv %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{ty: lv.ty, op: t}, nil
		case TokPercent:
			c.emit(fmt.Sprintf("%s = srem %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{ty: lv.ty, op: t}, nil
		case TokLT, TokLE, TokGT, TokGE, TokEqEq, TokNE:
			pred := map[TokKind]string{TokLT: "slt", TokLE: "sle", TokGT: "sgt", TokGE: "sge", TokEqEq: "eq", TokNE: "ne"}[q.Op]
			c.emit(fmt.Sprintf("%s = icmp %s %s %s, %s", t, pred, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{ty: Type{Name: "bool"}, op: t}, nil
		}
	case *MatchExpr:
		return c.matchExpr(q, e)
	case *IfExpr:
		return c.ifExpr(q, e)
	case *EscExpr:
		if len(c.worlds) == 0 { return cgVal{}, fmt.Errorf("codegen: esc outside world") }
		w := c.worlds[len(c.worlds)-1]
		v, err := c.expr(q.Value, e)
		if err != nil { return cgVal{}, err }
		c.emit(fmt.Sprintf("store %s %s, ptr %s", llvmTy(w.escTy), v.op, w.escSlot))
		c.emit("br label %" + w.escBlock.label)
		c.cur.term = true
		return cgVal{ty: Type{Name: "never"}}, nil
	case *WorldExpr:
		return c.world(q, e)
	}
	return cgVal{}, fmt.Errorf("codegen: unsupported expression")
}

func (c *cgFunc) matchExpr(q *MatchExpr, e *cgEnv) (cgVal, error) {
	sv, err := c.expr(q.Value, e)
	if err != nil { return cgVal{}, err }

	mergeB := c.newBlock("match.merge")
	armB := make([]*blockBuf, len(q.Arms))
	for i := range q.Arms { armB[i] = c.newBlock("match.arm") }

	// Emit the decision chain from the current block.
	for i, a := range q.Arms {
		if a.Pat.Kind == '_' {
			c.br(armB[i])
			break
		}
		miss := c.newBlock("match.next")
		var imm string
		switch a.Pat.Kind {
		case 't':
			imm = tagConst(a.Pat.Lit)
		case 'i':
			imm = a.Pat.Lit
		case 'b':
			imm = "0"
			if a.Pat.Lit == "true" { imm = "1" }
		}
		t := c.newTmp()
		c.emit(fmt.Sprintf("%s = icmp eq %s %s, %s", t, llvmTy(sv.ty), sv.op, imm))
		c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", t, armB[i].label, miss.label))
		c.cur.term = true
		c.cur = miss
	}
	if !c.cur.term {
		c.emit("unreachable")
		c.cur.term = true
	}

	var vals []cgVal
	var preds []string
	for i, a := range q.Arms {
		c.cur = armB[i]
		v, err := c.block(a.Body, e.clone())
		if err != nil { return cgVal{}, err }
		if !c.cur.term {
			preds = append(preds, c.cur.label)
			vals = append(vals, v)
			c.br(mergeB)
		}
	}

	c.cur = mergeB
	if len(vals) == 0 {
		c.emit("unreachable")
		c.cur.term = true
		return cgVal{ty: Type{Name: "never"}}, nil
	}
	if len(vals) == 1 { return vals[0], nil }

	t := c.newTmp()
	var ins []string
	for i, v := range vals {
		ins = append(ins, fmt.Sprintf("[ %s, %%%s ]", v.op, preds[i]))
	}
	c.emit(fmt.Sprintf("%s = phi %s %s", t, llvmTy(vals[0].ty), strings.Join(ins, ", ")))
	return cgVal{ty: vals[0].ty, op: t}, nil
}

func (c *cgFunc) ifExpr(q *IfExpr, e *cgEnv) (cgVal, error) {
	cv, err := c.expr(q.Cond, e)
	if err != nil { return cgVal{}, err }
	thenB := c.newBlock("if.then")
	elseB := c.newBlock("if.else")
	mergeB := c.newBlock("if.merge")
	c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", cv.op, thenB.label, elseB.label))
	c.cur.term = true

	c.cur = thenB
	tv, err := c.block(q.Then, e.clone())
	if err != nil { return cgVal{}, err }
	var tp string
	if !c.cur.term { tp = c.cur.label; c.br(mergeB) }

	c.cur = elseB
	ev, err := c.block(q.Else, e.clone())
	if err != nil { return cgVal{}, err }
	var ep string
	if !c.cur.term { ep = c.cur.label; c.br(mergeB) }

	c.cur = mergeB
	if tp != "" && ep != "" {
		t := c.newTmp()
		c.emit(fmt.Sprintf("%s = phi %s [ %s, %%%s ], [ %s, %%%s ]", t, llvmTy(tv.ty), tv.op, tp, ev.op, ep))
		return cgVal{ty: tv.ty, op: t}, nil
	}
	if tp != "" { return tv, nil }
	if ep != "" { return ev, nil }
	c.emit("unreachable")
	c.cur.term = true
	return cgVal{ty: Type{Name: "never"}}, nil
}

func (c *cgFunc) shortBool(q *BinaryExpr, e *cgEnv, and bool) (cgVal, error) {
	lv, err := c.expr(q.L, e)
	if err != nil { return cgVal{}, err }
	lhsPred := c.cur.label
	rhsB := c.newBlock("bool.rhs")
	mergeB := c.newBlock("bool.merge")
	if and {
		c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", lv.op, rhsB.label, mergeB.label))
	} else {
		c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", lv.op, mergeB.label, rhsB.label))
	}
	c.cur.term = true

	c.cur = rhsB
	rv, err := c.expr(q.R, e)
	if err != nil { return cgVal{}, err }
	var rhsPred string
	if !c.cur.term { rhsPred = c.cur.label; c.br(mergeB) }

	c.cur = mergeB
	if rhsPred == "" {
		if and { return cgVal{ty: Type{Name: "bool"}, op: "0"}, nil }
		return cgVal{ty: Type{Name: "bool"}, op: "1"}, nil
	}
	t := c.newTmp()
	skip := "1"
	if and { skip = "0" }
	c.emit(fmt.Sprintf("%s = phi i1 [ %s, %%%s ], [ %s, %%%s ]", t, skip, lhsPred, rv.op, rhsPred))
	return cgVal{ty: Type{Name: "bool"}, op: t}, nil
}

func (c *cgFunc) guardOr(q *BinaryExpr, e *cgEnv) (cgVal, error) {
	lv, err := c.expr(q.L, e)
	if err != nil { return cgVal{}, err }
	okb := c.newBlock("guard.ok")
	failb := c.newBlock("guard.fail")
	c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", lv.op, okb.label, failb.label))
	c.cur.term = true
	c.cur = failb
	if _, err = c.expr(q.R, e); err != nil { return cgVal{}, err }
	c.cur = okb
	return cgVal{ty: Type{Name: "bool"}, op: "1"}, nil
}

func (c *cgFunc) guardAnd(q *BinaryExpr, e *cgEnv) (cgVal, error) {
	lv, err := c.expr(q.L, e)
	if err != nil { return cgVal{}, err }
	failb := c.newBlock("guard.fail")
	okb := c.newBlock("guard.ok")
	c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", lv.op, failb.label, okb.label))
	c.cur.term = true
	c.cur = failb
	if _, err = c.expr(q.R, e); err != nil { return cgVal{}, err }
	c.cur = okb
	return cgVal{ty: Type{Name: "bool"}, op: "0"}, nil
}

func (c *cgFunc) world(w *WorldExpr, outer *cgEnv) (cgVal, error) {
	bodyEnv := newCGEnv()
	for _, n := range w.Admits { bodyEnv.vals[n] = outer.vals[n] }

	escB := c.newBlock("world.esc")
	mergeB := c.newBlock("world.merge")
	var slot string
	if w.Recov != nil {
		slot = c.newTmp()
		c.emit(fmt.Sprintf("%s = alloca %s", slot, llvmTy(w.EscTy)))
	}
	cw := &cgWorld{name: w.Name, escBlock: escB, escSlot: slot, escTy: w.EscTy}
	c.worlds = append(c.worlds, cw)

	bv, err := c.block(w.Body, bodyEnv)
	if err != nil { return cgVal{}, err }
	var bodyPred string
	if !c.cur.term { bodyPred = c.cur.label; c.br(mergeB) }
	c.worlds = c.worlds[:len(c.worlds)-1]

	var rv cgVal
	var recPred string
	if w.Recov != nil {
		c.cur = escB
		load := c.newTmp()
		c.emit(fmt.Sprintf("%s = load %s, ptr %s", load, llvmTy(w.EscTy), slot))
		re := outer.clone()
		re.vals[w.RecovVar] = cgBind{ty: w.EscTy, op: load}
		rv, err = c.block(w.Recov, re)
		if err != nil { return cgVal{}, err }
		if !c.cur.term { recPred = c.cur.label; c.br(mergeB) }
	} else {
		c.cur = escB
		c.emit("unreachable")
		c.cur.term = true
	}

	c.cur = mergeB
	if bodyPred != "" && recPred != "" {
		t := c.newTmp()
		c.emit(fmt.Sprintf("%s = phi %s [ %s, %%%s ], [ %s, %%%s ]", t, llvmTy(w.Ret), bv.op, bodyPred, rv.op, recPred))
		return cgVal{ty: w.Ret, op: t}, nil
	}
	if bodyPred != "" { return bv, nil }
	if recPred != "" { return rv, nil }
	return cgVal{ty: w.Ret, op: "0"}, nil
}
