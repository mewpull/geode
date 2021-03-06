package ast

import (
	"github.com/geode-lang/geode/llvm/ir"
	"github.com/geode-lang/geode/llvm/ir/types"
	"github.com/geode-lang/geode/llvm/ir/value"
)

// Callable is for the left side of a function call. It has functions for getting the function that it points to, etc...
type Callable interface {
	GetFunc(*Program, []types.Type) (*ir.Function, []value.Value, error)
}
