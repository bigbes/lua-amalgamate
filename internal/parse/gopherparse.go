package parse

import (
	"bytes"

	"github.com/yuin/gopher-lua/ast"
	"github.com/yuin/gopher-lua/parse"
)

type gopherParser struct{}

func (p *gopherParser) Parse(source []byte, filename string) ([]RequireInfo, error) {
	reader := bytes.NewReader(source)
	chunk, err := parse.Parse(reader, filename)
	if err != nil {
		return nil, err
	}

	walker := &requireWalker{
		requires: []RequireInfo{},
	}
	walkStmts(walker, chunk)
	return walker.requires, nil
}

type requireWalker struct {
	requires []RequireInfo
}

func walkStmts(w *requireWalker, stmts []ast.Stmt) {
	for _, stmt := range stmts {
		walkStmt(w, stmt)
	}
}

func walkStmt(w *requireWalker, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		walkExprs(w, s.Lhs)
		walkExprs(w, s.Rhs)
	case *ast.LocalAssignStmt:
		walkExprs(w, s.Exprs)
	case *ast.FuncCallStmt:
		walkExpr(w, s.Expr)
	case *ast.DoBlockStmt:
		walkStmts(w, s.Stmts)
	case *ast.WhileStmt:
		walkExpr(w, s.Condition)
		walkStmts(w, s.Stmts)
	case *ast.RepeatStmt:
		walkStmts(w, s.Stmts)
		walkExpr(w, s.Condition)
	case *ast.IfStmt:
		walkExpr(w, s.Condition)
		walkStmts(w, s.Then)
		if s.Else != nil {
			walkStmts(w, s.Else)
		}
	case *ast.NumberForStmt:
		walkExpr(w, s.Init)
		walkExpr(w, s.Limit)
		if s.Step != nil {
			walkExpr(w, s.Step)
		}
		walkStmts(w, s.Stmts)
	case *ast.GenericForStmt:
		walkExprs(w, s.Exprs)
		walkStmts(w, s.Stmts)
	case *ast.FuncDefStmt:
		walkStmts(w, s.Func.Stmts)
	case *ast.ReturnStmt:
		walkExprs(w, s.Exprs)
	case *ast.BreakStmt:
		// no children
	}
}

func walkExprs(w *requireWalker, exprs []ast.Expr) {
	for _, expr := range exprs {
		walkExpr(w, expr)
	}
}

func walkExpr(w *requireWalker, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.FuncCallExpr:
		checkRequireCall(w, e)
		walkExpr(w, e.Func)
		walkExprs(w, e.Args)
	case *ast.FunctionExpr:
		walkStmts(w, e.Stmts)
	case *ast.AttrGetExpr:
		walkExpr(w, e.Object)
		walkExpr(w, e.Key)
	case *ast.TableExpr:
		for _, field := range e.Fields {
			if field.Key != nil {
				walkExpr(w, field.Key)
			}
			walkExpr(w, field.Value)
		}
	case *ast.ArithmeticOpExpr:
		walkExpr(w, e.Lhs)
		walkExpr(w, e.Rhs)
	case *ast.RelationalOpExpr:
		walkExpr(w, e.Lhs)
		walkExpr(w, e.Rhs)
	case *ast.LogicalOpExpr:
		walkExpr(w, e.Lhs)
		walkExpr(w, e.Rhs)
	case *ast.StringConcatOpExpr:
		walkExpr(w, e.Lhs)
		walkExpr(w, e.Rhs)
	case *ast.UnaryMinusOpExpr:
		walkExpr(w, e.Expr)
	case *ast.UnaryNotOpExpr:
		walkExpr(w, e.Expr)
	case *ast.UnaryLenOpExpr:
		walkExpr(w, e.Expr)
	case *ast.IdentExpr, *ast.StringExpr, *ast.NumberExpr,
		*ast.TrueExpr, *ast.FalseExpr, *ast.NilExpr, *ast.Comma3Expr:
		// leaf expressions, no children
	default:
		// unknown expression type, ignore
	}
	// Leaf expressions have no children.
}

func checkRequireCall(w *requireWalker, call *ast.FuncCallExpr) {
	// Check for require("...") pattern
	if ident, ok := call.Func.(*ast.IdentExpr); ok && ident.Value == "require" {
		if len(call.Args) == 1 {
			if str, ok := call.Args[0].(*ast.StringExpr); ok {
				w.requires = append(w.requires, RequireInfo{
					Name:   str.Value,
					Line:   call.Line(),
					Static: true,
				})
				return
			}
		}
		// dynamic require
		w.requires = append(w.requires, RequireInfo{
			Name:   "",
			Line:   call.Line(),
			Static: false,
		})
		return
	}

	// Check for pcall(require, "...") pattern
	if ident, ok := call.Func.(*ast.IdentExpr); ok && ident.Value == "pcall" {
		if len(call.Args) >= 2 {
			if reqIdent, ok := call.Args[0].(*ast.IdentExpr); ok && reqIdent.Value == "require" {
				if str, ok := call.Args[1].(*ast.StringExpr); ok {
					w.requires = append(w.requires, RequireInfo{
						Name:   str.Value,
						Line:   call.Line(),
						Static: true,
					})
					return
				}
			}
		}
	}
}
