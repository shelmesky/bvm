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

// Run executes a bytecode
func (rt *Runtime) Run(code []Bcode, params []int64, gasLimit int64) (string, int64, error) {
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
	Vars := make([]int64, 0, 32)
	stack := make([]int64, 100)
	pars := make([]int64, 0, 32)
	calls := make([]int64, 1000)

	// top the latest value
	if code[0] == DATA {
		length := int64(uint64(code[1]))
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
	for i < length {
		gas++
		if gas > gasLimit {
			return ``, gas, fmt.Errorf(errGasLimit)
		}
		switch code[i] {
		case PUSH16:
			i++
			top++
			stack[top] = int64(code[i])
		case PUSH32:
			i += 2
			top++
			stack[top] = int64((uint64(code[i-1]) << 16) | uint64(code[i]&0xffff))
		case PUSHSTR:
			rt.Strings = append(rt.Strings, string(data[code[i+1]:code[i+1]+code[i+2]]))
			top++
			stack[top] = int64(len(rt.Strings) - 1)
			i += 2
		case INITVARS:
			count := int64(code[i+1])
			//			newCount()
			for iVar := int64(0); iVar < count; iVar++ {
				var v int64
				switch code[i+2+iVar] & 0xf {
				case parser.VStr:
					rt.Strings = append(rt.Strings, ``)
					v = int64(len(rt.Strings) - 1)
				case parser.VArr:
					rt.Objects = append(rt.Objects, []int64{})
					v = int64(len(rt.Objects) - 1)
				case parser.VMap:
					rt.Objects = append(rt.Objects, map[string]int64{})
					v = int64(len(rt.Objects) - 1)
				case parser.VMoney:
					rt.Objects = append(rt.Objects, decimal.New(0, 0))
					v = int64(len(rt.Objects) - 1)
				case parser.VBytes:
					rt.Objects = append(rt.Objects, []byte{})
					v = int64(len(rt.Objects) - 1)
				case parser.VFile:
					rt.Objects = append(rt.Objects, types.NewFile())
					v = int64(len(rt.Objects) - 1)
				}
				Vars = append(Vars, v)
			}
			i += count + 1
		case DELVARS:
			i++
			count := int64(code[i])
			Vars = Vars[:count]
			//delCount(false)
		case ADDINT:
			top--
			stack[top] += stack[top+1]
		case SUBINT:
			top--
			stack[top] -= stack[top+1]
		case MULINT:
			top--
			stack[top] *= stack[top+1]
		case DIVINT:
			top--
			if stack[top+1] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			stack[top] /= stack[top+1]
		case MODINT:
			top--
			if stack[top+1] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			stack[top] %= stack[top+1]
		case EQINT:
			var b int64
			top--
			if stack[top] == stack[top+1] {
				b = 1
			}
			stack[top] = b
		case LTINT:
			var b int64
			top--
			if stack[top] < stack[top+1] {
				b = 1
			}
			stack[top] = b
		case GTINT:
			var b int64
			top--
			if stack[top] > stack[top+1] {
				b = 1
			}
			stack[top] = b
		case AND:
			var b int64
			top--
			if stack[top] == 1 && stack[top+1] == 1 {
				b = 1
			}
			stack[top] = b
		case OR:
			var b int64
			top--
			if stack[top] == 1 || stack[top+1] == 1 {
				b = 1
			}
			stack[top] = b
		case DUP:
			top++
			stack[top] = stack[top-1]
		case GETVAR:
			i++
			top++
			stack[top] = Vars[code[i]]
		case SETVAR:
			i++
			top++
			stack[top] = int64(uintptr(unsafe.Pointer(&Vars[code[i]])))
		case JMP:
			i += int64(int16(code[i+1]))
			top = 0
			continue
		case JMPREL:
			i += int64(int16(code[i+1]))
			continue
		case JZE:
			top--
			if stack[top+1] == 0 {
				i += int64(int16(code[i+1]))
				continue
			}
			i++
		case JNZ:
			top--
			if stack[top+1] != 0 {
				i += int64(int16(code[i+1]))
				continue
			}
			i++
		case ASSIGNINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = stack[top]
			top -= 2
		case ASSIGNSTR:
			rt.Strings = append(rt.Strings, rt.Strings[stack[top]])
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Strings) - 1)
			top -= 2
		case ASSIGNADDINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) += stack[top]
			top -= 2
		case ASSIGNSUBINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) -= stack[top]
			top -= 2
		case ASSIGNMULINT:
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) *= stack[top]
			top -= 2
		case ASSIGNDIVINT:
			if stack[top] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) /= stack[top]
			top -= 2
		case ASSIGNMODINT:
			if stack[top] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) %= stack[top]
			top -= 2
		case CALLFUNC:
			calls[coff] = i + 2
			calls[coff+1] = int64(len(Vars))
			coff += 2
			i += int64(int16(code[i+1]))
			continue
		case CUSTOMFUNC:
			i++
			eFunc := rt.Funcs[code[i]]
			parCount := int64(len(eFunc.Params))
			parsFunc := make([]reflect.Value, parCount+1)
			top -= parCount
			parsFunc[0] = reflect.ValueOf(rt.Data)
			for k := int64(0); k < parCount; k++ {
				val := stack[top+k+1]
				switch eFunc.Params[k] {
				case parser.VStr:
					parsFunc[k+1] = reflect.ValueOf(rt.Strings[val])
				case parser.VObject, parser.VMoney:
					parsFunc[k+1] = reflect.ValueOf(rt.Objects[val])
				case parser.VFloat:
					parsFunc[k+1] = reflect.ValueOf(*(*float64)(unsafe.Pointer(&val)))
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
			result = reflect.ValueOf(eFunc.Func).Call(parsFunc)
			gas -= result[len(result)-2].Interface().(int64)
			last := result[len(result)-1].Interface()
			if last != nil {
				if _, isError := last.(error); isError {
					return ``, gas, result[len(result)-1].Interface().(error)
				}
			}
			top++
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
		case EMBEDFUNC:
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
		case CALLCONTRACT:
			i++
			top++
			result, cgas, cerr := rt.Run((*rt.Contracts)[code[i]].Code, pars, gasLimit-gas)
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
		case LOADPARS:
			for j := 0; j < (len(params) >> 1); j++ {
				Vars[params[j<<1]] = params[(j<<1)+1]
			}
		case PARCONTRACT:
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
		case GETPARAMS:
			i++
			for k := 1; k <= int(code[i]); k++ {
				Vars[len(Vars)-k] = stack[top]
				top--
			}
		case RETURN:
			result = print(rt, stack[top], int64(code[i+1]))
			break main
		case RETFUNC:
			Vars = Vars[:calls[coff-1]]
			coff -= 2
			i = calls[coff]
			continue
		case SIGNINT:
			stack[top] = -stack[top]
		case NOT:
			if stack[top] == 0 {
				stack[top] = 1
			} else {
				stack[top] = 0
			}
		case ADDSTR:
			top--
			rt.Strings = append(rt.Strings, rt.Strings[stack[top]]+rt.Strings[stack[top+1]])
			stack[top] = int64(len(rt.Strings) - 1)
		case EQSTR:
			top--
			if rt.Strings[stack[top]] == rt.Strings[stack[top+1]] {
				stack[top] = 1
			} else {
				stack[top] = 0
			}
		case ASSIGNADDSTR:
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Strings[ind] += rt.Strings[stack[top]]
			top -= 2
		case APPENDARR:
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
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
		case SETINDEX:
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
		case SETMAP:
			if stack[top] >= int64(len(rt.Strings)) || stack[top] < 0 {
				return ``, gas, fmt.Errorf(errIndexOut, stack[top], len(rt.Strings))
			}
		case COPYSTR:
			stack[top] = copy(rt, int64(parser.VStr), stack[top])
		case COPY:
			i++
			stack[top] = copy(rt, int64(code[i]), stack[top])
		case ASSIGNSETMAP:
			imap := rt.Objects[stack[top-2]].(map[string]int64)
			imap[rt.Strings[stack[top-1]]] = stack[top]
			top -= 3
		case ASSIGNSETARR:
			iarr := rt.Objects[stack[top-2]].([]int64)
			iarr[stack[top-1]] = stack[top]
			top -= 3
		case ASSIGNSETBYTES:
			ibyte := rt.Objects[stack[top-2]].([]uint8)
			if uint64(stack[top]) > 255 {
				return ``, gas, fmt.Errorf(errBytesVal)
			}
			ibyte[stack[top-1]] = uint8(stack[top])
			top -= 3
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
		case ENV:
			i++
			envVal := rt.Env[int64(code[i])]
			if !envVal.Init {
				return ``, gas, fmt.Errorf(errGlobVar)
			}
			top++
			stack[top] = envVal.Value
		case PUSH64:
			i += 4
			top++
			stack[top] = int64((uint64(code[i-3]) << 48) | (uint64(code[i-2]) << 32) |
				(uint64(code[i-1]) << 16) | (uint64(code[i]) & 0xffff))
		case SIGNFLOAT:
			f := -*(*float64)(unsafe.Pointer(&stack[top]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
		case ADDFLOAT:
			top--
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f += *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
		case SUBFLOAT:
			top--
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f -= *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
		case MULFLOAT:
			top--
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f *= *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
		case DIVFLOAT:
			top--
			if stack[top+1] == 0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			f := *(*float64)(unsafe.Pointer(&stack[top]))
			f /= *(*float64)(unsafe.Pointer(&stack[top+1]))
			stack[top] = *(*int64)(unsafe.Pointer(&f))
		case ASSIGNADDFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			f += *(*float64)(unsafe.Pointer(&stack[top]))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
		case ASSIGNSUBFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			f -= *(*float64)(unsafe.Pointer(&stack[top]))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
		case ASSIGNMULFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			f *= *(*float64)(unsafe.Pointer(&stack[top]))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
		case ASSIGNDIVFLOAT:
			f := *(*float64)(unsafe.Pointer(uintptr(stack[top-1])))
			d := *(*float64)(unsafe.Pointer(&stack[top]))
			if d == 0.0 {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			f /= d
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = *(*int64)(unsafe.Pointer(&f))
			top -= 2
		case EQFLOAT:
			var b int64
			top--
			if *(*float64)(unsafe.Pointer(&stack[top])) ==
				*(*float64)(unsafe.Pointer(&stack[top+1])) {
				b = 1
			}
			stack[top] = b
		case LTFLOAT:
			var b int64
			top--
			if *(*float64)(unsafe.Pointer(&stack[top])) <
				*(*float64)(unsafe.Pointer(&stack[top+1])) {
				b = 1
			}
			stack[top] = b
		case GTFLOAT:
			var b int64
			top--
			if *(*float64)(unsafe.Pointer(&stack[top])) >
				*(*float64)(unsafe.Pointer(&stack[top+1])) {
				b = 1
			}
			stack[top] = b
		case ADDMONEY:
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			rt.Objects = append(rt.Objects, d.Add(rt.Objects[stack[top+1]].(decimal.Decimal)))
			stack[top] = int64(len(rt.Objects) - 1)
		case SUBMONEY:
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			rt.Objects = append(rt.Objects, d.Sub(rt.Objects[stack[top+1]].(decimal.Decimal)))
			stack[top] = int64(len(rt.Objects) - 1)
		case SIGNMONEY:
			rt.Objects = append(rt.Objects, rt.Objects[stack[top]].(decimal.Decimal).Neg())
			stack[top] = int64(len(rt.Objects) - 1)
		case MULMONEY:
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			rt.Objects = append(rt.Objects, d.Mul(rt.Objects[stack[top+1]].(decimal.Decimal)))
			stack[top] = int64(len(rt.Objects) - 1)
		case DIVMONEY:
			top--
			d := rt.Objects[stack[top+1]].(decimal.Decimal)
			if d.IsZero() {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			rt.Objects = append(rt.Objects, rt.Objects[stack[top]].(decimal.Decimal).Div(d))
			stack[top] = int64(len(rt.Objects) - 1)
		case ASSIGNADDMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Add(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
		case ASSIGNSUBMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Sub(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
		case ASSIGNMULMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Mul(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
		case ASSIGNDIVMONEY:
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.IsZero() {
				return ``, gas, fmt.Errorf(errDivZero)
			}
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects = append(rt.Objects, rt.Objects[ind].(decimal.Decimal).Div(d))
			*(*int64)(unsafe.Pointer(uintptr(stack[top-1]))) = int64(len(rt.Objects) - 1)
			top -= 2
		case EQMONEY:
			var b int64
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.Equal(rt.Objects[stack[top+1]].(decimal.Decimal)) {
				b = 1
			}
			stack[top] = b
		case LTMONEY:
			var b int64
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.LessThan(rt.Objects[stack[top+1]].(decimal.Decimal)) {
				b = 1
			}
			stack[top] = b
		case GTMONEY:
			var b int64
			top--
			d := rt.Objects[stack[top]].(decimal.Decimal)
			if d.GreaterThan(rt.Objects[stack[top+1]].(decimal.Decimal)) {
				b = 1
			}
			stack[top] = b
		case ASSIGNADDBYTES:
			ind := *(*int64)(unsafe.Pointer(uintptr(stack[top-1])))
			rt.Objects[ind] = append(rt.Objects[ind].([]byte),
				rt.Objects[stack[top]].([]byte)...)
			top -= 2
		default:
			return ``, gas, fmt.Errorf(errCommand, code[i])
		}
		i++
	}
	return result, gas, nil
}
