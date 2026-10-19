package f11

type Type struct {
	Name string
	Mode byte // 0 owned, '&' shared, '!' unique, '*' raw
}

func (t Type) String() string {
	if t.Mode != 0 {
		return string(t.Mode) + t.Name
	}
	return t.Name
}

type Program struct{ Funcs []*FuncDecl }

type Param struct {
	Name string
	Ty   Type
	Pos  Pos
}

type FuncDecl struct {
	Name   string
	Params []Param
	Ret    Type
	Body   Expr
	Pos    Pos
}

type Expr interface {
	exprNode()
	GetPos() Pos
}

type IntExpr struct {
	Value string
	Pos   Pos
}

func (*IntExpr) exprNode()      {}
func (e *IntExpr) GetPos() Pos { return e.Pos }

type BoolExpr struct {
	Value bool
	Pos   Pos
}

func (*BoolExpr) exprNode()      {}
func (e *BoolExpr) GetPos() Pos { return e.Pos }

type VarExpr struct {
	Name string
	Pos  Pos
}

func (*VarExpr) exprNode()      {}
func (e *VarExpr) GetPos() Pos { return e.Pos }

type TagExpr struct {
	Name string
	Pos  Pos
}

func (*TagExpr) exprNode()      {}
func (e *TagExpr) GetPos() Pos { return e.Pos }

type CallExpr struct {
	Name string
	Args []Expr
	Pos  Pos
}

func (*CallExpr) exprNode()      {}
func (e *CallExpr) GetPos() Pos { return e.Pos }

type UnaryExpr struct {
	Op  TokKind
	X   Expr
	Pos Pos
}

func (*UnaryExpr) exprNode()      {}
func (e *UnaryExpr) GetPos() Pos { return e.Pos }

type BinaryExpr struct {
	Op   TokKind
	L, R Expr
	Pos  Pos
}

func (*BinaryExpr) exprNode()      {}
func (e *BinaryExpr) GetPos() Pos { return e.Pos }

type EscExpr struct {
	Target string
	Value  Expr
	Pos    Pos
}

func (*EscExpr) exprNode()      {}
func (e *EscExpr) GetPos() Pos { return e.Pos }

type Block struct {
	Stmts []Stmt
	Tail  Expr
	Pos   Pos
}

type WorldExpr struct {
	Name     string
	Admits   []string
	Ret      Type
	Body     *Block
	RecovVar string
	Recov    *Block
	EscTy    Type // filled by sema
	Pos      Pos
}

func (*WorldExpr) exprNode()      {}
func (e *WorldExpr) GetPos() Pos { return e.Pos }

type Stmt interface {
	stmtNode()
	GetPos() Pos
}

type BindStmt struct {
	Role byte
	Name string
	X    Expr
	Pos  Pos
} // 0 ordinary, K/J/N

func (*BindStmt) stmtNode()      {}
func (s *BindStmt) GetPos() Pos { return s.Pos }

type ExprStmt struct {
	X   Expr
	Pos Pos
}

func (*ExprStmt) stmtNode()      {}
func (s *ExprStmt) GetPos() Pos { return s.Pos }
