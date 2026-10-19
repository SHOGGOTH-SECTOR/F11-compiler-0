package f11

import (
	"fmt"
	"strconv"
)

type Sema struct{ funcs map[string]*FuncDecl }

type env struct{ vals map[string]Type }

func newEnv() *env { return &env{vals: map[string]Type{}} }
func (e *env) clone() *env {
	n := newEnv()
	for k, v := range e.vals {
		n.vals[k] = v
	}
	return n
}

type escCtx struct {
	name string
	ty   *Type
	seen bool
}

func Check(p *Program) error {
	s := &Sema{funcs: map[string]*FuncDecl{}}
	for _, f := range p.Funcs {
		if _, ok := s.funcs[f.Name]; ok {
			return fmt.Errorf("%d:%d: duplicate func %s", f.Pos.Line, f.Pos.Col, f.Name)
		}
		if !supportedType(f.Ret) {
			return fmt.Errorf("%d:%d: unsupported return type %s in bootstrap compiler", f.Pos.Line, f.Pos.Col, f.Ret.String())
		}
		for _, q := range f.Params {
			if !supportedType(q.Ty) {
				return fmt.Errorf("%d:%d: unsupported parameter type %s", q.Pos.Line, q.Pos.Col, q.Ty.String())
			}
		}
		s.funcs[f.Name] = f
	}
	for _, f := range p.Funcs {
		e := newEnv()
		for _, q := range f.Params {
			e.vals[q.Name] = q.Ty
		}
		ty, err := s.expr(f.Body, e, nil)
		if err != nil {
			return err
		}
		if !sameBase(ty, f.Ret) {
			return terr(f.Body.GetPos(), "func %s returns %s, declared %s", f.Name, ty.String(), f.Ret.String())
		}
	}
	return nil
}

func supportedType(t Type) bool {
	switch t.Name {
	case "i32", "i64", "bool", "tag":
		return t.Mode == 0
	default:
		return false
	}
}
func sameBase(a, b Type) bool { return a.Name == b.Name && a.Mode == b.Mode }
func terr(p Pos, f string, a ...any) error {
	return fmt.Errorf("%d:%d: %s", p.Line, p.Col, fmt.Sprintf(f, a...))
}

func (s *Sema) block(b *Block, e *env, esc *escCtx) (Type, error) {
	for _, st := range b.Stmts {
		switch q := st.(type) {
		case *BindStmt:
			ty, err := s.expr(q.X, e, esc)
			if err != nil {
				return Type{}, err
			}
			if q.Role == 'N' && ty.Name != "bool" {
				return Type{}, terr(q.Pos, "N %s requires bool, got %s", q.Name, ty.String())
			}
			if _, exists := e.vals[q.Name]; exists {
				return Type{}, terr(q.Pos, "duplicate binding %s", q.Name)
			}
			e.vals[q.Name] = ty
		case *ExprStmt:
			if _, err := s.expr(q.X, e, esc); err != nil {
				return Type{}, err
			}
		}
	}
	if b.Tail == nil {
		return Type{Name: "i32"}, nil
	}
	return s.expr(b.Tail, e, esc)
}

func (s *Sema) expr(x Expr, e *env, esc *escCtx) (Type, error) {
	switch q := x.(type) {
	case *BlockExpr:
		return s.block(q.Block, e.clone(), esc)
	case *IntExpr:
		if _, err := strconv.ParseInt(q.Value, 10, 32); err != nil {
			return Type{}, terr(q.Pos, "integer literal out of i32 range")
		}
		return Type{Name: "i32"}, nil
	case *BoolExpr:
		return Type{Name: "bool"}, nil
	case *TagExpr:
		return Type{Name: "tag"}, nil
	case *VarExpr:
		t, ok := e.vals[q.Name]
		if !ok {
			return Type{}, terr(q.Pos, "name %q is not admitted or bound in this world", q.Name)
		}
		return t, nil
	case *CallExpr:
		f, ok := s.funcs[q.Name]
		if !ok {
			return Type{}, terr(q.Pos, "unknown func %s", q.Name)
		}
		if len(q.Args) != len(f.Params) {
			return Type{}, terr(q.Pos, "%s expects %d args, got %d", q.Name, len(f.Params), len(q.Args))
		}
		for i, a := range q.Args {
			at, err := s.expr(a, e, esc)
			if err != nil {
				return Type{}, err
			}
			if !sameBase(at, f.Params[i].Ty) {
				return Type{}, terr(a.GetPos(), "arg %d to %s is %s, expected %s", i+1, q.Name, at.String(), f.Params[i].Ty.String())
			}
		}
		return f.Ret, nil
	case *UnaryExpr:
		t, err := s.expr(q.X, e, esc)
		if err != nil {
			return Type{}, err
		}
		if q.Op == TokMinus {
			if t.Name != "i32" && t.Name != "i64" {
				return Type{}, terr(q.Pos, "unary - requires integer")
			}
			return t, nil
		}
		if q.Op == TokBang {
			if t.Name != "bool" {
				return Type{}, terr(q.Pos, "! requires bool")
			}
			return Type{Name: "bool"}, nil
		}
	case *BinaryExpr:
		lt, err := s.expr(q.L, e, esc)
		if err != nil {
			return Type{}, err
		}
		if _, ok := q.R.(*EscExpr); ok && (q.Op == TokOr || q.Op == TokAnd) {
			if lt.Name != "bool" {
				return Type{}, terr(q.Pos, "boolean guard requires bool")
			}
			_, err = s.expr(q.R, e, esc)
			if err != nil {
				return Type{}, err
			}
			return Type{Name: "bool"}, nil
		}
		rt, err := s.expr(q.R, e, esc)
		if err != nil {
			return Type{}, err
		}
		switch q.Op {
		case TokPlus, TokMinus, TokStar, TokSlash, TokPercent:
			if !sameBase(lt, rt) || (lt.Name != "i32" && lt.Name != "i64") {
				return Type{}, terr(q.Pos, "arithmetic operands must have same integer type")
			}
			return lt, nil
		case TokLT, TokLE, TokGT, TokGE:
			if !sameBase(lt, rt) || (lt.Name != "i32" && lt.Name != "i64") {
				return Type{}, terr(q.Pos, "comparison operands must have same integer type")
			}
			return Type{Name: "bool"}, nil
		case TokEqEq, TokNE:
			if !sameBase(lt, rt) {
				return Type{}, terr(q.Pos, "equality operands differ: %s and %s", lt.String(), rt.String())
			}
			return Type{Name: "bool"}, nil
		case TokAnd, TokOr:
			if lt.Name != "bool" || rt.Name != "bool" {
				return Type{}, terr(q.Pos, "boolean operator requires bool")
			}
			return Type{Name: "bool"}, nil
		}
	case *EscExpr:
		if esc == nil {
			return Type{}, terr(q.Pos, "esc outside F11 world")
		}
		if q.Target != "" && q.Target != esc.name {
			return Type{}, terr(q.Pos, "targeted esc %s not supported unless it names current world %s", q.Target, esc.name)
		}
		t, err := s.expr(q.Value, e, esc)
		if err != nil {
			return Type{}, err
		}
		if !esc.seen {
			cp := t
			esc.ty = &cp
			esc.seen = true
		} else if !sameBase(*esc.ty, t) {
			return Type{}, terr(q.Pos, "esc payload %s differs from prior %s", t.String(), esc.ty.String())
		}
		return Type{Name: "never"}, nil
	case *WorldExpr:
		we := newEnv()
		for _, name := range q.Admits {
			t, ok := e.vals[name]
			if !ok {
				return Type{}, terr(q.Pos, "F11 %s admits unknown name %s", q.Name, name)
			}
			we.vals[name] = t
		}
		ec := &escCtx{name: q.Name}
		bt, err := s.block(q.Body, we, ec)
		if err != nil {
			return Type{}, err
		}
		if !sameBase(bt, q.Ret) {
			return Type{}, terr(q.Pos, "F11 %s body is %s, declared %s", q.Name, bt.String(), q.Ret.String())
		}
		if ec.seen {
			if q.Recov == nil {
				return Type{}, terr(q.Pos, "F11 %s can esc but has no recov arm", q.Name)
			}
			q.EscTy = *ec.ty
			re := newEnv()
			re.vals[q.RecovVar] = *ec.ty
			rt, err := s.block(q.Recov, re, nil)
			if err != nil {
				return Type{}, err
			}
			if !sameBase(rt, q.Ret) {
				return Type{}, terr(q.Recov.Pos, "recov is %s, expected %s", rt.String(), q.Ret.String())
			}
		} else if q.Recov != nil {
			return Type{}, terr(q.Pos, "recov arm present but F11 %s cannot esc", q.Name)
		}
		return q.Ret, nil
	}
	return Type{}, terr(x.GetPos(), "unsupported expression")
}
