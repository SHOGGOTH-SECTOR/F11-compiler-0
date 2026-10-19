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
type cgEnv struct{ vals map[string]cgVal }

func newCGEnv() *cgEnv { return &cgEnv{vals: map[string]cgVal{}} }
func (e *cgEnv) clone() *cgEnv {
	n := newCGEnv()
	for k, v := range e.vals {
		n.vals[k] = v
	}
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

type Codegen struct{ funcs map[string]*FuncDecl }

func EmitLLVM(p *Program) (string, error) {
	g := &Codegen{funcs: map[string]*FuncDecl{}}
	for _, f := range p.Funcs {
		g.funcs[f.Name] = f
	}
	var sb strings.Builder
	sb.WriteString("; F11 bootstrap LLVM IR\nsource_filename = \"f11\"\n\n")
	for _, f := range p.Funcs {
		x, err := g.emitFunc(f)
		if err != nil {
			return "", err
		}
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
	return "i32"
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
		e.vals[p.Name] = cgVal{p.Ty, op}
	}
	v, err := c.expr(f.Body, e)
	if err != nil {
		return "", err
	}
	if !c.cur.term {
		c.emit(fmt.Sprintf("ret %s %s", llvmTy(v.ty), v.op))
		c.cur.term = true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("define %s @%s(%s) {\n", llvmTy(f.Ret), f.Name, strings.Join(ps, ", ")))
	for _, b := range c.blocks {
		sb.WriteString(b.label + ":\n")
		for _, ln := range b.lines {
			sb.WriteString("  " + ln + "\n")
		}
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

func (c *cgFunc) block(b *Block, e *cgEnv) (cgVal, error) {
	for _, st := range b.Stmts {
		switch q := st.(type) {
		case *BindStmt:
			v, err := c.expr(q.X, e)
			if err != nil {
				return cgVal{}, err
			}
			e.vals[q.Name] = v
		case *ExprStmt:
			_, err := c.expr(q.X, e)
			if err != nil {
				return cgVal{}, err
			}
		}
		if c.cur.term {
			break
		}
	}
	if c.cur.term {
		return cgVal{Type{Name: "never"}, ""}, nil
	}
	if b.Tail == nil {
		return cgVal{Type{Name: "i32"}, "0"}, nil
	}
	return c.expr(b.Tail, e)
}

func (c *cgFunc) expr(x Expr, e *cgEnv) (cgVal, error) {
	switch q := x.(type) {
	case *IntExpr:
		return cgVal{Type{Name: "i32"}, q.Value}, nil
	case *BoolExpr:
		if q.Value {
			return cgVal{Type{Name: "bool"}, "1"}, nil
		}
		return cgVal{Type{Name: "bool"}, "0"}, nil
	case *TagExpr:
		h := fnv.New32a()
		_, _ = h.Write([]byte(q.Name))
		return cgVal{Type{Name: "tag"}, fmt.Sprintf("%d", h.Sum32()&0x7fffffff)}, nil
	case *VarExpr:
		v, ok := e.vals[q.Name]
		if !ok {
			return cgVal{}, fmt.Errorf("codegen: unknown %s", q.Name)
		}
		return v, nil
	case *CallExpr:
		f := c.g.funcs[q.Name]
		var as []string
		for _, a := range q.Args {
			v, err := c.expr(a, e)
			if err != nil {
				return cgVal{}, err
			}
			as = append(as, fmt.Sprintf("%s %s", llvmTy(v.ty), v.op))
		}
		t := c.newTmp()
		c.emit(fmt.Sprintf("%s = call %s @%s(%s)", t, llvmTy(f.Ret), q.Name, strings.Join(as, ", ")))
		return cgVal{f.Ret, t}, nil
	case *UnaryExpr:
		v, err := c.expr(q.X, e)
		if err != nil {
			return cgVal{}, err
		}
		t := c.newTmp()
		if q.Op == TokMinus {
			c.emit(fmt.Sprintf("%s = sub %s 0, %s", t, llvmTy(v.ty), v.op))
			return cgVal{v.ty, t}, nil
		}
		c.emit(fmt.Sprintf("%s = xor i1 %s, true", t, v.op))
		return cgVal{Type{Name: "bool"}, t}, nil
	case *BinaryExpr:
		if q.Op == TokOr {
			if _, ok := q.R.(*EscExpr); ok {
				return c.guardOr(q, e)
			}
		}
		lv, err := c.expr(q.L, e)
		if err != nil {
			return cgVal{}, err
		}
		rv, err := c.expr(q.R, e)
		if err != nil {
			return cgVal{}, err
		}
		t := c.newTmp()
		switch q.Op {
		case TokPlus:
			c.emit(fmt.Sprintf("%s = add %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{lv.ty, t}, nil
		case TokMinus:
			c.emit(fmt.Sprintf("%s = sub %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{lv.ty, t}, nil
		case TokStar:
			c.emit(fmt.Sprintf("%s = mul %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{lv.ty, t}, nil
		case TokSlash:
			c.emit(fmt.Sprintf("%s = sdiv %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{lv.ty, t}, nil
		case TokPercent:
			c.emit(fmt.Sprintf("%s = srem %s %s, %s", t, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{lv.ty, t}, nil
		case TokLT, TokLE, TokGT, TokGE, TokEqEq, TokNE:
			pred := map[TokKind]string{TokLT: "slt", TokLE: "sle", TokGT: "sgt", TokGE: "sge", TokEqEq: "eq", TokNE: "ne"}[q.Op]
			c.emit(fmt.Sprintf("%s = icmp %s %s %s, %s", t, pred, llvmTy(lv.ty), lv.op, rv.op))
			return cgVal{Type{Name: "bool"}, t}, nil
		case TokAnd:
			c.emit(fmt.Sprintf("%s = and i1 %s, %s", t, lv.op, rv.op))
			return cgVal{Type{Name: "bool"}, t}, nil
		case TokOr:
			c.emit(fmt.Sprintf("%s = or i1 %s, %s", t, lv.op, rv.op))
			return cgVal{Type{Name: "bool"}, t}, nil
		}
	case *EscExpr:
		if len(c.worlds) == 0 {
			return cgVal{}, fmt.Errorf("codegen: esc outside world")
		}
		w := c.worlds[len(c.worlds)-1]
		v, err := c.expr(q.Value, e)
		if err != nil {
			return cgVal{}, err
		}
		c.emit(fmt.Sprintf("store %s %s, ptr %s", llvmTy(w.escTy), v.op, w.escSlot))
		c.emit("br label %" + w.escBlock.label)
		c.cur.term = true
		return cgVal{Type{Name: "never"}, ""}, nil
	case *WorldExpr:
		return c.world(q, e)
	}
	return cgVal{}, fmt.Errorf("codegen: unsupported expression")
}

func (c *cgFunc) guardOr(q *BinaryExpr, e *cgEnv) (cgVal, error) {
	lv, err := c.expr(q.L, e)
	if err != nil {
		return cgVal{}, err
	}
	okb := c.newBlock("guard.ok")
	failb := c.newBlock("guard.fail")
	c.emit(fmt.Sprintf("br i1 %s, label %%%s, label %%%s", lv.op, okb.label, failb.label))
	c.cur.term = true
	c.cur = failb
	_, err = c.expr(q.R, e)
	if err != nil {
		return cgVal{}, err
	}
	c.cur = okb
	return cgVal{Type{Name: "bool"}, "1"}, nil
}

func (c *cgFunc) world(w *WorldExpr, outer *cgEnv) (cgVal, error) {
	bodyEnv := newCGEnv()
	for _, n := range w.Admits {
		bodyEnv.vals[n] = outer.vals[n]
	}
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
	if err != nil {
		return cgVal{}, err
	}
	var bodyPred string
	if !c.cur.term {
		bodyPred = c.cur.label
		c.br(mergeB)
	}
	c.worlds = c.worlds[:len(c.worlds)-1]

	var rv cgVal
	var recPred string
	if w.Recov != nil {
		c.cur = escB
		load := c.newTmp()
		c.emit(fmt.Sprintf("%s = load %s, ptr %s", load, llvmTy(w.EscTy), slot))
		re := newCGEnv()
		re.vals[w.RecovVar] = cgVal{w.EscTy, load}
		rv, err = c.block(w.Recov, re)
		if err != nil {
			return cgVal{}, err
		}
		if !c.cur.term {
			recPred = c.cur.label
			c.br(mergeB)
		}
	} else {
		c.cur = escB
		c.emit("unreachable")
		c.cur.term = true
	}
	c.cur = mergeB
	if bodyPred != "" && recPred != "" {
		t := c.newTmp()
		c.emit(fmt.Sprintf("%s = phi %s [ %s, %%%s ], [ %s, %%%s ]", t, llvmTy(w.Ret), bv.op, bodyPred, rv.op, recPred))
		return cgVal{w.Ret, t}, nil
	}
	if bodyPred != "" {
		return bv, nil
	}
	if recPred != "" {
		return rv, nil
	}
	return cgVal{w.Ret, "0"}, nil
}
