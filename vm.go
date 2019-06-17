package simvolio

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/shelmesky/bvm/compiler"
	"github.com/shelmesky/bvm/parser"
	"github.com/shelmesky/bvm/runtime"
	"github.com/shelmesky/bvm/types"

	"github.com/shopspring/decimal"
)

const (
	DEFAULT_GAS_LIMIT = 2000000000

	errCntExists    = `Contract %s has already been defined`
	errCntNotExists = `Contract %s doesn't exists`
	errGlobVar      = `Wrong count of global variables`
	errGlobType     = `%s has unsupported type`
)

const (
	Void    = iota
	Int     // int
	Bool    // bool
	Str     // str
	Arr     // arr
	Map     // map
	Float   // float
	Money   // money
	Object  // object
	Bytes   // bytes
	File    // file
	ObjList // list in object
)

type FuncItem struct {
	Name   string
	Params []uint32
	Result uint32
	Read   bool
	Func   interface{}
}

type EnvItem struct {
	Name string
	Type uint32
}

type VMSettings struct {
	Funcs    []FuncItem
	Env      []EnvItem
	GasLimit int64
}

// VM is a virtual machine structure
type VM struct {
	Contracts []*runtime.Contract
	NameSpace map[string]uint32 // common namespace
	Settings  VMSettings
	Custom    *runtime.Custom
}

// NewVM creates a new virtual machine
func NewVM(settings VMSettings) *VM {
	if settings.GasLimit == 0 {
		settings.GasLimit = DEFAULT_GAS_LIMIT
	}
	env := make(map[string]runtime.EnvItem)
	for i, val := range settings.Env {
		env[val.Name] = runtime.EnvItem{
			Index: i,
			Type:  val.Type,
		}
	}
	funcs := make([]runtime.FuncItem, len(settings.Funcs))
	for i, val := range settings.Funcs {
		funcs[i] = runtime.FuncItem{
			Result: val.Result,
			Params: val.Params,
			Name:   val.Name,
			Read:   val.Read,
			Func:   val.Func,
		}
	}

	return &VM{
		Contracts: make([]*runtime.Contract, 0, 1000),
		NameSpace: make(map[string]uint32),
		Settings:  settings,
		Custom: &runtime.Custom{
			Env:   env,
			Funcs: funcs,
		},
	}
}

// Compile compiles the contract and returns its structure
func (vm *VM) Compile(input string) (cnt *runtime.Contract, err error) {
	return compiler.Compile(input, &vm.NameSpace, &vm.Contracts, vm.Custom)
}

// GetContract returns the contract by its name
func (vm *VM) GetContract(name string) *runtime.Contract {
	if ind, ok := vm.NameSpace[name]; ok {
		return vm.Contracts[ind]
	}
	return nil
}

// Link links the compiled contract to VM
func (vm *VM) Link(cnt *runtime.Contract, reload bool) error {
	var (
		ind uint32
		ok  bool
	)
	if ind, ok = vm.NameSpace[cnt.Name]; ok && !reload {
		return fmt.Errorf(errCntExists, cnt.Name)
	} else if !ok && reload {
		return fmt.Errorf(errCntNotExists, cnt.Name)
	}
	if reload {
		vm.Contracts[ind] = cnt
	} else {
		vm.Contracts = append(vm.Contracts, cnt)
		ind = uint32(len(vm.Contracts) - 1)
	}
	vm.NameSpace[cnt.Name] = ind
	return nil
}

// LoadContract compiles and link the contract
func (vm *VM) LoadContract(input string, id int64) error {
	cnt, err := vm.Compile(input)
	if err != nil {
		return err
	}
	if err = vm.Link(cnt, false); err != nil {
		return err
	}
	cnt.ID = id
	return nil
}

// Run executes the contract
func (vm *VM) Run(cnt *runtime.Contract, data runtime.IData) (string, int64, error) {
	rt := runtime.NewRuntime(&vm.Contracts)
	env := make([]runtime.EnvVal, len(vm.Custom.Env))
	envData := data.GetEnv()
	if len(envData) != len(vm.Custom.Env) {
		return ``, 0, fmt.Errorf(errGlobVar)
	}

	for i, val := range envData {
		var (
			vEnv int64
		)
		switch v := val.(type) {
		case int64:
			vEnv = v
		case int:
			vEnv = int64(v)
		case string:
			rt.Strings = append(rt.Strings, v)
			vEnv = int64(len(rt.Strings) - 1)
		default:
			var name string
			for key, eVal := range vm.Custom.Env {
				if eVal.Index == i {
					name = key
					break
				}
			}
			return ``, 0, fmt.Errorf(errGlobType, name)
		}
		env[i] = runtime.EnvVal{
			Value: vEnv,
			Init:  true,
		}
	}
	rt.Env = env
	rt.Data = data
	rt.Funcs = vm.Custom.Funcs
	params := make([]int64, 0)
	for key, vi := range cnt.Params {
		var (
			err error
			val int64
			v   interface{}
		)
		if v = data.GetParam(key); v == nil {
			continue
		}
		switch vVal := v.(type) {
		case string:
			switch vi.Type {
			case parser.VInt:
				val, err = strconv.ParseInt(vVal, 10, 64)
			case parser.VStr:
				rt.Strings = append(rt.Strings, vVal)
				val = int64(len(rt.Strings) - 1)
			case parser.VFloat:
				var f float64
				f, err = strconv.ParseFloat(vVal, 64)
				if err == nil {
					val = *(*int64)(unsafe.Pointer(&f))
				}
			case parser.VBool:
				if vVal == `0` || len(vVal) == 0 || vVal == `false` {
					val = 0
				} else {
					val = 1
				}
			case parser.VMoney:
				var d decimal.Decimal
				d, err = decimal.NewFromString(vVal)
				if err != nil {
					return ``, 0, err
				}
				rt.Objects = append(rt.Objects, d.Floor())
				val = int64(len(rt.Objects) - 1)
			case parser.VBytes:
				var b []byte
				b, err = hex.DecodeString(vVal)
				if err == nil {
					rt.Objects = append(rt.Objects, b)
					val = int64(len(rt.Objects) - 1)
				}
			default:
				return ``, 0, fmt.Errorf(`Unsupported type of parameter`)
			}
		case []byte:
			switch vi.Type {
			case parser.VBytes:
				rt.Objects = append(rt.Objects, vVal)
				val = int64(len(rt.Objects) - 1)
			default:
				return ``, 0, fmt.Errorf(`Unsupported type of parameter`)
			}
		case *types.File:
			switch vi.Type {
			case parser.VFile:
				rt.Objects = append(rt.Objects, vVal)
				val = int64(len(rt.Objects) - 1)
			default:
				return ``, 0, fmt.Errorf(`Unsupported type of parameter`)
			}
		default:
			err = fmt.Errorf(`Params must have string or []bytes type`)
		}
		if err != nil {
			return ``, 0, err
		}
		params = append(params, int64(vi.Index))
		params = append(params, val)
	}
	return rt.Run(cnt.Code, params, vm.Settings.GasLimit)
}

// RunByName executes the contract
func (vm *VM) RunByName(name string, data runtime.IData) (string, int64, error) {
	cnt := vm.GetContract(name)
	if cnt == nil {
		return ``, 0, fmt.Errorf(errCntNotExists, name)
	}
	return vm.Run(cnt, data)
}
