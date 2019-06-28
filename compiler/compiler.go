package compiler

import (
	"fmt"
	"math"
	"strings"
	"unsafe"

	"github.com/shelmesky/bvm/parser"
	rt "github.com/shelmesky/bvm/runtime"
)

const (
	Debug = false
)

type jumps struct {
	Breaks    []int
	Continues []int
}

type compiler struct {
	Contract  *rt.Contract
	Blocks    []*parser.Node
	Contracts *[]*rt.Contract
	Custom    *rt.Custom
	NameSpace *map[string]uint32
	RetFunc   int64
	InFunc    bool
	Data      []byte
	Jumps     []*jumps
}

func DebugPrintf(formatString string, a ...interface{}) {
	fmt.Printf(formatString, a...)
}

func (cmpl *compiler) Append(codes ...rt.Bcode) {
	// for debug
	if Debug {
		fmt.Println(codes)
		codeLen := len(codes)
		if codeLen > 0 {
			instr := codes[0]
			fmt.Printf("指令: %d ", instr)
			if codeLen > 1 {
				fmt.Printf("操作数 ")
				for i := 1; i < codeLen; i++ {
					fmt.Printf("%d ", codes[i])
				}
			}
			fmt.Printf("\n")
		}
	}

	for _, code := range codes {
		cmpl.Contract.Code = append(cmpl.Contract.Code, code)
	}
}

func (cmpl *compiler) JumpOff(node *parser.Node, off int) (rt.Bcode, error) {
	if off < math.MinInt16 || off > math.MaxInt16 {
		return rt.NOP, cmpl.Error(node, errJump)
	}
	return rt.Bcode(off), nil
}

func (cmpl *compiler) ConditionCode(node *parser.Node) (before int, after int, err error) {
	before = len(cmpl.Contract.Code)
	if err = nodeToCode(node, cmpl); err != nil {
		return
	}
	if node.Result != parser.VBool {
		err = cmpl.ErrorParam(node, errCond, Type2Str(node.Result))
		return
	}
	after = len(cmpl.Contract.Code)
	return
}

/*
在Contract.Vars保存变量， 并将变量的出现顺序作为字节码操作的索引， 例如SETVAR/GETVAR指令。
然后生成INITVARS指令， 指令的参数是需要初始化的变量数量。
编译是有3个地方调用本函数：
1. 合约的参数部分， 即data块。
2. 函数定义中的参数列表。
3. 声明一个变量。
*/
func (cmpl *compiler) InitVars(node *parser.Node, vars []parser.NVar) error {
	if len(vars) == 0 {
		return nil
	}
	types := make([]rt.Bcode, len(vars))
	for i, v := range vars {
		if _, ok := cmpl.Contract.Vars[v.Name]; ok {
			return cmpl.ErrorParam(node, errVarExists, v.Name)
		}
		vType := v.Type.Value.(*parser.NType).Type
		if vType == parser.VVoid {
			return cmpl.Error(v.Type, errInvalidType)
		}
		types[i] = rt.Bcode(vType)
		cmpl.Contract.Vars[v.Name] = rt.VarInfo{
			Index: uint16(len(cmpl.Contract.Vars)),
			Type:  uint16(vType),
		}
	}
	cmpl.Append(rt.INITVARS, rt.Bcode(len(types)))
	cmpl.Append(types...)
	for _, v := range vars {
		if v.Exp != nil {
			if err := nodeToCode(v.Exp, cmpl); err != nil {
				return err
			}
		}
	}
	return nil
}

func nodeToCode(node *parser.Node, cmpl *compiler) error {
	var (
		err                error
		vinfo              rt.VarInfo
		ok                 bool
		sizeCode, sizeCond int
	)
	if node == nil {
		return nil
	}

	fmt.Printf("compile node type: %s\n", parser.GetNodeType(node.Type))

	switch node.Type {
	case parser.TBlock: // 编译代码块
		/*
			合约contract{}和函数定义func xxx() {}都会匹配到TBlock类型，
			如果类型是合约， 则其中的data{}会被当作Block的参数.
			如果类型是函数， 进入到这里表明已经在处理函数体， 而函数的参数和返回值在TFunc类型中已经被处理。
		*/
		varsCount := uint16(len(cmpl.Contract.Vars)) // 当前合约内所有的变量(var声明和函数参数)
		funcsCount := len(cmpl.Contract.Funcs)       // 当前合约的所有函数
		cmpl.Blocks = append(cmpl.Blocks, node)
		pars := node.Value.(*parser.NBlock).Params // 当前Block代码的参数数量
		// 如果参数数量大于0 (进入到次分支， 说明正在编译contract的data结构。函数的Block不会进入此分支.
		// 因为函数的参数已经在TFunc类型中处理。
		if len(pars) > 0 {
			if err = cmpl.InitVars(node, pars); err != nil { // 初始化变量， 生成INITVARS指令.
				return err
			}
			cmpl.Contract.Params = make(map[string]rt.VarInfo) // 初始化contract的参数
			for k, ipar := range pars {                        // 将每个参数保存在编译结果的Contract.Params这个map中
				cmpl.Contract.Params[ipar.Name] = rt.VarInfo{Index: uint16(k),
					Type: uint16(ipar.Type.Value.(*parser.NType).Type)}
			}
			cmpl.Append(rt.LOADPARS) // 生成LOADPARS指令
		}

		for _, child := range node.Value.(*parser.NBlock).Statements { // 编译block中的语句
			if err = nodeToCode(child, cmpl); err != nil {
				return err
			}
		}

		cmpl.Blocks = cmpl.Blocks[:len(cmpl.Blocks)-1]

		if uint16(len(cmpl.Contract.Vars)) != varsCount &&
			cmpl.Contract.Code[len(cmpl.Contract.Code)-1] != rt.RETFUNC {
			cmpl.Append(rt.DELVARS, rt.Bcode(varsCount))
		}

		//vinfo.Index >= varsCount说明在进入Block编译后，比进入Block之前多了很多变量，这些变量是Block内的变量
		//应该在contract.Vars中删除这些局部变量?
		// Remove vars

		for key, _ := range cmpl.Contract.Vars {
			if vinfo.Index >= varsCount {
				delete(cmpl.Contract.Vars, key)
			}
		}

		if funcsCount < len(cmpl.Contract.Funcs) {
			// Remove funcs
			for i := funcsCount; i < len(cmpl.Contract.Funcs); i++ {
				delete(*cmpl.NameSpace, getFuncKey(cmpl.Contract.Funcs[i]))
			}
			cmpl.Contract.Funcs = cmpl.Contract.Funcs[:funcsCount]
		}

	case parser.TContract:
		cmpl.Contract.Name = node.Value.(*parser.NContract).Name
		cmpl.Contract.Read = node.Value.(*parser.NContract).Read
		if err = nodeToCode(node.Value.(*parser.NContract).Block, cmpl); err != nil {
			return err
		}
	case parser.TReturn:
		var vtype uint32
		expr := node.Value.(*parser.NReturn).Expr
		if expr != nil {
			if err = nodeToCode(expr, cmpl); err != nil {
				return err
			}
			vtype = expr.Result
		}
		if cmpl.InFunc {
			if vtype != uint32(cmpl.RetFunc) {
				if cmpl.RetFunc == parser.VVoid {
					return cmpl.Error(node, errNotReturn)
				}
				if vtype == parser.VVoid {
					return cmpl.Error(node, errFuncReturn)
				}
				return cmpl.ErrorParam(node, errReturnType, Type2Str(uint32(cmpl.RetFunc)))
			}
			cmpl.Append(rt.RETFUNC)
		} else {
			cmpl.Append(rt.RETURN, rt.Bcode(vtype))
		}
	case parser.TBinary:
		nBinary := node.Value.(*parser.NBinary)
		if nBinary.Left.Type == parser.TGetIndex && nBinary.Oper == parser.ASSIGN {
			nBinary.Left.Type = parser.TSetIndex
		}
		if err = nodeToCode(nBinary.Left, cmpl); err != nil {
			return err
		}
		if nBinary.Left.Type == parser.TVars { // type varName =
			nBinary.Left = &parser.Node{
				Type: parser.TSetVar,
				Value: &parser.NVarValue{
					Name: nBinary.Left.Value.(*parser.NVars).Vars[0].Name,
				},
			}
			if err = nodeToCode(nBinary.Left, cmpl); err != nil {
				return err
			}
		}
		forJump := len(cmpl.Contract.Code)
		if err = nodeToCode(nBinary.Right, cmpl); err != nil {
			return err
		}
		var notCmp bool
		if nBinary.Oper == parser.NOT_EQ || nBinary.Oper == parser.LTE || nBinary.Oper == parser.GTE {
			notCmp = true
			switch nBinary.Oper {
			case parser.NOT_EQ:
				nBinary.Oper = parser.EQ
			case parser.LTE:
				nBinary.Oper = parser.GT
			case parser.GTE:
				nBinary.Oper = parser.LT
			}
		}
		code, result := cmpl.findBinary(nBinary)
		var jumpCmd rt.Bcode
		switch code {
		case rt.NOP:
			return cmpl.ErrorOperator(node)
		case rt.AND:
			jumpCmd = rt.Bcode(rt.JZE)
		case rt.OR:
			jumpCmd = rt.Bcode(rt.JNZ)
		}
		cmpl.Append(code)
		if notCmp {
			cmpl.Append(rt.NOT)
		}
		if jumpCmd != rt.NOP {
			cmpl.Contract.Code = append(cmpl.Contract.Code[:forJump],
				append([]rt.Bcode{rt.DUP, jumpCmd, rt.Bcode(len(cmpl.Contract.Code) - forJump + 2)},
					cmpl.Contract.Code[forJump:]...)...)
		}
		node.Result = result
	case parser.TUnary:
		nUnary := node.Value.(*parser.NUnary)
		if err = nodeToCode(nUnary.Operand, cmpl); err != nil {
			return err
		}
		code, result := cmpl.findUnary(nUnary)
		if code == rt.NOP {
			return cmpl.ErrorOperator(node)
		}
		cmpl.Append(code)
		node.Result = result
	case parser.TQuestion:
		nQuestion := node.Value.(*parser.NQuestion)
		_, sizeCond, err = cmpl.ConditionCode(nQuestion.Cond)
		if err != nil {
			return err
		}
		cmpl.Append(rt.JZE, 0)
		if err = nodeToCode(nQuestion.Left, cmpl); err != nil {
			return err
		}
		sizeCode = len(cmpl.Contract.Code)
		cmpl.Append(rt.JMPREL, 0)
		if err = nodeToCode(nQuestion.Right, cmpl); err != nil {
			return err
		}
		if nQuestion.Left.Result != nQuestion.Right.Result {
			return cmpl.Error(node, errQuestTypes)
		}
		var off rt.Bcode
		if off, err = cmpl.JumpOff(node, sizeCode-sizeCond+2); err != nil {
			return err
		}
		cmpl.Contract.Code[sizeCond+1] = off

		if off, err = cmpl.JumpOff(node, len(cmpl.Contract.Code)-sizeCode); err != nil {
			return err
		}
		cmpl.Contract.Code[sizeCode+1] = off
		node.Result = nQuestion.Left.Result

	case parser.TValue: // 某种类型的字面值，例如123, "abc", 0.12等
		switch v := node.Value.(type) {
		case int64:
			if v <= math.MaxInt16 && v >= math.MinInt16 {
				cmpl.Append(rt.PUSH16, rt.Bcode(v))
			} else if v <= math.MaxInt32 && v >= math.MinInt32 {
				u32 := uint32(v)
				cmpl.Append(rt.PUSH32, rt.Bcode(u32>>16), rt.Bcode(u32&0xffff))
			} else {
				u64 := uint64(v)
				cmpl.Append(rt.PUSH64, rt.Bcode(u64>>48), rt.Bcode((u64>>32)&0xffff),
					rt.Bcode((u64>>16)&0xffff), rt.Bcode(u64&0xffff))
			}
			node.Result = parser.VInt
		case bool:
			var bcode rt.Bcode
			if v {
				bcode = 1
			}
			cmpl.Append(rt.PUSH16, bcode)
			node.Result = parser.VBool
		case string:
			if cmpl.Data == nil {
				cmpl.Data = make([]byte, 0, 1024)
			}
			cmpl.Append(rt.PUSHSTR, rt.Bcode(len(cmpl.Data)), rt.Bcode(len(v)))
			cmpl.Data = append(cmpl.Data, []byte(v)...)
		case float64:
			var uf = uintptr(unsafe.Pointer(&v))
			u64 := *(*uint64)(unsafe.Pointer(uf))
			cmpl.Append(rt.PUSH64, rt.Bcode(u64>>48), rt.Bcode((u64>>32)&0xffff),
				rt.Bcode((u64>>16)&0xffff), rt.Bcode(u64&0xffff))
			node.Result = parser.VFloat
		default:
			return cmpl.ErrorParam(node, errType, node.Value)
		}

	case parser.TVars: // 定义新的变量，例如 str a，或赋值str a = "aaa"
		if err = cmpl.InitVars(node, node.Value.(*parser.NVars).Vars); err != nil {
			return err
		}

	case parser.TGetVar: // 在表达式中出现的变量，需要对其求值
		name := node.Value.(*parser.NVarValue).Name
		if vinfo, ok = cmpl.Contract.Vars[name]; !ok {
			return cmpl.ErrorParam(node, errVarUnknown, name)
		}
		cmpl.Append(rt.GETVAR, rt.Bcode(vinfo.Index))
		node.Result = uint32(vinfo.Type)

	case parser.TSetVar: // 在语句中出现的变量，即单独一行语句。例如定义变量：var_a，或复制 var_a = "111"
		name := node.Value.(*parser.NVarValue).Name
		if vinfo, ok = cmpl.Contract.Vars[name]; !ok {
			return cmpl.ErrorParam(node, errVarUnknown, name)
		}
		cmpl.Append(rt.SETVAR, rt.Bcode(vinfo.Index))
		node.Result = uint32(vinfo.Type)

	case parser.TWhile:
		nWhile := node.Value.(*parser.NWhile)
		cmpl.Jumps = append(cmpl.Jumps, &jumps{})
		sizeCode, sizeCond, err = cmpl.ConditionCode(nWhile.Cond)
		if err != nil {
			return err
		}
		cmpl.Append(rt.JZE, 0)
		if err = nodeToCode(nWhile.Body, cmpl); err != nil {
			return err
		}
		for _, b := range cmpl.Jumps[len(cmpl.Jumps)-1].Continues {
			cmpl.Contract.Code[b] = rt.Bcode(sizeCode - b + 1)
		}
		var off rt.Bcode
		if off, err = cmpl.JumpOff(node, sizeCode-len(cmpl.Contract.Code)); err != nil {
			return err
		}
		cmpl.Append(rt.JMP, off)

		if off, err = cmpl.JumpOff(node, len(cmpl.Contract.Code)-sizeCond); err != nil {
			return err
		}
		cmpl.Contract.Code[sizeCond+1] = off
		for _, b := range cmpl.Jumps[len(cmpl.Jumps)-1].Breaks {
			cmpl.Contract.Code[b] = rt.Bcode(len(cmpl.Contract.Code) - b + 1)
		}
		cmpl.Jumps = cmpl.Jumps[:len(cmpl.Jumps)-1]
	case parser.TIf:
		ends := make([]int, 0, 16)
		nIf := node.Value.(*parser.NIf)
		_, sizeCond, err = cmpl.ConditionCode(nIf.Cond)
		if err != nil {
			return err
		}
		cmpl.Append(rt.JZE, 0)
		if err = nodeToCode(nIf.IfBody, cmpl); err != nil {
			return err
		}
		ends = append(ends, len(cmpl.Contract.Code))
		cmpl.Append(rt.JMP, 0)
		var off rt.Bcode
		if off, err = cmpl.JumpOff(node, len(cmpl.Contract.Code)-sizeCond); err != nil {
			return err
		}
		cmpl.Contract.Code[sizeCond+1] = off
		if nIf.ElifBody != nil {
			nElif := nIf.ElifBody.Value.(*parser.NElif)
			for _, child := range nElif.List {
				_, sizeCond, err = cmpl.ConditionCode(child.Cond)
				if err != nil {
					return err
				}
				cmpl.Append(rt.JZE, 0)
				if err = nodeToCode(child.Body, cmpl); err != nil {
					return err
				}
				ends = append(ends, len(cmpl.Contract.Code))
				cmpl.Append(rt.JMP, 0)
				if off, err = cmpl.JumpOff(node, len(cmpl.Contract.Code)-sizeCond); err != nil {
					return err
				}
				cmpl.Contract.Code[sizeCond+1] = off
			}
		}
		if nIf.ElseBody != nil {
			if err = nodeToCode(nIf.ElseBody, cmpl); err != nil {
				return err
			}
		}
		size := len(cmpl.Contract.Code)
		for _, end := range ends {
			if off, err = cmpl.JumpOff(node, size-end); err != nil {
				return err
			}
			cmpl.Contract.Code[end+1] = off
		}
	case parser.TFunc: //编译函数定义
		var (
			off rt.Bcode
		)
		retType := int64(parser.VVoid) // 初始化类型为void
		nFunc := node.Value.(*parser.NFunc)
		if nFunc.Result != nil { // 如果有返回类型
			retType = nFunc.Result.Value.(*parser.NType).Type // 获取返回类型
			if retType == parser.VVoid {
				return cmpl.Error(nFunc.Result, errInvalidType)
			}
		}
		finfo := &rt.FuncInfo{
			Name:   nFunc.Name,                        // 函数名称
			Result: retType,                           // 返回类型
			Params: make([]rt.Var, len(nFunc.Params)), // 参数列表
		}
		for ipar, v := range nFunc.Params { // 依次获取每个参数的类型和名称
			finfo.Params[ipar] = rt.Var{Type: v.Type.Value.(*parser.NType).Type, Name: v.Name}
		}
		if cmpl.InFunc { // 不允许在函数内定义函数
			return cmpl.Error(node, errFuncLevel)
		}
		cmpl.RetFunc = retType

		// 如果在namespace中已经存在此函数则返回错误
		if code, _ := cmpl.findFunc(finfo); code != rt.NOP {
			return cmpl.ErrorParam(node, errFuncExists, nFunc.Name)
		}

		start := len(cmpl.Contract.Code) // 目前合约所有的代码
		cmpl.Append(rt.JMP, 0)           // 在代码中插入JMP, 0指令
		finfo.Offset = start + 2         // 函数代码在
		cmpl.InFunc = true               // 设置"在函数中"标志为true

		// 初始化函数参数: 在code数组中插入[INITVARS, 类型长度，类型列表]
		// 为函数调用前做准备
		if err = cmpl.InitVars(node, nFunc.Params); err != nil {
			return err
		}

		// 如果函数有参数，则插入[GETPARAMS, 参数长度]指令
		// 这个指令会在函数执行之前从栈中获取每个参数的值，将这些值复制给之前初始化的参数
		if len(nFunc.Params) > 0 {
			cmpl.Append(rt.GETPARAMS, rt.Bcode(len(nFunc.Params)))
		}

		// 生成函数体指令
		if err = nodeToCode(nFunc.Body, cmpl); err != nil {
			return err
		}
		cmpl.InFunc = false // 设置在函数中标志为false

		// 如果函数最后的指令不是RETFUNC，且函数类型不是Void则报错
		if cmpl.Contract.Code[len(cmpl.Contract.Code)-1] != rt.RETFUNC {
			if cmpl.RetFunc != parser.VVoid {
				return cmpl.Error(node, errFuncReturn)
			}
			// 函数最后没有return关键字，则强行插入RETFUNC指令
			cmpl.Append(rt.RETFUNC)
		}

		// 将函数开始插入的JMP, 0变为真实的off值
		// 值为函数代码结束的位置
		// 因为函数执行需要靠别的代码调用，这里只是函数代码定义
		if off, err = cmpl.JumpOff(nFunc.Body, len(cmpl.Contract.Code)-start); err != nil {
			return err
		}
		cmpl.Contract.Code[start+1] = off

		// 在Contract.Funcs中保存函数信息
		cmpl.Contract.Funcs = append(cmpl.Contract.Funcs, finfo)
		// 在namespace中保存函数签名hash
		(*cmpl.NameSpace)[getFuncKey(finfo)] = uint32(len(cmpl.Contract.Funcs) | int(finfo.Result<<24))

	case parser.TCallFunc: // 函数调用
		nFunc := node.Value.(*parser.NCallFunc) // 函数名
		if nFunc.Params != nil {                //如果调用时有参数，则编译参数
			for _, expr := range nFunc.Params.Value.(*parser.NParams).Expr {
				if err = nodeToCode(expr, cmpl); err != nil {
					return err
				}
			}
		}
		code, ftype := cmpl.findCallFunc(nFunc)
		if code == rt.NOP {
			pars := make([]string, 0, 10)
			if nFunc.Params != nil {
				for _, par := range nFunc.Params.Value.(*parser.NParams).Expr {
					pars = append(pars, Type2Str(uint32(par.Result)))
				}
			}
			return cmpl.ErrorParam(node, errFuncNotExists, fmt.Sprintf("%s(%s)", nFunc.Name,
				strings.Join(pars, `, `)))
		}
		node.Result = ftype
		if code >= CUSTOM {
			if ftype > parser.VStr {
				return cmpl.ErrorParam(node, errRetType, nFunc.Name)
			}
			cmpl.Append(rt.CUSTOMFUNC, code-CUSTOM)

			if cmpl.Contract.Read && !cmpl.Custom.Funcs[code-CUSTOM].Read {
				return cmpl.Error(node, errReadContract)
			}
		} else if code >= EMBEDDED {
			cmpl.Append(rt.EMBEDFUNC, code-EMBEDDED)
		} else {
			var off rt.Bcode
			if off, err = cmpl.JumpOff(node, cmpl.Contract.Funcs[code-1].Offset-
				len(cmpl.Contract.Code)); err != nil {
				return err
			}
			cmpl.Append(rt.CALLFUNC, off)
		}
	case parser.TCallContract: // 调用其他合约
		nCallContract := node.Value.(*parser.NCallContract)
		// 在命名空间中根据合同名称寻找索引, 每编译好一个合约就在vm.Contracts中append，
		// 拿到索引值后添加到vm.NameSpace中。(代码在vm.Link函数中)
		// cmpl.NameSpace和cmpl.Contracts其实就是vm.NameSpace和vm.Contracts
		ind, ok := (*cmpl.NameSpace)[nCallContract.Name]
		if !ok {
			return cmpl.ErrorParam(node, errContractNotExists, nCallContract.Name)
		}
		cnt := (*cmpl.Contracts)[ind] // 找到真正contract
		if cmpl.Contract.Read && !cnt.Read {
			return cmpl.Error(node, errReadContract)
		}
		if len(nCallContract.Params) > 0 { // 如果调用contract时指定了参数
			if cnt.Params == nil { // 但如果真正的contract没有参数， 返回错误
				return cmpl.ErrorParam(node, errContractNoParams, nCallContract.Name)
			}
			for _, ipar := range nCallContract.Params { // 循环处理每个调用时指定的参数
				var (
					vinfo rt.VarInfo
					vok   bool
				)
				if vinfo, vok = cnt.Params[ipar.Name]; !vok { // 如果指定的参数
					return cmpl.ErrorParam(node, errContractNoParam, ipar.Name)
				}
				if err = nodeToCode(ipar.Expr, cmpl); err != nil { // 编译每个调用时指定的参数
					return err
				}
				if uint32(vinfo.Type) != ipar.Expr.Result { // 如果指定的参数类型不符合contract定义的参数类型就报错
					return cmpl.ErrorParam(node, errParamType, Type2Str(uint32(vinfo.Type)))
				}
				cmpl.Append(rt.PARCONTRACT, rt.Bcode(vinfo.Index), rt.Bcode(vinfo.Type))
			}
		}
		cmpl.Append(rt.CALLCONTRACT, rt.Bcode(ind))
		node.Result = parser.VStr
	case parser.TGetIndex:
		nGetIndex := node.Value.(*parser.NGetIndex)
		name := nGetIndex.Name
		if vinfo, ok = cmpl.Contract.Vars[name]; !ok {
			return cmpl.ErrorParam(node, errVarUnknown, name)
		}
		cmpl.Append(rt.GETVAR, rt.Bcode(vinfo.Index))
		itype := uint32(vinfo.Type)
		var outtype, subtype uint32
		for _, item := range nGetIndex.Indexes {
			outtype, subtype = parseType(itype)
			if err = nodeToCode(item, cmpl); err != nil {
				return err
			}
			cmdInd := rt.Bcode(rt.GETINDEX)
			switch outtype {
			case parser.VArr, parser.VBytes:
				if item.Result != parser.VInt {
					return cmpl.ErrorParam(node, errIndexInt, Type2Str(item.Result))
				}
			case parser.VMap:
				if item.Result != parser.VStr {
					return cmpl.ErrorParam(node, errIndexStr, Type2Str(item.Result))
				}
				cmdInd = rt.GETMAP
			default:
				return cmpl.ErrorParam(node, errIndexType, Type2Str(itype))
			}
			cmpl.Append(cmdInd)
			itype = subtype
		}
		node.Result = itype
	case parser.TSetIndex:
		nGetIndex := node.Value.(*parser.NGetIndex)
		name := nGetIndex.Name
		if vinfo, ok = cmpl.Contract.Vars[name]; !ok {
			return cmpl.ErrorParam(node, errVarUnknown, name)
		}
		cmpl.Append(rt.GETVAR, rt.Bcode(vinfo.Index))
		itype := uint32(vinfo.Type)
		retType := itype
		var outtype, subtype uint32
		for i, item := range nGetIndex.Indexes {
			outtype, subtype = parseType(itype)
			if err = nodeToCode(item, cmpl); err != nil {
				return err
			}
			var formap rt.Bcode
			switch outtype {
			case parser.VArr, parser.VBytes:
				if item.Result != parser.VInt {
					return cmpl.ErrorParam(node, errIndexInt, Type2Str(item.Result))
				}
			case parser.VMap:
				if item.Result != parser.VStr {
					return cmpl.ErrorParam(node, errIndexStr, Type2Str(item.Result))
				}
				formap = 2
			default:
				return cmpl.ErrorParam(node, errIndexType, Type2Str(itype))
			}
			if i == len(nGetIndex.Indexes)-1 {
				cmpl.Append(rt.SETINDEX + formap)
			} else {
				cmpl.Append(rt.GETINDEX + formap)
			}
			if i > 0 {
				retType = itype
			}
			itype = subtype
		}
		node.Result = retType
	case parser.TFor:
		if err = forCode(node, cmpl); err != nil {
			return err
		}
	case parser.TForInt:
		if err = forInt(node, cmpl); err != nil {
			return err
		}
	case parser.TBreak:
		if len(cmpl.Jumps) == 0 {
			return cmpl.Error(node, errBreak)
		}
		cmpl.Append(rt.JMP, 0)
		cmpl.Jumps[len(cmpl.Jumps)-1].Breaks = append(cmpl.Jumps[len(cmpl.Jumps)-1].Breaks,
			len(cmpl.Contract.Code)-1)
	case parser.TContinue:
		if len(cmpl.Jumps) == 0 {
			return cmpl.Error(node, errContinue)
		}
		cmpl.Append(rt.JMP, 0)
		cmpl.Jumps[len(cmpl.Jumps)-1].Continues = append(cmpl.Jumps[len(cmpl.Jumps)-1].Continues,
			len(cmpl.Contract.Code)-1)
	case parser.TEndLabel:
		for _, b := range cmpl.Jumps[len(cmpl.Jumps)-1].Continues {
			cmpl.Contract.Code[b] = rt.Bcode(len(cmpl.Contract.Code) - b + 1)
		}
		cmpl.Jumps[len(cmpl.Jumps)-1].Continues = nil
	case parser.TArray:
		var atype uint32
		nArray := node.Value.(*parser.NArray)
		for i, par := range nArray.List {
			if err = nodeToCode(par, cmpl); err != nil {
				return err
			}
			if i == 0 {
				atype = par.Result
			} else if atype != par.Result {
				return cmpl.ErrorParam(par, errParamType, Type2Str(atype))
			}
		}
		cmpl.Append(rt.INITARR, rt.Bcode(len(nArray.List)))
		node.Result = (atype << 4) | parser.VArr
	case parser.TMap:
		var atype uint32
		nMap := node.Value.(*parser.NMap)
		for i, par := range nMap.List {
			if cmpl.Data == nil {
				cmpl.Data = make([]byte, 0, 1024)
			}
			cmpl.Append(rt.PUSHSTR, rt.Bcode(len(cmpl.Data)), rt.Bcode(len(par.Key)))
			cmpl.Data = append(cmpl.Data, []byte(par.Key)...)
			if err = nodeToCode(par.Value, cmpl); err != nil {
				return err
			}
			if i == 0 {
				atype = par.Value.Result
			} else if atype != par.Value.Result {
				return cmpl.ErrorParam(par.Value, errParamType, Type2Str(atype))
			}
		}
		cmpl.Append(rt.INITMAP, rt.Bcode(len(nMap.List)))
		node.Result = (atype << 4) | parser.VMap
	case parser.TEnv:
		var (
			val rt.EnvItem
			ok  bool
		)
		nEnv := node.Value.(*parser.NEnv)
		if cmpl.Custom == nil {
			return cmpl.ErrorParam(node, errEnv, nEnv.Name)
		}
		if val, ok = cmpl.Custom.Env[nEnv.Name]; !ok {
			return cmpl.ErrorParam(node, errEnv, nEnv.Name)
		}
		cmpl.Append(rt.ENV, rt.Bcode(val.Index))
		node.Result = val.Type
	case parser.TObject:
		nObj := node.Value.(*parser.NObject)
		for _, par := range nObj.List {
			if cmpl.Data == nil {
				cmpl.Data = make([]byte, 0, 1024)
			}
			cmpl.Append(rt.PUSHSTR, rt.Bcode(len(cmpl.Data)), rt.Bcode(len(par.Key)))
			cmpl.Data = append(cmpl.Data, []byte(par.Key)...)
			if err = nodeToCode(par.Value, cmpl); err != nil {
				return err
			}
			cmpl.Append(rt.PUSH16, rt.Bcode(par.Value.Result))
		}
		cmpl.Append(rt.INITOBJ, rt.Bcode(len(nObj.List)))
		node.Result = parser.VObject
	case parser.TObjArr:
		nObjArr := node.Value.(*parser.NObjArr)
		for _, par := range nObjArr.List {
			if err = nodeToCode(par, cmpl); err != nil {
				return err
			}
			cmpl.Append(rt.PUSH16, rt.Bcode(par.Result))
		}
		cmpl.Append(rt.INITOBJLIST, rt.Bcode(len(nObjArr.List)))
		node.Result = parser.VObjList
	case parser.TObjList:
		nObjList := node.Value.(*parser.NObjList)
		if err = nodeToCode(nObjList.Obj, cmpl); err != nil {
			return err
		}
		cmpl.Append(rt.OBJ2LIST)
		node.Result = parser.VObject
	case parser.TSwitch:
		nSwitch := node.Value.(*parser.NSwitch)
		if err = nodeToCode(nSwitch.Expr, cmpl); err != nil {
			return err
		}
		result := nSwitch.Expr.Result
		if result != parser.VInt && result != parser.VStr && result != parser.VFloat {
			return cmpl.ErrorParam(nSwitch.Expr, errSwitchType, Type2Str(result))
		}
		cmp := rt.Bcode(rt.EQINT)
		if result == parser.VStr {
			cmp = rt.EQSTR
		} else if result == parser.VFloat {
			cmp = rt.EQFLOAT
		}
		var off rt.Bcode
		exps := make([]int, 0, 16)
		ends := make([]int, 0, 16)
		for _, icase := range nSwitch.Case.Value.(*parser.NCase).List {
			for _, iexpr := range icase.ExprList.Value.(*parser.NArray).List {
				cmpl.Append(rt.DUP)
				if err = nodeToCode(iexpr, cmpl); err != nil {
					return err
				}
				if iexpr.Result != result {
					return cmpl.ErrorTwoParam(nSwitch.Expr, errCaseType, Type2Str(iexpr.Result),
						Type2Str(result))
				}
				cmpl.Append(cmp)
				exps = append(exps, len(cmpl.Contract.Code))
				cmpl.Append(rt.JNZ, 0)
			}
			next := len(cmpl.Contract.Code)
			cmpl.Append(rt.JMPREL, 0)
			sizeCode := len(cmpl.Contract.Code)
			for _, eoff := range exps {
				if off, err = cmpl.JumpOff(node, sizeCode-eoff); err != nil {
					return err
				}
				cmpl.Contract.Code[eoff+1] = off
			}
			exps = exps[:0]
			if err = nodeToCode(icase.Body, cmpl); err != nil {
				return err
			}
			ends = append(ends, len(cmpl.Contract.Code))
			cmpl.Append(rt.JMPREL, 0)
			if off, err = cmpl.JumpOff(node, len(cmpl.Contract.Code)-next); err != nil {
				return err
			}
			cmpl.Contract.Code[next+1] = off
		}
		if nSwitch.Default != nil {
			if err = nodeToCode(nSwitch.Default, cmpl); err != nil {
				return err
			}
		}
		sizeCode := len(cmpl.Contract.Code)
		for _, eoff := range ends {
			if off, err = cmpl.JumpOff(node, sizeCode-eoff); err != nil {
				return err
			}
			cmpl.Contract.Code[eoff+1] = off
		}
	default:
		fmt.Println(`Ooops`)
		return cmpl.Error(node, errNodeType)
	}
	return nil
}

// Compile compiles contract
/*
编译合约生成字节码.
参数nameSpace和contracts是vm对象的参数： &vm.NameSpace, &vm.Contracts
*/
func Compile(input string, nameSpace *map[string]uint32, contracts *[]*rt.Contract,
	custom *rt.Custom) (*rt.Contract, error) {
	var root *parser.Node

	cmpl := &compiler{
		Contract: &rt.Contract{
			Code: make([]rt.Bcode, 0, 64),
			Vars: make(map[string]rt.VarInfo),
		},
		NameSpace: nameSpace,
		Contracts: contracts,
		Custom:    custom,
	}

	if len(*nameSpace) == 0 {
		initNameSpace(cmpl, nameSpace)
	}

	root, err := parser.Parser(input)
	if err != nil {
		return nil, err
	}
	defer func() {
		for i := 0; i < len(cmpl.Contract.Funcs); i++ {
			delete(*cmpl.NameSpace, getFuncKey(cmpl.Contract.Funcs[i]))
		}
	}()
	if err = nodeToCode(root, cmpl); err != nil {
		return nil, err
	}
	if len(cmpl.Data) > 0 {
		length := len(cmpl.Data)
		if length > 0xffff {
			return nil, cmpl.Error(root, errData)
		}
		if length&0x1 == 1 {
			cmpl.Data = append(cmpl.Data, 0)
			length++
		}
		length >>= 1
		data := make([]rt.Bcode, length+2)
		data[0] = rt.DATA
		data[1] = rt.Bcode(length)
		var off int
		for i := 0; i < length; i++ {
			data[i+2] = rt.Bcode(uint16(cmpl.Data[off])<<8 | uint16(cmpl.Data[off+1]))
			off += 2
		}
		cmpl.Contract.Code = append(data, cmpl.Contract.Code...)
	}
	return cmpl.Contract, nil
}
