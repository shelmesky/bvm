package parser

import (
	"math/rand"
	"time"
)

// Types of Node
const (
	TContract = iota + 1 // contract
	TData                // Data info
	TBlock
	TValue
	TVars
	TBinary
	TUnary
	TSetVar
	TIf
	TElif
	TReturn
	TGetVar
	TWhile
	TQuestion
	TFunc
	TCallFunc
	TParams
	TCallContract
	TContractParams
	TType
	TGetIndex
	TSetIndex
	TFor
	TForInt
	TBreak
	TContinue
	TEndLabel
	TArray
	TMap
	TEnv
	TObject
	TObjArr
	TObjList
	TSwitch
	TCase
)

var (
	nodeTypeMap = map[int]string{
		1:  "TContract",
		2:  "TData",
		3:  "TBlock",
		4:  "TValue",
		5:  "TVars",
		6:  "TBinary",
		7:  "TUnary",
		8:  "TSetVar",
		9:  "TIf",
		10: "TElif",
		11: "TReturn",
		12: "TGetVar",
		13: "TWhile",
		14: "TQuestion",
		15: "TFunc",
		16: "TCallFunc",
		17: "TParams",
		18: "TCallContract",
		19: "TContractParams",
		20: "TType",
		21: "TGetIndex",
		22: "TSetIndex",
		23: "TFor",
		24: "TForInt",
		25: "TBreak",
		26: "TContinue",
		27: "TEndLabel",
		28: "TArray",
		29: "TMap",
		30: "TEnv",
		31: "TObject",
		32: "TObjArr",
		33: "TObjList",
		34: "TSwitch",
		35: "TCase",
	}
)

func GetNodeType(nodeType int) string {
	if v, ok := nodeTypeMap[nodeType]; ok {
		return v
	}

	return "[TYPE NOT FOUND]"
}

const (
	VVoid   = iota
	VInt    // int
	VBool   // bool
	VStr    // str
	VArr    // arr
	VMap    // map
	VFloat  // float
	VMoney  // money
	VObject // object
	VBytes  // bytes
	VFile   // file
	VObjList
)

// NSwitch - switch statement
type NSwitch struct {
	Expr    *Node
	Case    *Node
	Default *Node
}

// NCaseItem - case statement
type NCaseItem struct {
	ExprList *Node
	Body     *Node
}

// NCase - case statement
type NCase struct {
	List []*NCaseItem
}

// NObject contains object data
type NObject struct {
	List []KeyVal
}

// NObjList - init array in object
type NObjArr struct {
	List []*Node
}

// NObjArr - init array in object
type NObjList struct {
	Obj *Node
}

// NVar contains type and name of variable or parameter
type NVar struct {
	Type *Node
	Name string
	Exp  *Node
}

// NType contains the type
type NType struct {
	Type int64
	Def  bool
}

// NEnv contains a name of environment variable
type NEnv struct {
	Name string
}

// NVarValue contains the name of variable
type NVarValue struct {
	Name string
}

// NWhile - while statement
type NWhile struct {
	Cond *Node
	Body *Node
}

// NFor - for statement
type NFor struct {
	VarName string
	KeyName string
	Expr    *Node
	Body    *Node
}

// NForInt - for statement
type NForInt struct {
	VarName string
	From    *Node
	To      *Node
	Body    *Node
}

// NArray - init array
type NArray struct {
	List []*Node
}

type KeyVal struct {
	Key   string
	Value *Node
}

// NMap - init map
type NMap struct {
	List []KeyVal
}

// NGetIndex - getting index
type NGetIndex struct {
	Name    string
	Indexes []*Node
}

// NFunc - function
type NFunc struct {
	Name   string
	Result *Node
	Params []NVar
	Body   *Node
}

// NCallFunc - call function
type NCallFunc struct {
	Name   string
	Params *Node
}

// NCallContract - call contract
type NCallContract struct {
	Name   string
	Params []ContractParam
}

// NIf - if statement
type NIf struct {
	Cond     *Node
	IfBody   *Node
	ElifBody *Node
	ElseBody *Node
}

// NElifBody - elif statement
type NElifBody struct {
	Cond *Node
	Body *Node
}

// NElif - elif statement
type NElif struct {
	List []*NElifBody
}

// NBlock contains body of contract
type NBlock struct {
	Params     []NVar
	Statements []*Node
}

// NParams contains param expressions
type NParams struct {
	Expr []*Node
}

type ContractParam struct {
	Name string
	Expr *Node
}

// NContractParams contains contract param expressions
type NContractParams struct {
	Params []ContractParam
}

// NBinary contains binary operator
type NBinary struct {
	Oper  int
	Left  *Node
	Right *Node
}

// NQuestion contains ? operator
type NQuestion struct {
	Cond  *Node
	Left  *Node
	Right *Node
}

// NUnary contains an unary operator
type NUnary struct {
	Oper    int
	Operand *Node
}

// NVars contains type and name of variable or parameter
type NVars struct {
	Vars []NVar
}

// NReturn is a return statement
type NReturn struct {
	Expr *Node
}

// NContract is a root node
type NContract struct {
	Name  string // the name of the contract
	Read  bool
	Block *Node
}

// Node is a common node structure for yacc
type Node struct {
	Type   int
	Line   int
	Column uint32
	Result uint32
	Value  interface{}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandName() string {
	alpha := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	length := len(alpha)
	b := make([]rune, 16)
	for i := range b {
		b[i] = alpha[rand.Intn(length)]
	}
	return string(b)
}

func setPos(node *Node, l yyLexer) *Node {
	pos := l.(*lexer).FilePosition()
	node.Line = pos.Line
	node.Column = uint32(pos.Column)
	return node
}

func newBreak(l yyLexer) *Node {
	return setPos(&Node{
		Type: TBreak,
	}, l)
}

func newContinue(l yyLexer) *Node {
	return setPos(&Node{
		Type: TContinue,
	}, l)
}

func newContract(name string, read bool, block *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TContract,
		Value: &NContract{
			Name:  name,
			Read:  read,
			Block: block,
		},
	}, l)
}

func newBlock(vars []NVar, block *Node, l yyLexer) *Node {
	if block == nil {
		block = setPos(&Node{
			Type:  TBlock,
			Value: &NBlock{},
		}, l)
	}
	if len(vars) > 0 {
		block.Value.(*NBlock).Params = vars
	}
	return block
}

func newParam(expr *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TParams,
		Value: &NParams{
			Expr: []*Node{expr},
		},
	}, l)
}

func newType(itype int64, l yyLexer) *Node {
	var def bool
	if itype == VArr || itype == VMap {
		def = true
		itype |= VStr << 4
	}
	return setPos(&Node{
		Type: TType,
		Value: &NType{
			Type: itype,
			Def:  def,
		},
	}, l)
}

func addSubtype(tNode *Node, ichild int64, l yyLexer) *Node {
	itype := tNode.Value.(*NType).Type
	var (
		i uint64
	)
	if tNode.Value.(*NType).Def {
		for i = 12; i > 0; i -= 4 {
			if (itype >> i) != 0 {
				itype &= ^(0xf << i)
				break
			}
		}
		tNode.Value.(*NType).Def = false
	}
	if (itype >> 12) == 0 {
		for i = 4; i < 16; i += 4 {
			if itype&(0xf<<i) == 0 {
				if (itype>>(i-4)) != VArr && (itype>>(i-4)) != VMap {
					itype = VVoid
				} else {
					itype |= ichild << i
				}
				break
			}
		}
	} else {
		itype = VVoid
	}
	if itype != VVoid && (ichild == VArr || ichild == VMap) {
		if itype&0xf000 != 0 {
			itype = VVoid
		} else {
			tNode.Value.(*NType).Def = true
			itype |= VStr << (i + 4)
		}
	}
	tNode.Value.(*NType).Type = itype
	return tNode
}

func addParam(node *Node, expr *Node) *Node {
	node.Value.(*NParams).Expr = append(node.Value.(*NParams).Expr, expr)
	return node
}

func newContractParam(name string, expr *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TContractParams,
		Value: &NContractParams{
			Params: []ContractParam{
				{
					Name: name,
					Expr: expr,
				},
			},
		},
	}, l)
}

func addContractParam(params *Node, name string, expr *Node) *Node {
	params.Value.(*NContractParams).Params = append(params.Value.(*NContractParams).Params,
		ContractParam{
			Name: name,
			Expr: expr,
		})
	return params
}

func newReturn(expr *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TReturn,
		Value: &NReturn{
			Expr: expr,
		},
	}, l)
}

func newVars(vtype *Node, vars []string) []NVar {
	va := make([]NVar, len(vars))
	for i, name := range vars {
		va[i] = NVar{
			Type: vtype,
			Name: name,
		}
	}
	return va
}

func newVarExp(vtype *Node, name string, exp *Node, l yyLexer) []NVar {
	va := make([]NVar, 1)
	va = []NVar{
		{
			Type: vtype,
			Name: name,
			Exp:  newBinary(newVarValue(name, l), exp, ASSIGN, l),
		},
	}
	return va
}

func addStatement(statements *Node, statement *Node, l yyLexer) *Node {
	if statements == nil {
		list := &NBlock{
			Statements: make([]*Node, 1, 10),
		}
		list.Statements[0] = statement
		statements = setPos(&Node{
			Type:  TBlock,
			Value: list,
		}, l)
	} else {
		statements.Value.(*NBlock).Statements = append(statements.Value.(*NBlock).Statements, statement)
	}
	return statements
}

func newValue(value interface{}, l yyLexer) *Node {
	var vtype uint32
	switch value.(type) {
	case float64:
		vtype = VFloat
	case int64:
		vtype = VInt
	case string:
		vtype = VStr
	case bool:
		vtype = VBool
	}
	return setPos(&Node{
		Type:   TValue,
		Value:  value,
		Result: vtype,
	}, l)
}

func newVarDecl(tNode *Node, vars []string, l yyLexer) *Node {
	list := make([]NVar, len(vars))
	for i, v := range vars {
		list[i] = NVar{
			Type: tNode,
			Name: v,
		}
	}
	return setPos(&Node{
		Type: TVars,
		Value: &NVars{
			Vars: list,
		},
	}, l)
}

func newBinary(left *Node, right *Node, oper int, l yyLexer) *Node {
	return setPos(&Node{
		Type: TBinary,
		Value: &NBinary{
			Oper:  oper,
			Left:  left,
			Right: right,
		},
	}, l)
}

func newVarValue(name string, l yyLexer) *Node {
	return setPos(&Node{
		Type: TSetVar,
		Value: &NVarValue{
			Name: name,
		},
	}, l)
}

func newEnv(name string, l yyLexer) *Node {
	return setPos(&Node{
		Type: TEnv,
		Value: &NEnv{
			Name: name,
		},
	}, l)
}

func newGetVar(name string, l yyLexer) *Node {
	return setPos(&Node{
		Type: TGetVar,
		Value: &NVarValue{
			Name: name,
		},
	}, l)
}

func newIndex(name string, index *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TGetIndex,
		Value: &NGetIndex{
			Name:    name,
			Indexes: []*Node{index},
		},
	}, l)
}

func addIndex(left *Node, index *Node, l yyLexer) *Node {
	left.Value.(*NGetIndex).Indexes = append(left.Value.(*NGetIndex).Indexes, index)
	return left
}

func newUnary(operand *Node, oper int, l yyLexer) *Node {
	return setPos(&Node{
		Type: TUnary,
		Value: &NUnary{
			Oper:    oper,
			Operand: operand,
		},
	}, l)
}

func newWhile(cond *Node, body *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TWhile,
		Value: &NWhile{
			Cond: cond,
			Body: body,
		},
	}, l)
}

func newFor(ivar string, expr *Node, body *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TFor,
		Value: &NFor{
			VarName: ivar,
			KeyName: ``,
			Expr:    expr,
			Body:    body,
		},
	}, l)
}

func newForAll(ivar, ikey string, expr *Node, body *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TFor,
		Value: &NFor{
			VarName: ivar,
			KeyName: ikey,
			Expr:    expr,
			Body:    body,
		},
	}, l)
}

func newForInt(ivar string, from *Node, to *Node, body *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TForInt,
		Value: &NForInt{
			VarName: ivar,
			From:    from,
			To:      to,
			Body:    body,
		},
	}, l)
}

func newIf(cond *Node, ifbody *Node, elif *Node, elsebody *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TIf,
		Value: &NIf{
			Cond:     cond,
			IfBody:   ifbody,
			ElifBody: elif,
			ElseBody: elsebody,
		},
	}, l)
}

func newElif(statements *Node, cond *Node, statement *Node, l yyLexer) *Node {
	if statements == nil {
		list := &NElif{
			List: make([]*NElifBody, 1, 10),
		}
		list.List[0] = &NElifBody{
			Cond: cond,
			Body: statement,
		}
		statements = setPos(&Node{
			Type:  TElif,
			Value: list,
		}, l)
	} else {
		statements.Value.(*NElif).List = append(statements.Value.(*NElif).List, &NElifBody{
			Cond: cond,
			Body: statement,
		})
	}
	return statements
}

func newQuestion(cond *Node, left *Node, right *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TQuestion,
		Value: &NQuestion{
			Cond:  cond,
			Left:  left,
			Right: right,
		},
	}, l)
}

func newFunc(name string, pars []NVar, retType *Node, body *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TFunc,
		Value: &NFunc{
			Name:   name,
			Result: retType,
			Body:   body,
			Params: pars,
		},
	}, l)
}

func newCallFunc(name string, params *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TCallFunc,
		Value: &NCallFunc{
			Name:   name,
			Params: params,
		},
	}, l)
}

func newCallContract(name string, params *Node, l yyLexer) *Node {
	var list []ContractParam
	if params != nil {
		list = params.Value.(*NContractParams).Params
	}
	return setPos(&Node{
		Type: TCallContract,
		Value: &NCallContract{
			Name:   name[1:],
			Params: list,
		},
	}, l)
}

func newArray(par *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TArray,
		Value: &NArray{
			List: []*Node{par},
		},
	}, l)
}

func appendArray(list *Node, par *Node, l yyLexer) *Node {
	list.Value.(*NArray).List = append(list.Value.(*NArray).List, par)
	return list
}

func newMap(key string, par *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TMap,
		Value: &NMap{
			List: []KeyVal{{Key: key, Value: par}},
		},
	}, l)
}

func appendMap(list *Node, key string, par *Node, l yyLexer) *Node {
	list.Value.(*NMap).List = append(list.Value.(*NMap).List, KeyVal{
		Key: key, Value: par,
	})
	return list
}

func newObj(key string, par *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TObject,
		Value: &NObject{
			List: []KeyVal{{Key: key, Value: par}},
		},
	}, l)
}

func appendObj(list *Node, key string, par *Node, l yyLexer) *Node {
	list.Value.(*NObject).List = append(list.Value.(*NObject).List, KeyVal{
		Key: key, Value: par,
	})
	return list
}

func newObjList(par *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TObjList,
		Value: &NObjList{
			Obj: par,
		},
	}, l)
}

func newObjArr(par *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TObjArr,
		Value: &NObjArr{
			List: []*Node{par},
		},
	}, l)
}

func appendObjArr(list *Node, par *Node, l yyLexer) *Node {
	list.Value.(*NObjArr).List = append(list.Value.(*NObjArr).List, par)
	return list
}

func createCase(expr *Node, statement *Node, l yyLexer) *Node {
	list := make([]*NCaseItem, 1, 10)
	list[0] = &NCaseItem{
		ExprList: expr,
		Body:     statement,
	}
	return setPos(&Node{
		Type: TCase,
		Value: &NCase{
			List: list,
		},
	}, l)
}

func newCase(statements *Node, expr *Node, statement *Node, l yyLexer) *Node {
	if statements == nil {
		return createCase(expr, statement, l)
	}
	statements.Value.(*NCase).List = append(statements.Value.(*NCase).List, &NCaseItem{
		ExprList: expr,
		Body:     statement,
	})
	return statements
}

func newSwitch(expr *Node, icase *Node, def *Node, l yyLexer) *Node {
	return setPos(&Node{
		Type: TSwitch,
		Value: &NSwitch{
			Expr:    expr,
			Case:    icase,
			Default: def,
		},
	}, l)
}

// Parser creates AST
func Parser(input string) (*Node, error) {
	yyErrorVerbose = true

	l, err := NewLexer(``, input)
	if err != nil {
		return nil, err
	}
	yyParse(l)
	if l.err != nil {
		return nil, l.err
	}
	return l.result.(*Node), nil
}
