package compiler

import (
	"fmt"

	"github.com/shelmesky/bvm/parser"
)

const (
	errOperator          = `Operator %s has not been found`
	errType              = `Unknown type %T`
	errNodeType          = `Unknown node type`
	errVarExists         = `Variable %s has already been defined`
	errFuncExists        = `Function %s has already been defined`
	errVarUnknown        = `Variable %s hasn't been defined`
	errCond              = `Unexpected type %s of expression; expecting bool`
	errJump              = `Too big relative jump`
	errQuestTypes        = `Different types of ?`
	errFuncNotExists     = `Function %s hasn't been defined`
	errFuncLevel         = `Function cannot be defined inside another function`
	errFuncReturn        = `Function must return a value`
	errNotReturn         = `Function cannot return a value`
	errReturnType        = `Function must return %s`
	errData              = `DATA section is too big`
	errContractNotExists = `Contract %s hasn't been found`
	errContractNoParams  = `Contract %s doesn't have parameters`
	errContractNoParam   = `Contract doesn't have %s parameter`
	errParamType         = `Unexpected type of the parameter; expecting %s`
	errInvalidType       = `Invalid type`
	errIndexType         = `Type %s doesn't support indexing`
	errIndexInt          = `Unexpected type %s of expression; expecting int`
	errIndexStr          = `Unexpected type %s of expression; expecting str`
	errForType           = `Unexpected type %s of expression; expecting array, bytes or map`
	errBreak             = `break must be inside of while or for`
	errContinue          = `continue must be inside of while or for`
	errEnv               = `Environment variable $%s is undefined`
	errRetType           = `unsupported type of result value in %s`
	errSwitchType        = `switch doesn't support %s type`
	errCaseType          = `Unexpected type %s of expression; expecting %s`
	errReadContract      = `Calling mutable function or contract from the read contract`
)

func (cmpl *compiler) Error(node *parser.Node, text string) error {
	return fmt.Errorf("%s %d:%d: %s", cmpl.Contract.Name, node.Line, node.Column, text)
}

func (cmpl *compiler) ErrorParam(node *parser.Node, text string, value interface{}) error {
	return cmpl.Error(node, fmt.Sprintf(text, value))
}

func (cmpl *compiler) ErrorTwoParam(node *parser.Node, text string, par1, par2 interface{}) error {
	return cmpl.Error(node, fmt.Sprintf(text, par1, par2))
}

func (cmpl *compiler) ErrorOperator(node *parser.Node) error {
	var (
		oper              int
		name, left, right string
	)
	if node.Type == parser.TUnary {
		nUnary := node.Value.(*parser.NUnary)
		oper = nUnary.Oper
		right = Type2Str(nUnary.Operand.Result)
	} else {
		nBinary := node.Value.(*parser.NBinary)
		oper = nBinary.Oper
		left = Type2Str(nBinary.Left.Result)
		right = Type2Str(nBinary.Right.Result)
	}
	switch oper {
	case parser.SUB:
		name = `-`
	case parser.ADD:
		name = `+`
	case parser.MUL:
		name = `*`
	case parser.DIV:
		name = `/`
	case parser.MOD:
		name = `%`
	case parser.ASSIGN:
		name = `=`
	case parser.ADD_ASSIGN:
		name = `+=`
	case parser.SUB_ASSIGN:
		name = `-=`
	case parser.MUL_ASSIGN:
		name = `*=`
	case parser.DIV_ASSIGN:
		name = `/=`
	case parser.MOD_ASSIGN:
		name = `%=`
	case parser.EQ:
		name = `==`
	case parser.NOT_EQ:
		name = `!=`
	case parser.LT:
		name = `<`
	case parser.LTE:
		name = `<=`
	case parser.GT:
		name = `>`
	case parser.GTE:
		name = `>=`
	case parser.AND:
		name = `&&`
	case parser.OR:
		name = `||`
	}
	return cmpl.Error(node, fmt.Sprintf(errOperator, left+name+right))
}
