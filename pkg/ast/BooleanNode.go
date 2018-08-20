package ast

import (
	"github.com/geode-lang/llvm/ir/constant"
	"github.com/geode-lang/llvm/ir/types"
	"github.com/geode-lang/llvm/ir/value"
)

// BooleanNode is an integer literal
type BooleanNode struct {
	NodeType
	TokenReference
	Accessable

	Value string
}

// NameString implements Node.NameString
func (n BooleanNode) NameString() string { return "BooleanNode" }

// InferType implements Node.InferType
func (n BooleanNode) InferType(scope *Scope) string { return "bool" }

// Codegen implements Node.Codegen for BooleanNode
func (n BooleanNode) Codegen(prog *Program) value.Value {
	options := map[string]int64{
		"true":  1,
		"false": 0,
	}
	return constant.NewInt(options[n.Value], types.I1)
}

func (n BooleanNode) String() string {
	return n.Value
}

// GenAccess implements Accessable.GenAccess
func (n BooleanNode) GenAccess(prog *Program) value.Value {
	return n.Codegen(prog)
}
