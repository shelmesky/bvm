package runtime

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/shopspring/decimal"

	"github.com/shelmesky/bvm/parser"
	"github.com/shelmesky/bvm/types"
)

type Bcode uint16

const (
	errDivZero      = `dividing by zero`
	errCommand      = `unknown command %d`
	errGasLimit     = `gas is over`
	errIndexOut     = `index out of range index:%d len:%d`
	errIndexMap     = `Key %s doesn't exist`
	errStr2Int      = `cannot convert %s to int`
	errGlobVar      = `global variable is undefined`
	errRetType      = `unsupported type of result value in %s`
	errFloatResult  = `incorrect float result`
	errInvalidParam = `invalid parameters`
	errTypeJSON     = `Value doesn't support json marshalling`
	errBytesVal     = `The byte value is greater than 255`
)

type objCount struct {
	Strings int
	Objects int
}

func DebugPrintf(formatString string, a ...interface{}) {
	Debug := true
	if Debug {
		fmt.Printf("vm execute: ")
		fmt.Printf(formatString, a...)
	}
}

// Run executes a bytecode
func (rt *Runtime) Run(contract *Contract, code []Bcode, params []int64, gasLimit int64) (string, int64, error) {
	var (
		i, top, gas, coff int64
		result            string
		data              []byte
		counts            []objCount
		isParContract     bool
	)
	length := int64(len(code))
	if length == 0 {
		return result, gas, nil
	}
	newCount := func() {
		counts = append(counts, objCount{
			Strings: len(rt.Strings),
			Objects: len(rt.Objects),
		})
	}
	delCount := func(full bool) {
		var last int
		if !full {
			last = len(counts) - 1
		}
		ocount := counts[last]
		rt.Strings = rt.Strings[:ocount.Strings]
		rt.Objects = rt.Objects[:ocount.Objects]
		counts = counts[:last]
	}
	newCount()
	defer delCount(true)
	Vars := make([]int64, 0, 1024)
	stack := make([]int64, 100) // 运行栈
	pars := make([]int64, 0, 32)
	calls := make([]int64, 1000)

	for _, value := range contract.VarsList {
		var v int64

		Type := value.Type & 0xf

		switch Type {
		case parser.VStr:
			rt.Strings = append(rt.Strings, ``)
			v = int64(len(rt.Strings) - 1)
			Vars = append(Vars, v)
		case parser.VInt:
			Vars = append(Vars, 0)
		default:
			rt.Strings = append(rt.Strings, ``)
			v = int64(len(rt.Strings) - 1)
		}

		switch Type {
		case parser.VStr:
			rt.Objects = append(rt.Objects, ``)
			v = int64(len(rt.Objects) - 1)
		case parser.VArr:
			rt.Objects = append(rt.Objects, []int64{}) // 空64位整形数组
			v = int64(len(rt.Objects) - 1)
			Vars = append(Vars, v)
		case parser.VMap:
			rt.Objects = append(rt.Objects, map[string]int64{}) // 空map
			v = int64(len(rt.Objects) - 1)
			Vars = append(Vars, v)
		case parser.VMoney:
			rt.Objects = append(rt.Objects, decimal.New(0, 0)) // 空的Money类型
			v = int64(len(rt.Objects) - 1)
			Vars = append(Vars, v)
		case parser.VBytes:
			rt.Objects = append(rt.Objects, []byte{}) // 空的字节数组类型
			v = int64(len(rt.Objects) - 1)
			Vars = append(Vars, v)
		case parser.VFile:
			rt.Objects = append(rt.Objects, types.NewFile()) //空的文件类型
			v = int64(len(rt.Objects) - 1)
			Vars = append(Vars, v)
		}
	}

	/*
		for idx, value := range contract.VarsList {
			var v int64
			fmt.Printf("init vars for: %d ", idx)
			Type := value.Type & 0xf
			switch Type { // code[i+2+iVar]保存的是需要初始化的变量的类型
			case parser.VStr:
				fmt.Printf("type: VStr    ")
				rt.Strings = append(rt.Strings, ``) // 空字符串
				v = int64(len(rt.Strings) - 1)
			case parser.VArr:
				fmt.Printf("type: VArr    ")
				rt.Objects = append(rt.Objects, []int64{}) // 空64位整形数组
				v = int64(len(rt.Objects) - 1)
			case parser.VMap:
				fmt.Printf("type: VMap    ")
				rt.Objects = append(rt.Objects, map[string]int64{}) // 空map
				v = int64(len(rt.Objects) - 1)
			case parser.VMoney:
				fmt.Printf("type: VMoney    ")
				rt.Objects = append(rt.Objects, decimal.New(0, 0)) // 空的Money类型
				v = int64(len(rt.Objects) - 1)
			case parser.VBytes:
				fmt.Printf("type: VBytes    ")
				rt.Objects = append(rt.Objects, []byte{}) // 空的字节数组类型
				v = int64(len(rt.Objects) - 1)
			case parser.VFile:
				fmt.Printf("type: VFile    ")
				rt.Objects = append(rt.Objects, types.NewFile()) //空的文件类型
				v = int64(len(rt.Objects) - 1)
			default:
				fmt.Printf("type: Int    ")
			}

			fmt.Printf("\n")

			Vars = append(Vars, v)
		}
	*/

	// top the latest value
	if code[0] == DATA {
		// 执行字节码之前，恢复保存在代码序列最前面的数据
		// data切片中保存的是字符串
		length := int64(uint64(code[1]))
		DebugPrintf("DATA    length: %d\n", length)
		data = make([]byte, length<<1)
		length += 2
		var off int
		for i = 2; i < length; i++ {
			data[off] = byte(code[i] >> 8)
			data[off+1] = byte(code[i] & 0xff)
			off += 2
		}
	}
main:
	// i起到指令指针的作用
	for i < length {
		gas++
		if gas > gasLimit {
			return ``, gas, fmt.Errorf(errGasLimit)
		}
		switch code[i] {
		case PUSH16: // 在栈顶保存16位数据
			/* code[i]保存的是指令本身 */
			i++                         // 指令指针+1，+1处保存的是操作数
			top++                       // 栈指针+1
			stack[top] = int64(code[i]) // 将code[i]处保存的值复制到栈顶
			DebugPrintf("PUSH16    %d\n", code[i])

		case PUSH32: // 在栈顶保存32位数据

			i += 2 // 单个字节码是16位， 所以指令指针+2
			top++
			stack[top] = int64((uint64(code[i-1]) << 16) | uint64(code[i]&0xffff))
			DebugPrintf("PUSH32    %d    %d\n", code[i-1], code[i])

		case PUSHSTR: // 从data中复制字符串，将字符串保存在runtime的String列表中，然后在栈顶保存字符串在列表中的索引
			// code[i+1]处保存的是字符串在data数组中的开始位置, code[i+2]保存的是结束位置
			rt.Strings = append(rt.Strings, string(data[code[i+1]:code[i+1]+code[i+2]]))
			DebugPrintf("PUSHSTR    start:%d    end: %d\n", code[i+1], code[i+1]+code[i+2])
			top++
			stack[top] = int64(len(rt.Strings) - 1) // 在栈顶保存字符串在rt.Strings切片中的索引
			i += 2                                  // 指令指针+2

		case INITVARS: // 初始化变量指令
			count := int64(code[i+1]) // 操作数为需要初始化的变量的数量
			DebugPrintf("INITVARS    count: %d    ", count)
			//			newCount()
			for iVar := int64(0); iVar < count; iVar++ {
				var v int64
				switch code[i+2+iVar] & 0xf { // code[i+2+iVar]保存的是需要初始化的变量的类型
				case parser.VStr:
					fmt.Printf("type: VStr    ")
					rt.Strings = append(rt.Strings, ``) // 空字符串
					v = int64(len(rt.Strings) - 1)
				case parser.VArr:
					fmt.Printf("type: VArr    ")
					rt.Objects = append(rt.Objects, []int64{}) // 空64位整形数组
					v = int64(len(rt.Objects) - 1)
				case parser.VMap:
					fmt.Printf("type: VMap    ")
					rt.Objects = append(rt.Objects, map[string]int64{}) // 空map
					v = int64(len(rt.Objects) - 1)
				case parser.VMoney:
					fmt.Printf("type: VMoney    ")
					rt.Objects = append(rt.Objects, decimal.New(0, 0)) // 空的Money类型
					v = int64(len(rt.Objects) - 1)
				case parser.VBytes:
					fmt.Printf("type: VBytes    ")
					rt.Objects = append(rt.Objects, []byte{}) // 空的字节数组类型
					v = int64(len(rt.Objects) - 1)
				case parser.VFile:
					fmt.Printf("type: VFile    ")
					rt.Objects = append(rt.Objects, types.NewFile()) //空的文件类型
					v = int64(len(rt.Objects) - 1)
				default:
					fmt.Printf("type: Int    ")
				}
				// v是已经初始化的对象在rt.Objects中保存的索引位置，将v保存在Vars数组中
				// 因为for循环挨个初始化在code数组中保存的变量类型，所以直接按照这个顺序将索引保存在Vars数组中
				Vars = append(Vars, v)
			}
			fmt.Printf("\n")
			i += count + 1

		case DELVARS:
			i++
			count := int64(code[i])
			Vars = Vars[:count]
			//delCount(false)
			DebugPrintf("DELVARS    count: %d\n", count)

		case ADDINT:
			top--
			stack[top] += stack[top+1]
			DebugPrintf("ADDINT\n")

		case SUBINT:
			top--
			stack[top] -= stack[top+1]
			DebugPrintf("SUBINT\n")

		case MULINT:
			top--
			stack[top] *= stack[top+1]
			DebugPrintf("MULINT\n")

		case DIVINT:
			top--
			if stack[top+1] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			stack[top] /= stack[top+1]
			DebugPrintf("DIVINT\n")

		case MODINT:
			top--
			if stack[top+1] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			stack[top] %= stack[top+1]
			DebugPrintf("MODINT\n")

		case EQINT:
			var b int64
			top--
			if stack[top] == stack[top+1] {
				b = 1
			}
			stack[top] = b
			DebugPrintf("EQINT\n")

		case LTINT:
			var b int64
			top--
			if stack[top] < stack[top+1] {
				b = 1
			}
			stack[top] = b
			DebugPrintf("LTINT\n")

		case GTINT:
			var b int64
			top--
			if stack[top] > stack[top+1] {
				b = 1
			}
			stack[top] = b
			DebugPrintf("GTINT\n")

		case AND:
			var b int64
			top--
			if stack[top] == 1 && stack[top+1] == 1 {
				b = 1
			}
			stack[top] = b
			DebugPrintf("AND\n")

		case OR:
			var b int64
			top--
			if stack[top] == 1 || stack[top+1] == 1 {
				b = 1
			}
			stack[top] = b
			DebugPrintf("OR\n")

		case DUP:
			top++
			stack[top] = stack[top-1]
			DebugPrintf("DUP\n")

		case GETVAR:

			i++
			top++
			a := code[i]
			DebugPrintf("GETVAR    Vars_index: %d\n", code[i])
			b := Vars[a]
			stack[top] = b

		case SETVAR:
			// 将code[i]值作为索引在Vars中寻找
			// 并将Vars[x]当作变量取其地址， 将地址放在栈顶
			// 本质是获取Vars数组中某个项的值
			i++
			top++
			/*
				a := code[i]
				if int(a) > len(Vars)-1 {
					DebugPrintf("SETVAR    code[i]:%d  Vars_Length:[%d]  index failed!!!\n", a, len(Vars))
					return ``, gas, fmt.Errorf("SETVAR index failed!\n")
				}
				b := Vars[a]
				DebugPrintf("SETVAR    code[i]:%d,  Vars[code[i]]:%d  Vars_length:[%d]\n", a, b, len(Vars))
				c := &b
				stack[top] = int64(uintptr(unsafe.Pointer(c)))
			*/

			a := code[i]
			b := Vars[a]
			DebugPrintf("SETVAR    code[i]:%d,  Vars[code[i]]:%d  Vars_length:[%d]\n", a, b, len(Vars))
			stack[top] = int64(uintptr(unsafe.Pointer(&Vars[code[i]])))

		case JMP:
			i += int64(int16(code[i+1]))
			top = 0
			DebugPrintf("JMP    label: %d\n", code[i+1])
			continue

		case JMPREL:
			i += int64(int16(code[i+1]))
			DebugPrintf("JMPREL    label: %d\n", code[i+1])
			continue

		case JZE:
			top--
			DebugPrintf("JZE    ")
			if stack[top+1] == 0 {
				DebugPrintf("label: %d", code[i+1])
				i += int64(int16(code[i+1]))
				continue
			}
			i++
			DebugPrintf("\n")

		case JNZ:
			top--
			DebugPrintf("JNZ    ")
			if stack[top+1] != 0 {
				DebugPrintf("label: %d", code[i+1])
				i += int64(int16(code[i+1]))
				continue
			}
			i++
			DebugPrintf("\n")

		case ASSIGNINT:
			DebugPrintf("ASSIGNINT\n")
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = stack[top]
			top -= 2

		case ASSIGNSTR:
			// TODO: 实现不完整
			DebugPrintf("ASSIGNSTR\n")
			rt.Strings = append(rt.Strings, rt.Strings[stack[top]])
			idx := stack[top-1]
			a := uintptr(idx)
			b := unsafe.Pointer(a)
			c := (*int64)(b)
			*c = int64(len(rt.Strings) - 1)
			top -= 2

		case ASSIGNADDINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) += stack[top]
			top -= 2
			DebugPrintf("ASSIGNADDINT\n")

		case ASSIGNSUBINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) -= stack[top]
			top -= 2
			DebugPrintf("ASSIGNSUBINT\n")

		case ASSIGNMULINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) *= stack[top]
			top -= 2
			DebugPrintf("ASSIGNMULINT\n")

		case ASSIGNDIVINT:
			if stack[top] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) /= stack[top]
			top -= 2
			DebugPrintf("ASSIGNDIVINT\n")

		case ASSIGNMODINT:
			if stack[top] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) %= stack[top]
			top -= 2
			DebugPrintf("ASSIGNMODINT\n")

		case CALLFUNC: // 函数调用
			calls[coff] = i + 2              // 在coff处将当前指令后的2条指令指针保存
			calls[coff+1] = int64(len(Vars)) //在coff+1处保存Vars数组的长度
			coff += 2                        // coff变量+2
			i += int64(int16(code[i+1]))     // 为函数调用修改变量指针地址
			DebugPrintf("CALLFUNC    IP:%d\n", i)
			continue

		case CUSTOMFUNC: // 调用用户自定义函数
			i++
			eFunc := rt.Funcs[code[i]]                    // runtime中保存的函数对象
			parCount := int64(len(eFunc.Params))          // 函数的参数数量
			parsFunc := make([]reflect.Value, parCount+1) // 用户保存参数的列表
			top -= parCount
			parsFunc[0] = reflect.ValueOf(rt.Data) // 第一个参数
			for k := int64(0); k < parCount; k++ { // 从栈中获取每个参数的索引
				val := stack[top+k+1]
				switch eFunc.Params[k] {
				case parser.VStr:
					parsFunc[k+1] = reflect.ValueOf(rt.Strings[val]) // 在字符串表中获取对象
				case parser.VObject, parser.VMoney:
					parsFunc[k+1] = reflect.ValueOf(rt.Objects[val]) // 在对象表中获取对象
				case parser.VFloat:
					parsFunc[k+1] = reflect.ValueOf(*(*float64)(unsafe.Pointer(&val))) // 解析浮点数
				case parser.VBool:
					var b bool
					if val != 0 {
						b = true
					}
					parsFunc[k+1] = reflect.ValueOf(b)
				default:
					parsFunc[k+1] = reflect.ValueOf(val)
				}
			}
			var result []reflect.Value
			result = reflect.ValueOf(eFunc.Func).Call(parsFunc) // 调用自定义函数
			gas -= result[len(result)-2].Interface().(int64)    // 从预定义的gas数量中减去消耗的
			last := result[len(result)-1].Interface()           // 检查函数执行是否返回错误
			if last != nil {
				if _, isError := last.(error); isError {
					return ``, gas, result[len(result)-1].Interface().(error)
				}
			}
			top++
			// 根据函数预定义的返回值类型， 将执行结果转换为对应的类型，再将值保存在栈顶
			switch eFunc.Result {
			case parser.VVoid:
			case parser.VInt:
				stack[top] = result[0].Interface().(int64)
			case parser.VBool:
				if result[0].Interface().(bool) {
					stack[top] = 1
				} else {
					stack[top] = 0
				}
			case parser.VStr:
				rt.Strings = append(rt.Strings, result[0].Interface().(string))
				stack[top] = int64(len(rt.Strings) - 1)
			default:
				return ``, gas, fmt.Errorf(errRetType, eFunc.Name)
			}

			DebugPrintf("CUSTOMFUNC    name: %s    args_count:%d\n", eFunc.Name, len(eFunc.Params))

		case EMBEDFUNC: // 内置函数调用
			i++
			eFunc := StdLib[code[i]]
			parsFunc := make([]reflect.Value, eFunc.Params+1)
			//			if Runtime {
			parsFunc[0] = reflect.ValueOf(rt)
			//		}*/
			top -= eFunc.Params
			for k := int64(0); k < eFunc.Params; k++ {
				parsFunc[k+1] = reflect.ValueOf(stack[top+k+1])
			}
			var result []reflect.Value
			result = reflect.ValueOf(eFunc.Func).Call(parsFunc)
			gas -= eFunc.Gas
			if len(result) > 0 {
				last := result[len(result)-1].Interface()
				if last != nil {
					if _, isError := last.(error); isError {
						return ``, gas, result[len(result)-1].Interface().(error)
					}
				}
				top++
				stack[top] = result[0].Interface().(int64)
			}
			DebugPrintf("EMBEDFUNC    name: %s    args_count:%d\n", eFunc.Name, eFunc.Params+1)

		case CALLCONTRACT: // 调用其他contract
			i++
			top++
			result, cgas, cerr := rt.Run((*rt.Contracts)[code[i]], (*rt.Contracts)[code[i]].Code, pars, gasLimit-gas)
			if isParContract {
				delCount(false)
				isParContract = false
			}

			pars = pars[:0]
			gas -= cgas
			if cerr != nil {
				return ``, gas, cerr
			}
			rt.Strings = append(rt.Strings, result)
			stack[top] = int64(len(rt.Strings) - 1)

			DebugPrintf("CALLCONTRACT    idx: %d    pars_count: %d\n", code[i], len(pars))

		case LOADPARS: // 从本函数的参数params中载入参数
			for j := 0; j < (len(params) >> 1); j++ {
				a := params[j<<1]
				b := (j << 1) + 1
				Vars[a] = params[b]
			}
			DebugPrintf("LOADPARS    pars_count: %d\n", len(params))

		case PARCONTRACT: // 载入合约参数
			if !isParContract {
				newCount()
				isParContract = true
			}
			i += 2
			switch code[i] & 0xf {
			case parser.VArr, parser.VMap, parser.VStr, parser.VMoney,
				parser.VBytes, parser.VFile: // Create a copy of the object
				stack[top] = copy(rt, int64(code[i]), stack[top])
			}
			pars = append(pars, int64(code[i-1]), stack[top])
			top--
			DebugPrintf("PARCONTRACT\n")

		case GETPARAMS:
			// code[i]是参数的数量
			// 将N个从栈复制到Vars数组中
			// 栈上保存的是调用函数前PUSH指令放到栈上的参数索引
			// 参数可能在
			i++
			count := int(code[i])

			DebugPrintf("GETPARAMS    count: %d\n", count)

			for j := 0; j < count; j++ {
				i++
				idx := code[i]
				Vars[idx] = stack[top]
				top--
			}
			/*
				for k := 1; k <= int(code[i]); k++ {
					a := len(Vars) - k
					Vars[a] = stack[top]
					top--
				}
			*/

		case RETURN:
			DebugPrintf("RETURN\n")
			result = print(rt, stack[top], int64(code[i+1]))
			break main

		case RETFUNC:
			DebugPrintf("RETFUNC\n")
			//a := coff - 1
			//b := calls[a]
			//Vars = Vars[:b] // 恢复Vars数组
			coff -= 2
			i = calls[coff] // 恢复指令指针
			continue

		case SIGNINT:
			stack[top] = -stack[top]
			DebugPrintf("SIGNINT\n")

		case NOT:
			if stack[top] == 0 {
				stack[top] = 1
			} else {
				stack[top] = 0
			}
			DebugPrintf("NOT\n")

		case ADDSTR:
			top--
			rt.Strings = append(rt.Strings, rt.Strings[stack[top]]+rt.Strings[stack[top+1]])
			stack[top] = int64(len(rt.Strings) - 1)
			DebugPrintf("ADDSTR\n")

		case EQSTR:
			top--
			if rt.Strings[stack[top]] == rt.Strings[stack[top+1]] {
				stack[top] = 1
			} else {
				stack[top] = 0
			}
			DebugPrintf("EQSTR\n")

		case ASSIGNADDSTR:
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Strings[ind] += rt.Strings[stack[top]]
			top -= 2
			DebugPrintf("ASSIGNADDSTR\n")

		case APPENDARR:
			a := stack[top-1]
			b := uintptr(a)
			c := (*int64)(unsafe.Pointer(b))
			ind := *c
			DebugPrintf("APPENDARR    index: %d    Objects length:%d\n", ind, len(rt.Objects))
			rt.Objects[ind] = append(rt.Objects[ind].([]int64), stack[top])
			top -= 2

		case GETINDEX:
			switch v := rt.Objects[stack[top-1]].(type) {
			case []int64:
				if stack[top] >= int64(len(v)) || stack[top] < 0 {
					return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(v))
				}
				stack[top-1] = v[stack[top]]
			case []uint8:
				if stack[top] >= int64(len(v)) || stack[top] < 0 {
					return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(v))
				}
				stack[top-1] = int64(v[stack[top]])
			}
			top--
			DebugPrintf("GETINDEX\n")

		case SETINDEX:
			idx := stack[top-1]
			DebugPrintf("SETINDEX    index:%d    Objects length: %d\n", idx, len(rt.Objects))
			switch v := rt.Objects[stack[top-1]].(type) {
			case []int64:
				if stack[top] >= int64(len(v)) || stack[top] < 0 {
					return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(v))
				}
			case []uint8:
				if stack[top] >= int64(len(v)) || stack[top] < 0 {
					return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(v))
				}
			}

		case GETMAP:
			imap := rt.Objects[stack[top-1]].(map[string]int64)
			if stack[top] >= int64(len(rt.Strings)) || stack[top] < 0 {
				return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(rt.Strings))
			}
			if val, ok := imap[rt.Strings[stack[top]]]; ok {
				stack[top-1] = val
			} else {
				return ``, gas, fmt.Errorf(errIndexMap, rt.Strings[stack[top]])
			}
			top--
			DebugPrintf("GETMAP\n")

		case SETMAP:
			if stack[top] >= int64(len(rt.Strings)) || stack[top] < 0 {
				return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(rt.Strings))
			}
			DebugPrintf("SETMAP\n")

		case COPYSTR:
			stack[top] = copy(rt, int64(parser.VStr), stack[top])
			DebugPrintf("COPYSTR\n")

		case COPY:
			i++
			stack[top] = copy(rt, int64(code[i]), stack[top])
			DebugPrintf("COPY\n")

		case ASSIGNSETMAP:
			imap := rt.Objects[stack[top-2]].(map[string]int64)
			imap[rt.Strings[stack[top-1]]] = stack[top]
			top -= 3
			DebugPrintf("ASSIGNSETMAP\n")

		case ASSIGNSETARR:
			iarr := rt.Objects[stack[top-2]].([]int64)
			iarr[stack[top-1]] = stack[top]
			top -= 3
			DebugPrintf("ASSIGNSETARR\n")

		case ASSIGNSETBYTES:
			ibyte := rt.Objects[stack[top-2]].([]uint8)
			if uint64(stack[top]) > 255 {
				return ``, gas, fmt.Errorf(errBytesVal)
			}
			ibyte[stack[top-1]] = uint8(stack[top])
			top -= 3
			DebugPrintf("ASSIGNSETBYTES\n")

		case INITARR:
			i++
			count := int64(code[i])
			iarr := make([]int64, count)
			for k := range iarr {
				iarr[k] = stack[top-count+int64(k)+1]
			}
			rt.Objects = append(rt.Objects, iarr)
			top -= count - 1
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("INITARR    count: %d\n", count)

		case INITMAP:
			i++
			count := int64(code[i])
			imap := make(map[string]int64)
			for k := int64(0); k < count; k++ {
				cur := top - 2*(count-k) + 1
				key := rt.Strings[stack[cur]]
				imap[key] = stack[cur+1]
			}
			rt.Objects = append(rt.Objects, imap)
			top -= 2*count - 1
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("INITMAP    count: %d\n", count)

		case INITOBJ:
			i++
			count := int64(code[i])
			imap := types.NewMap()
			for k := int64(0); k < count; k++ {
				cur := top - 3*(count-k) + 1
				key := rt.Strings[stack[cur]]
				switch stack[cur+2] {
				case parser.VObjList:
					imap.Set(key, rt.Objects[stack[cur+1]])
				case parser.VObject:
					imap.Set(key, rt.Objects[stack[cur+1]])
				default:
					imap.Set(key, print(rt, stack[cur+1], stack[cur+2]))
				}
			}
			rt.Objects = append(rt.Objects, imap)
			top -= 3*count - 1
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("INITOBJ    count: %d\n", count)

		case INITOBJLIST:
			i++
			count := int64(code[i])
			ilist := make([]interface{}, count)
			for k := int64(0); k < count; k++ {
				cur := top - 2*(count-k) + 1
				switch stack[cur+1] {
				case parser.VObjList:
					ilist[k] = rt.Objects[stack[cur]]
				case parser.VObject:
					ilist[k] = rt.Objects[stack[cur]]
				default:
					ilist[k] = print(rt, stack[cur], stack[cur+1])
				}
			}
			rt.Objects = append(rt.Objects, ilist)
			top -= 2*count - 1
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("INITOBJLIST    count: %d\n", count)

		case OBJ2LIST:
			obj := rt.Objects[stack[top]].(*types.Map)
			ilist := make([]interface{}, obj.Size())
			for k, key := range obj.Keys() {
				val, _ := obj.Get(key)
				ilist[k] = types.LoadMap(map[string]interface{}{
					key: val,
				})
			}
			rt.Objects = append(rt.Objects, ilist)
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("OBJ2LIST\n")

		case ENV:
			i++
			envVal := rt.Env[int64(code[i])]
			if !envVal.Init {
				return ``, gas, fmt.Errorf(errGlobVar)
			}
			top++
			stack[top] = envVal.Value
			DebugPrintf("ENV    index: %d\n", code[i])

		case PUSH64:
			i += 4
			top++
			stack[top] = int64((uint64(code[i-3]) << 48) | (uint64(code[i-2]) << 32) |
				(uint64(code[i-1]) << 16) | (uint64(code[i]) & 0xffff))
			DebugPrintf("PUSH64\n")

		case SIGNFLOAT:
			f := -*(*float64)(unsafe.Pointer(&stack[top]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
			DebugPrintf("SIGNFLOAT\n")

		case ADDFLOAT:
			top--
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f += *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
			DebugPrintf("ADDFLOAT\n")

		case SUBFLOAT:
			top--
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f -= *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
			DebugPrintf("SUBFLOAT\n")

		case MULFLOAT:
			top--
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f *= *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
			DebugPrintf("MULFLOAT\n")

		case DIVFLOAT:
			top--
			if stack[top+1] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f /= *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
			DebugPrintf("DIVLOAT\n")

		case ASSIGNADDFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			f += *(*float64)(unsafe.Pointer(&stack[top]))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
			DebugPrintf("ASSIGNADDFLOAT\n")

		case ASSIGNSUBFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			f -= *(*float64)(unsafe.Pointer(&stack[top]))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
			DebugPrintf("ASSIGNSUBFLOAT\n")

		case ASSIGNMULFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			f *= *(*float64)(unsafe.Pointer(&stack[top]))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
			DebugPrintf("ASSIGNMULFLOAT\n")

		case ASSIGNDIVFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			d := *(*float64)(unsafe.Pointer(&stack[top]))
			if d == 0.0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			f /= d
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
			DebugPrintf("ASSIGNDIVFLOAT\n")

		case EQFLOAT:
			var b int64
			top--
			if *(*float64)(unsafe.Pointer(&stack[top])) ==
				*(*float64)(unsafe.Pointer(&stack[top+1])) {
				b = 1
			}
			stack[top] = b
			DebugPrintf("EQFLOAT\n")

		case LTFLOAT:
			var b int64
			top--
			if *(*float64)(unsafe.Pointer(&stack[top])) <
				*(*float64)(unsafe.Pointer(&stack[top+1])) {
				b = 1
			}
			stack[top] = b
			DebugPrintf("LTFLOAT\n")

		case GTFLOAT:
			var b int64
			top--
			if *(*float64)(unsafe.Pointer(&stack[top])) >
				*(*float64)(unsafe.Pointer(&stack[top+1])) {
				b = 1
			}
			stack[top] = b
			DebugPrintf("GTFLOAT\n")

		case ADDMONEY:
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			rt.Objects = append(rt.Objects, d.Add(rt.Objects[stack[top+1]].(decimal.Decimal)))
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("ADDMONEY\n")

		case SUBMONEY:
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			rt.Objects = append(rt.Objects, d.Sub(rt.Objects[stack[top+1]].(decimal.Decimal)))
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("SUBMONEY\n")

		case SIGNMONEY:
			rt.Objects = append(rt.Objects, rt.Objects[stack[top]].(decimal.Decimal).Neg())
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("SIGNMONEY\n")

		case MULMONEY:
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			rt.Objects = append(rt.Objects, d.Mul(rt.Objects[stack[top+1]].(decimal.Decimal)))
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("MULMONEY\n")

		case DIVMONEY:
			top--
			d := rt.Objects[stack[top+1]].(decimal.Decimal)
			if d.IsZero() {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			rt.Objects = append(rt.Objects, rt.Objects[stack[top]].(decimal.Decimal).Div(d))
			stack[top] = int64(len(rt.Objects) - 1)
			DebugPrintf("DIVMONEY\n")

		case ASSIGNADDMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Add(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
			DebugPrintf("ASSIGNADDMONEY\n")

		case ASSIGNSUBMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Sub(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
			DebugPrintf("ASSIGNSUBMONEY\n")

		case ASSIGNMULMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Mul(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
			DebugPrintf("ASSIGNMULMONEY\n")

		case ASSIGNDIVMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.IsZero() {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Div(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
			DebugPrintf("ASSIGNDIVMONEY\n")

		case EQMONEY:
			var b int64
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.Equal(rt.Objects[stack[top+1]].(decimal.Decimal)) {
				b = 1
			}
			stack[top] = b
			DebugPrintf("EQMONEY\n")

		case LTMONEY:
			var b int64
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.LessThan(rt.Objects[stack[top+1]].(decimal.Decimal)) {
				b = 1
			}
			stack[top] = b
			DebugPrintf("LTMONEY\n")

		case GTMONEY:
			var b int64
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.GreaterThan(rt.Objects[stack[top+1]].(decimal.Decimal)) {
				b = 1
			}
			stack[top] = b
			DebugPrintf("GTMONEY\n")

		case ASSIGNADDBYTES:
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects[ind] = append(rt.Objects[ind].([]byte),
				rt.Objects[stack[top]].([]byte)...)
			top -= 2
			DebugPrintf("ASSIGNADDBYTES\n")

		default:
			return ``, gas, fmt.Errorf(errCommand, code[i])
		}
		i++
	}
	return result, gas, nil
}
