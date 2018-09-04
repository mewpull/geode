package ast

import (
	"fmt"
	"os"

	"github.com/geode-lang/llvm/ir"
	"github.com/geode-lang/llvm/ir/constant"
	"github.com/geode-lang/llvm/ir/types"
	"github.com/geode-lang/llvm/ir/value"
)

// A global number to indicate which `name index` we are on. This way,
// the mangler will never output the same name twice as this number is monotonic
var nameNumber int

func mangleName(name string) string {
	nameNumber++
	return fmt.Sprintf("%s_%d", name, nameNumber)
}

func branchIfNoTerminator(blk *ir.BasicBlock, to *ir.BasicBlock) {
	if blk.Term == nil {
		blk.NewBr(to)
	}
}

// Codegen returns some NamespaceNode's arguments
func (n NamespaceNode) Codegen(prog *Program) (value.Value, error) { return nil, nil }

// Handle will do ast-level handling for a dependency node
func (n DependencyNode) Handle(prog *Program) value.Value {

	return nil
}

// Codegen implements Node.Codegen for DependencyNode
func (n DependencyNode) Codegen(prog *Program) (value.Value, error) { return nil, nil }

// Codegen implements Node.Codegen for IfNode
func (n IfNode) Codegen(prog *Program) (value.Value, error) {

	predicate, err := n.If.Codegen(prog)
	if err != nil {
		return nil, err
	}
	zero := constant.NewInt(0, types.I32)
	// The name of the blocks is prefixed because
	namePrefix := fmt.Sprintf("if.%d.", n.Index)
	parentBlock := prog.Compiler.CurrentBlock()
	c, err := createTypeCast(prog, predicate, types.I32)
	if err != nil {
		return nil, err
	}
	predicate = parentBlock.NewICmp(ir.IntNE, zero, c)
	parentFunc := parentBlock.Parent

	var thenGenBlk *ir.BasicBlock
	var endBlk *ir.BasicBlock

	thenBlk := parentFunc.NewBlock(mangleName(namePrefix + "then"))

	prog.Compiler.genInBlock(thenBlk, func() error {
		gen, gerr := n.Then.Codegen(prog)
		if gerr != nil {
			return gerr
		}
		thenGenBlk = gen.(*ir.BasicBlock)
		return nil
	})

	elseBlk := parentFunc.NewBlock(mangleName(namePrefix + "else"))
	var elseGenBlk *ir.BasicBlock

	prog.Compiler.genInBlock(elseBlk, func() error {
		// We only want to construct the else block if there is one.
		if n.Else != nil {
			gen, gerr := n.Else.Codegen(prog)
			if gerr != nil {
				return gerr
			}
			elseGenBlk, _ = gen.(*ir.BasicBlock)
		}
		return nil
	})

	endBlk = parentFunc.NewBlock(mangleName(namePrefix + "end"))
	prog.Compiler.PushBlock(endBlk)
	// We need to make sure these blocks have terminators.
	// in order to do that, we branch to the end block
	branchIfNoTerminator(thenBlk, endBlk)
	branchIfNoTerminator(thenGenBlk, endBlk)
	branchIfNoTerminator(elseBlk, endBlk)

	if elseGenBlk != nil {
		branchIfNoTerminator(elseGenBlk, endBlk)
	}

	parentBlock.NewCondBr(predicate, thenBlk, elseBlk)

	return endBlk, nil
}

// Codegen implements Node.Codegen for ForNode
func (n ForNode) Codegen(prog *Program) (value.Value, error) {

	// The name of the blocks is prefixed so we can determine which for loop a block is for.
	namePrefix := fmt.Sprintf("for.%X.", n.Index)
	parentBlock := prog.Compiler.CurrentBlock()
	prog.Scope = prog.Scope.SpawnChild()
	var err error
	var predicate value.Value
	var condBlk *ir.BasicBlock
	var bodyBlk *ir.BasicBlock
	var bodyGenBlk *ir.BasicBlock
	var endBlk *ir.BasicBlock
	parentFunc := parentBlock.Parent

	condBlk = parentFunc.NewBlock(namePrefix + "cond")

	n.Init.Codegen(prog)

	parentBlock.NewBr(condBlk)

	err = prog.Compiler.genInBlock(condBlk, func() error {
		predicate, _ = n.Cond.Codegen(prog)
		one := constant.NewInt(1, types.I1)

		c, err := createTypeCast(prog, predicate, types.I1)
		if err != nil {
			return err
		}
		predicate = condBlk.NewICmp(ir.IntEQ, one, c)
		return nil
	})
	if err != nil {
		return nil, err
	}
	bodyBlk = parentFunc.NewBlock(namePrefix + "body")
	err = prog.Compiler.genInBlock(bodyBlk, func() error {
		gen, err := n.Body.Codegen(prog)
		if err != nil {
			return err
		}
		bodyGenBlk = gen.(*ir.BasicBlock)
		err = prog.Compiler.genInBlock(bodyGenBlk, func() error {
			_, err := n.Step.Codegen(prog)
			return err
		})
		if err != nil {
			return err
		}
		branchIfNoTerminator(bodyBlk, condBlk)
		branchIfNoTerminator(bodyGenBlk, condBlk)
		return nil
	})
	endBlk = parentFunc.NewBlock(namePrefix + "end")
	prog.Compiler.PushBlock(endBlk)
	condBlk.NewCondBr(predicate, bodyBlk, endBlk)

	prog.Scope = prog.Scope.Parent
	return endBlk, nil
}

// Codegen implements Node.Codegen for CharNode
func (n CharNode) Codegen(prog *Program) (value.Value, error) {
	return constant.NewInt(int64(n.Value), types.I8), nil
}

// GenAccess returns the value from a given CharNode
func (n CharNode) GenAccess(prog *Program) (value.Value, error) {
	return n.Codegen(prog)
}

// Codegen implements Node.Codegen for UnaryNode
func (n UnaryNode) Codegen(prog *Program) (value.Value, error) {

	operandValue, err := n.Operand.Codegen(prog)
	if err != nil {
		return nil, err
	}
	if operandValue == nil {
		n.Operand.SyntaxError()
		return nil, fmt.Errorf("nil operand")
	}

	if n.Operator == "-" {

		if types.IsFloat(operandValue.Type()) {
			return prog.Compiler.CurrentBlock().NewFSub(constant.NewFloat(0, types.Double), operandValue), nil
		} else if types.IsInt(operandValue.Type()) {
			return prog.Compiler.CurrentBlock().NewSub(constant.NewInt(0, types.I64), operandValue), nil
		}
		n.SyntaxError()
		return nil, fmt.Errorf("Unable to make a non integer/float into a negative")

	}

	// handle reference operation
	if n.Operator == "&" {
		return operandValue, nil
	}
	// handle dereference operation
	if n.Operator == "*" {
		if types.IsPointer(operandValue.Type()) {
			return prog.Compiler.CurrentBlock().NewLoad(operandValue), nil
		}
		n.SyntaxError()
		return nil, fmt.Errorf("attempt to dereference a non-pointer variable")
	}

	return operandValue, nil
}

// GenAccess implements Accessable.GenAccess
func (n UnaryNode) GenAccess(prog *Program) (value.Value, error) {
	return n.Codegen(prog)
}

// Codegen implements Node.Codegen for WhileNode
func (n WhileNode) Codegen(prog *Program) (value.Value, error) {

	// The name of the blocks is prefixed because
	namePrefix := fmt.Sprintf("while_%d_", n.Index)
	parentBlock := prog.Compiler.CurrentBlock()

	parentFunc := parentBlock.Parent
	startblock := parentFunc.NewBlock(mangleName(namePrefix + "start"))
	prog.Compiler.PushBlock(startblock)
	predicate, err := n.If.Codegen(prog)
	if err != nil {
		return nil, err
	}
	one := constant.NewInt(1, types.I1)
	prog.Compiler.PopBlock()
	branchIfNoTerminator(parentBlock, startblock)
	c, err := createTypeCast(prog, predicate, types.I1)
	if err != nil {
		return nil, err
	}
	predicate = startblock.NewICmp(ir.IntEQ, one, c)

	var endBlk *ir.BasicBlock

	bodyBlk := parentFunc.NewBlock(mangleName(namePrefix + "body"))
	prog.Compiler.PushBlock(bodyBlk)

	v, err := n.Body.Codegen(prog)
	if err != nil {
		return nil, err
	}
	bodyGenBlk := v.(*ir.BasicBlock)

	// If there is no terminator for the block, IE: no return
	// branch to the merge block

	endBlk = parentFunc.NewBlock(mangleName(namePrefix + "merge"))
	prog.Compiler.PushBlock(endBlk)

	branchIfNoTerminator(bodyBlk, startblock)
	branchIfNoTerminator(bodyGenBlk, startblock)

	startblock.NewCondBr(predicate, bodyBlk, endBlk)

	// branchIfNoTerminator(c.CurrentBlock(), endBlk)

	return endBlk, nil
}

func typeSize(t types.Type) int {
	if types.IsInt(t) {
		return t.(*types.IntType).Size
	}
	if types.IsFloat(t) {
		return int(t.(*types.FloatType).Kind)
	}

	return -1
}

func typesAreLooselyEqual(a, b types.Type) bool {
	return types.IsNumber(a) && types.IsNumber(b)
}

// createTypeCast is where most, if not all, type casting happens in the language.
func createTypeCast(prog *Program, in value.Value, to types.Type) (value.Value, error) {

	inType := in.Type()
	fromInt := types.IsInt(inType)
	fromFloat := types.IsFloat(inType)

	toInt := types.IsInt(to)
	toFloat := types.IsFloat(to)

	inSize := typeSize(inType)
	outSize := typeSize(to)

	if c, ok := in.(*constant.Int); ok && types.IsInt(to) {
		c.Typ = to.(*types.IntType)
		return c, nil
	}

	if c, ok := in.(*constant.Float); ok && types.IsFloat(to) {
		c.Typ = to.(*types.FloatType)
		return c, nil
	}

	if types.Equal(to, types.Void) {
		return nil, nil
	}

	if types.IsPointer(inType) && types.IsPointer(to) {
		return prog.Compiler.CurrentBlock().NewBitCast(in, to), nil
	}

	if fromFloat && toInt {
		return prog.Compiler.CurrentBlock().NewFPToSI(in, to), nil
	}

	if fromInt && toFloat {
		return prog.Compiler.CurrentBlock().NewSIToFP(in, to), nil
	}

	if fromInt && toInt {
		if inSize < outSize {
			return prog.Compiler.CurrentBlock().NewSExt(in, to), nil
		}
		if inSize == outSize {
			return in, nil
		}
		return prog.Compiler.CurrentBlock().NewTrunc(in, to), nil
	}

	if fromFloat && toFloat {
		if inSize < outSize {
			return prog.Compiler.CurrentBlock().NewFPExt(in, to), nil
		}
		if inSize == outSize {
			return in, nil
		}
		return prog.Compiler.CurrentBlock().NewFPTrunc(in, to), nil
	}

	// If the cast would not change the type, just return the in value, nil
	if types.Equal(inType, to) {
		return in, nil
	}

	if types.IsPointer(inType) && types.IsInt(to) {
		return prog.Compiler.CurrentBlock().NewPtrToInt(in, to), nil
	}

	if types.IsInt(inType) && types.IsPointer(to) {
		return prog.Compiler.CurrentBlock().NewIntToPtr(in, to), nil
	}

	return nil, fmt.Errorf("Failed to typecast type %s to %s", inType.String(), to)
}

// Codegen implements Node.Codegen for ReturnNode
func (n ReturnNode) Codegen(prog *Program) (value.Value, error) {

	var retVal value.Value
	var err error

	if prog.Compiler.FN.Sig.Ret != types.Void {
		if n.Value != nil {
			retVal, err = n.Value.Codegen(prog)
			if err != nil {
				return nil, err
			}
			given := retVal.Type()
			expected := prog.Compiler.FN.Sig.Ret
			if !types.Equal(given, expected) {
				if !(types.IsInt(given) && types.IsInt(expected)) {
					n.SyntaxError()
					fnName, err := UnmangleFunctionName(prog.Compiler.FN.Name)
					if err != nil {
						return nil, err
					}
					expectedName, err := prog.Scope.FindTypeName(expected)
					if err != nil {
						return nil, err
					}
					givenName, err := prog.Scope.FindTypeName(given)
					if err != nil {
						return nil, err
					}
					n.SyntaxError()
					return nil, fmt.Errorf("incorrect return value for function %s. =expected: %s (%s). given: %s (%s)", fnName, expectedName, expected, givenName, given)
				}
				retVal, err = createTypeCast(prog, retVal, prog.Compiler.FN.Sig.Ret)
				if err != nil {
					return nil, err
				}
			}
		} else {
			retVal = nil
		}
	}

	prog.Compiler.CurrentBlock().NewRet(retVal)

	return retVal, nil
}

func newCharArray(s string) *constant.Array {
	var bs []constant.Constant
	for i := 0; i < len(s); i++ {
		b := constant.NewInt(int64(s[i]), types.I8)
		bs = append(bs, b)
	}
	bs = append(bs, constant.NewInt(0, types.I8))
	c := constant.NewArray(bs...)
	c.CharArray = true
	return c
}

// CreateEntryBlockAlloca - Create an alloca instruction in the entry block of
// the function.  This is used for mutable variables etc.
func createBlockAlloca(f *ir.Function, elemType types.Type, name string) *ir.InstAlloca {
	// Create a new allocation in the root of the function
	alloca := f.Blocks[0].NewAlloca(elemType)
	// Set the name of the allocation (the variable name)
	// alloca.SetName(name)
	return alloca
}

// Allow functions to return an error isntead of having to manage closing the program each time.
func codegenError(str string, args ...interface{}) value.Value {
	fmt.Fprintf(os.Stderr, "Error: %s\n", fmt.Sprintf(str, args...))
	return nil
}
