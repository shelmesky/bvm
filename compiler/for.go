package compiler

import (
	"github.com/shelmesky/bvm/parser"
)

func newNVar(vtype uint32, name string) parser.NVar {
	return parser.NVar{
		Type: &parser.Node{
			Type: parser.TType,
			Value: &parser.NType{
				Type: int64(vtype),
			},
		},
		Name: name,
	}
}

func newGetVar(name string) *parser.Node {
	return &parser.Node{
		Type: parser.TGetVar,
		Value: &parser.NVarValue{
			Name: name,
		},
	}
}

func newSetVar(name string) *parser.Node {
	return &parser.Node{
		Type: parser.TSetVar,
		Value: &parser.NVarValue{
			Name: name,
		},
	}
}

func newCallFunc(name string, param *parser.Node) *parser.Node {
	return &parser.Node{
		Type: parser.TCallFunc,
		Value: &parser.NCallFunc{
			Name: name,
			Params: &parser.Node{
				Type: parser.TParams,
				Value: &parser.NParams{
					Expr: []*parser.Node{param},
				},
			},
		},
	}
}

func newBinary(oper int, left, right *parser.Node) *parser.Node {
	return &parser.Node{
		Type: parser.TBinary,
		Value: &parser.NBinary{
			Oper:  oper,
			Left:  left,
			Right: right,
		},
	}
}

func forCode(node *parser.Node, cmpl *compiler) error {
	var (
		err      error
		keysName string
	)
	nFor := node.Value.(*parser.NFor)
	isKey := true
	curLen := len(cmpl.Contract.Code)
	if err = nodeToCode(nFor.Expr, cmpl); err != nil {
		return err
	}
	if len(nFor.KeyName) == 0 {
		nFor.KeyName = parser.RandName()
		isKey = false
	}
	cmpl.Contract.Code = cmpl.Contract.Code[:curLen]
	maintype, subtype := parseType(nFor.Expr.Result)
	if maintype != parser.VArr && maintype != parser.VMap && maintype != parser.VBytes {
		return cmpl.ErrorParam(nFor.Expr, errForType, Type2Str(nFor.Expr.Result))
	}
	keyType := parser.VInt
	if maintype == parser.VMap {
		keyType = parser.VStr
		keysName = parser.RandName()
	}
	iKey := parser.RandName()
	objName := parser.RandName()
	vars := []parser.NVar{
		newNVar(nFor.Expr.Result, objName),
		newNVar(subtype, nFor.VarName),
		newNVar(uint32(keyType), nFor.KeyName),
		newNVar(parser.VInt, iKey),
	}
	if maintype == parser.VMap {
		vars = append(vars, newNVar((parser.VStr<<4)|parser.VArr, keysName))
	}

	if nFor.Body == nil {
		nFor.Body = &parser.Node{
			Type: parser.TBlock,
			Value: &parser.NBlock{
				Statements: make([]*parser.Node, 0, 10),
			},
		}
	}
	var code, before []*parser.Node

	if maintype == parser.VMap {
		before = []*parser.Node{newBinary(parser.ASSIGN, newSetVar(nFor.KeyName),
			&parser.Node{
				Type: parser.TGetIndex,
				Value: &parser.NGetIndex{
					Name:    keysName,
					Indexes: []*parser.Node{newGetVar(iKey)},
				},
			}),
			newBinary(parser.ASSIGN, newSetVar(nFor.VarName),
				&parser.Node{
					Type: parser.TGetIndex,
					Value: &parser.NGetIndex{
						Name:    objName,
						Indexes: []*parser.Node{newGetVar(nFor.KeyName)},
					},
				}),
		}
	} else {
		before = []*parser.Node{newBinary(parser.ASSIGN, newSetVar(nFor.VarName),
			&parser.Node{
				Type: parser.TGetIndex,
				Value: &parser.NGetIndex{
					Name:    objName,
					Indexes: []*parser.Node{newGetVar(iKey)},
				},
			}),
		}
		if isKey {
			before = append(before, newBinary(parser.ASSIGN, newSetVar(nFor.KeyName), newGetVar(iKey)))
		}
	}
	nFor.Body.Value.(*parser.NBlock).Statements = append(before, nFor.Body.Value.(*parser.NBlock).Statements...)

	nFor.Body.Value.(*parser.NBlock).Statements = append(nFor.Body.Value.(*parser.NBlock).Statements,
		[]*parser.Node{&parser.Node{
			Type: parser.TEndLabel,
		},
			newBinary(parser.ADD_ASSIGN, newSetVar(iKey), &parser.Node{
				Type:   parser.TValue,
				Value:  int64(1),
				Result: parser.VInt,
			})}...)

	initVars := &parser.Node{
		Line:   node.Line,
		Column: node.Column,
		Type:   parser.TVars,
		Value: &parser.NVars{
			Vars: vars,
		},
	}
	if maintype == parser.VMap {
		code = []*parser.Node{initVars,
			newBinary(parser.ASSIGN, newSetVar(objName), nFor.Expr),
			newBinary(parser.ASSIGN, newSetVar(keysName), newCallFunc(`Keys`, newGetVar(objName))),
			&parser.Node{
				Line:   node.Line,
				Column: node.Column,
				Type:   parser.TWhile,
				Value: &parser.NWhile{
					Cond: newBinary(parser.LT, newGetVar(iKey),
						newCallFunc(`Len`, newGetVar(keysName))),
					Body: nFor.Body,
				},
			},
		}
	} else {
		code = []*parser.Node{initVars,
			newBinary(parser.ASSIGN, newSetVar(objName), nFor.Expr),
			&parser.Node{
				Line:   node.Line,
				Column: node.Column,
				Type:   parser.TWhile,
				Value: &parser.NWhile{
					Cond: newBinary(parser.LT, newGetVar(iKey),
						newCallFunc(`Len`, newGetVar(objName))),
					Body: nFor.Body,
				},
			},
		}
	}
	return nodeToCode(&parser.Node{
		Type: parser.TBlock,
		Value: &parser.NBlock{
			Statements: code}}, cmpl)
}

func forInt(node *parser.Node, cmpl *compiler) error {
	var (
		err error
	)
	nFor := node.Value.(*parser.NForInt)
	curLen := len(cmpl.Contract.Code)
	if err = nodeToCode(nFor.From, cmpl); err != nil {
		return err
	}
	if nFor.From.Result != parser.VInt {
		return cmpl.ErrorParam(nFor.From, errIndexInt, Type2Str(nFor.From.Result))
	}
	if err = nodeToCode(nFor.To, cmpl); err != nil {
		return err
	}
	if nFor.To.Result != parser.VInt {
		return cmpl.ErrorParam(nFor.To, errIndexInt, Type2Str(nFor.To.Result))
	}
	cmpl.Contract.Code = cmpl.Contract.Code[:curLen]
	maxName := parser.RandName()
	vars := []parser.NVar{
		newNVar(parser.VInt, nFor.VarName),
		newNVar(parser.VInt, maxName),
	}
	if nFor.Body == nil {
		nFor.Body = &parser.Node{
			Type: parser.TBlock,
			Value: &parser.NBlock{
				Statements: make([]*parser.Node, 0, 10),
			},
		}
	}
	var code []*parser.Node

	nFor.Body.Value.(*parser.NBlock).Statements = append(nFor.Body.Value.(*parser.NBlock).Statements,
		[]*parser.Node{&parser.Node{
			Type: parser.TEndLabel,
		},
			newBinary(parser.ADD_ASSIGN, newSetVar(nFor.VarName), &parser.Node{
				Type:   parser.TValue,
				Value:  int64(1),
				Result: parser.VInt,
			})}...)

	initVars := &parser.Node{
		Line:   node.Line,
		Column: node.Column,
		Type:   parser.TVars,
		Value: &parser.NVars{
			Vars: vars,
		},
	}
	code = []*parser.Node{initVars,
		newBinary(parser.ASSIGN, newSetVar(nFor.VarName), nFor.From),
		newBinary(parser.ASSIGN, newSetVar(maxName), nFor.To),
		&parser.Node{
			Line:   node.Line,
			Column: node.Column,
			Type:   parser.TWhile,
			Value: &parser.NWhile{
				Cond: newBinary(parser.LTE, newGetVar(nFor.VarName), newGetVar(maxName)),
				Body: nFor.Body,
			},
		},
	}
	return nodeToCode(&parser.Node{
		Type: parser.TBlock,
		Value: &parser.NBlock{
			Statements: code}}, cmpl)
}
