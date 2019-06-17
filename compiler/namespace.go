package compiler

import (
	"fmt"

	"github.com/shelmesky/bvm/parser"
	rt "github.com/shelmesky/bvm/runtime"
)

const (
	EMBEDDED = 0x1000
	CUSTOM   = 0x2000
)

var (
	operators = [][]uint32{
		// BCode, Result, Operator, Type of parameters...
		{rt.SIGNINT, parser.VInt, parser.SUB, parser.VInt},                                 // -int
		{rt.NOT, parser.VBool, parser.NOT, parser.VBool},                                   // !bool
		{rt.ADDINT, parser.VInt, parser.ADD, parser.VInt, parser.VInt},                     // int+int
		{rt.SUBINT, parser.VInt, parser.SUB, parser.VInt, parser.VInt},                     // int-int
		{rt.MULINT, parser.VInt, parser.MUL, parser.VInt, parser.VInt},                     // int*int
		{rt.DIVINT, parser.VInt, parser.DIV, parser.VInt, parser.VInt},                     // int/int
		{rt.MODINT, parser.VInt, parser.MOD, parser.VInt, parser.VInt},                     // int%int
		{rt.ASSIGNINT, parser.VVoid, parser.ASSIGN, parser.VInt, parser.VInt},              // int = int
		{rt.ASSIGNADDINT, parser.VVoid, parser.ADD_ASSIGN, parser.VInt, parser.VInt},       // int += int
		{rt.ASSIGNSUBINT, parser.VVoid, parser.SUB_ASSIGN, parser.VInt, parser.VInt},       // int -= int
		{rt.ASSIGNMULINT, parser.VVoid, parser.MUL_ASSIGN, parser.VInt, parser.VInt},       // int *= int
		{rt.ASSIGNDIVINT, parser.VVoid, parser.DIV_ASSIGN, parser.VInt, parser.VInt},       // int /= int
		{rt.ASSIGNMODINT, parser.VVoid, parser.MOD_ASSIGN, parser.VInt, parser.VInt},       // int %= int
		{rt.EQINT, parser.VBool, parser.EQ, parser.VInt, parser.VInt},                      // int == int
		{rt.LTINT, parser.VBool, parser.LT, parser.VInt, parser.VInt},                      // int < int
		{rt.GTINT, parser.VBool, parser.GT, parser.VInt, parser.VInt},                      // int > int
		{rt.ASSIGNINT, parser.VVoid, parser.ASSIGN, parser.VBool, parser.VBool},            // bool = bool
		{rt.AND, parser.VBool, parser.AND, parser.VBool, parser.VBool},                     // bool && bool
		{rt.OR, parser.VBool, parser.OR, parser.VBool, parser.VBool},                       // bool || bool
		{rt.ADDSTR, parser.VStr, parser.ADD, parser.VStr, parser.VStr},                     // str+str
		{rt.ASSIGNSTR, parser.VVoid, parser.ASSIGN, parser.VStr, parser.VStr},              // str = str
		{rt.ASSIGNADDSTR, parser.VVoid, parser.ADD_ASSIGN, parser.VStr, parser.VStr},       // str += str
		{rt.EQSTR, parser.VBool, parser.EQ, parser.VStr, parser.VStr},                      // str == str
		{rt.SIGNFLOAT, parser.VFloat, parser.SUB, parser.VFloat},                           // -float
		{rt.ADDFLOAT, parser.VFloat, parser.ADD, parser.VFloat, parser.VFloat},             // float+float
		{rt.SUBFLOAT, parser.VFloat, parser.SUB, parser.VFloat, parser.VFloat},             // float-float
		{rt.ASSIGNINT, parser.VVoid, parser.ASSIGN, parser.VFloat, parser.VFloat},          // float = float
		{rt.MULFLOAT, parser.VFloat, parser.MUL, parser.VFloat, parser.VFloat},             // float*float
		{rt.DIVFLOAT, parser.VFloat, parser.DIV, parser.VFloat, parser.VFloat},             // float/float
		{rt.ASSIGNADDFLOAT, parser.VVoid, parser.ADD_ASSIGN, parser.VFloat, parser.VFloat}, // float += float
		{rt.ASSIGNSUBFLOAT, parser.VVoid, parser.SUB_ASSIGN, parser.VFloat, parser.VFloat}, // float -= float
		{rt.ASSIGNMULFLOAT, parser.VVoid, parser.MUL_ASSIGN, parser.VFloat, parser.VFloat}, // float *= float
		{rt.ASSIGNDIVFLOAT, parser.VVoid, parser.DIV_ASSIGN, parser.VFloat, parser.VFloat}, // float /= float
		{rt.ASSIGNINT, parser.VVoid, parser.ASSIGN, parser.VMoney, parser.VMoney},          // money = money
		{rt.ADDMONEY, parser.VMoney, parser.ADD, parser.VMoney, parser.VMoney},             // money+money
		{rt.SUBMONEY, parser.VMoney, parser.SUB, parser.VMoney, parser.VMoney},             // money-money
		{rt.SIGNMONEY, parser.VMoney, parser.SUB, parser.VMoney},                           // -money
		{rt.MULMONEY, parser.VMoney, parser.MUL, parser.VMoney, parser.VMoney},             // money*money
		{rt.DIVMONEY, parser.VMoney, parser.DIV, parser.VMoney, parser.VMoney},             // money/money
		{rt.ASSIGNADDMONEY, parser.VVoid, parser.ADD_ASSIGN, parser.VMoney, parser.VMoney}, // money += money
		{rt.ASSIGNSUBMONEY, parser.VVoid, parser.SUB_ASSIGN, parser.VMoney, parser.VMoney}, // money -= money
		{rt.ASSIGNMULMONEY, parser.VVoid, parser.MUL_ASSIGN, parser.VMoney, parser.VMoney}, // money *= money
		{rt.ASSIGNDIVMONEY, parser.VVoid, parser.DIV_ASSIGN, parser.VMoney, parser.VMoney}, // money /= money
		{rt.EQFLOAT, parser.VBool, parser.EQ, parser.VFloat, parser.VFloat},                // float == float
		{rt.LTFLOAT, parser.VBool, parser.LT, parser.VFloat, parser.VFloat},                // float < float
		{rt.GTFLOAT, parser.VBool, parser.GT, parser.VFloat, parser.VFloat},                // float > float
		{rt.EQMONEY, parser.VBool, parser.EQ, parser.VMoney, parser.VMoney},                // money == money
		{rt.LTMONEY, parser.VBool, parser.LT, parser.VMoney, parser.VMoney},                // money < money
		{rt.GTMONEY, parser.VBool, parser.GT, parser.VMoney, parser.VMoney},                // money > money
		{rt.ASSIGNINT, parser.VObject, parser.ASSIGN, parser.VObject, parser.VObject},      // obj = obj
		{rt.ASSIGNINT, parser.VVoid, parser.ASSIGN, parser.VBytes, parser.VBytes},          // bytes = bytes
		{rt.ASSIGNADDBYTES, parser.VVoid, parser.ADD_ASSIGN, parser.VBytes, parser.VBytes}, // bytes += bytes
	}
)

func parseType(intype uint32) (outtype, subtype uint32) {
	if intype == parser.VBytes {
		subtype = parser.VInt
		outtype = intype
	}
	if intype > 0xf {
		subtype = intype >> 4
		outtype = intype & 0xf
	}
	return
}

func (cmpl *compiler) findBinary(binary *parser.NBinary) (rt.Bcode, uint32) {
	key := fmt.Sprintf("#%d#%d#%d", binary.Oper, binary.Left.Result, binary.Right.Result)
	if v, ok := (*cmpl.NameSpace)[key]; ok {
		return rt.Bcode(v & 0xffff), v >> 24
	}
	if binary.Oper == parser.ADD_ASSIGN {
		outtype, subtype := parseType(binary.Left.Result)
		if outtype&0xf == parser.VArr && subtype == binary.Right.Result {
			return rt.APPENDARR, parser.VVoid
		}
	}
	if binary.Oper == parser.ASSIGN && (binary.Left.Result&0xf == parser.VArr ||
		binary.Left.Result&0xf == parser.VMap || binary.Left.Result&0xf == parser.VBytes) {
		if binary.Left.Result == binary.Right.Result {
			return rt.ASSIGNINT, parser.VVoid
		}
		outtype, subtype := parseType(binary.Left.Result)
		if subtype == binary.Right.Result {
			if binary.Right.Result == parser.VStr {
				cmpl.Append(rt.COPYSTR)
			}
			if binary.Right.Result == parser.VMoney {
				cmpl.Append(rt.COPY, parser.VMoney)
			}
			if outtype&0xf == parser.VMap {
				return rt.ASSIGNSETMAP, parser.VVoid
			}
			if outtype&0xf == parser.VArr {
				return rt.ASSIGNSETARR, parser.VVoid
			}
			if outtype&0xf == parser.VBytes {
				return rt.ASSIGNSETBYTES, parser.VVoid
			}
		}
	}
	return rt.NOP, 0
}

func (cmpl *compiler) findUnary(unary *parser.NUnary) (rt.Bcode, uint32) {
	key := fmt.Sprintf("#%d#%d", unary.Oper, unary.Operand.Result)
	if v, ok := (*cmpl.NameSpace)[key]; ok {
		return rt.Bcode(v & 0xffff), v >> 24
	}
	return rt.NOP, 0
}

func getFuncKey(nfunc *rt.FuncInfo) string {
	ret := fmt.Sprintf("$%s", nfunc.Name)
	for _, par := range nfunc.Params {
		ret += fmt.Sprintf(`$%d`, par.Type)
	}
	return ret
}

// 在namespace中查找一个函数，如果存在则返回对象的value(namesapce是一个map[string]uint32)
// 不存在则返回rt.NOP
func (cmpl *compiler) findFunc(nfunc *rt.FuncInfo) (rt.Bcode, uint32) {
	key := getFuncKey(nfunc)                 //key是: $函数名$第一个参数类型值$第二个参数类型值...， 例如$myfunc$1$2$5
	if v, ok := (*cmpl.NameSpace)[key]; ok { // 如果在copl.NameSpace中存在这个key，说明存在这个函数，返回
		return rt.Bcode(v & 0xffff), v >> 24
	}
	return rt.NOP, 0
}

func (cmpl *compiler) findCallFunc(nfunc *parser.NCallFunc) (rt.Bcode, uint32) {
	key := fmt.Sprintf("$%s", nfunc.Name)
	softkey := key
	if nfunc.Params != nil {
		for _, par := range nfunc.Params.Value.(*parser.NParams).Expr {
			parkey := fmt.Sprintf(`$%d`, par.Result)
			if par.Result > 0xf {
				outtype, _ := parseType(par.Result)
				softkey += fmt.Sprintf(`$%d`, outtype)
			} else {
				softkey += parkey
			}
			key += parkey
		}
	}
	if v, ok := (*cmpl.NameSpace)[key]; ok {
		return rt.Bcode(v & 0xffff), v >> 24
	} else if len(softkey) > 0 {
		if v, ok := (*cmpl.NameSpace)[softkey]; ok {
			return rt.Bcode(v & 0xffff), v >> 24
		}
	}
	return rt.NOP, 0
}

/*
初始化命名空间
命令空间保存的又下列内容：
1. 表达式操作符
2. 内置函数
3. 用户自定义函数

命名空间的原理是在map中保存对象的信息，这种信息类似于给对象创建一个hash.
key是对象中各元素的组合的字符串，value是另外一些元素的hash.

*/
func initNameSpace(cmpl *compiler, nameSpace *map[string]uint32) {
	for _, oper := range operators { // 内置操作符
		var key string
		for i := 2; i < len(oper); i++ {
			key += fmt.Sprintf(`#%d`, oper[i])
		}
		(*nameSpace)[key] = oper[0] | (oper[1] << 24)
	}

	for i, eFunc := range rt.StdLib { // 内置函数
		key := fmt.Sprintf(`$%s`, eFunc.Name)
		for _, par := range eFunc.PTypes {
			key += fmt.Sprintf(`$%d`, par)
		}
		(*nameSpace)[key] = uint32(i+EMBEDDED) | (eFunc.Result << 24)
	}

	for i, fItem := range cmpl.Custom.Funcs { // 用户自定义函数
		key := fmt.Sprintf(`$%s`, fItem.Name)
		for _, par := range fItem.Params {
			key += fmt.Sprintf(`$%d`, par)
		}
		(*nameSpace)[key] = uint32(i+CUSTOM) | (fItem.Result << 24)
	}
}

// Type2Str return a name of the type
func Type2Str(vtype uint32) (ret string) {
main:
	for i := 0; i < 4; i++ {
		itype := vtype & 0xf
		if i > 0 && itype != parser.VVoid {
			ret += `.`
		}
		switch itype {
		case parser.VInt:
			ret += `int`
		case parser.VBool:
			ret += `bool`
		case parser.VStr:
			ret += `str`
		case parser.VArr:
			ret += `arr`
		case parser.VMap:
			ret += `map`
		case parser.VFloat:
			ret += `float`
		case parser.VMoney:
			ret += `money`
		case parser.VObject:
			ret += `obj`
		case parser.VBytes:
			ret += `bytes`
		default:
			break main
		}
		vtype >>= 4
	}
	if len(ret) == 0 {
		ret = `unknown`
	}
	return
}
