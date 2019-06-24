package main

import (
	"encoding/binary"
	"fmt"
	"github.com/shelmesky/bvm"
	"github.com/shelmesky/bvm/runtime"
	"github.com/shelmesky/bvm/types"
	"github.com/shopspring/decimal"
	"github.com/vmihailenco/msgpack"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	vmConfig = simvolio.VMSettings{
		GasLimit: 200000000,
		Env: []simvolio.EnvItem{
			{Name: `block`, Type: simvolio.Int},
			{Name: `ecosystem`, Type: simvolio.Int},
			{Name: `key`, Type: simvolio.Str},
		},
		Funcs: []simvolio.FuncItem{
			{Func: readFunc, Name: `readFunc`, Read: true,
				Params: []uint32{simvolio.Str, simvolio.Int}, Result: simvolio.Str},
			{Func: testFunc, Name: `testFunc`, Params: []uint32{simvolio.Str, simvolio.Int}, Result: simvolio.Str},
			{Func: fbmFunc, Name: `fbmFunc`, Params: []uint32{simvolio.Float, simvolio.Bool, simvolio.Money},
				Result: simvolio.Str},
			{Func: voidFunc, Name: `voidFunc`, Params: []uint32{simvolio.Str}},
			{Func: objFunc, Name: `objFunc`, Params: []uint32{simvolio.Object}, Result: simvolio.Str},
			{Func: printFunc, Name: `println`, Params: []uint32{simvolio.Str}},
		},
	}
)

func printFunc(data runtime.IData, s string) (int64, error) {
	fmt.Println("contract:", s)
	return 0, nil
}

func testFunc(data runtime.IData, s string, i int64) (string, int64, error) {
	return s + fmt.Sprint(i), 100, nil
}

func voidFunc(data runtime.IData, s string) (int64, error) {
	return 20, fmt.Errorf(`errorVoidFunc`)
}

func objFunc(data runtime.IData, obj *types.Map) (string, int64, error) {
	return fmt.Sprint(obj), 20, nil
}

func readFunc(data runtime.IData, s string, i int64) (string, int64, error) {
	return s + `=` + fmt.Sprint(i), 50, nil
}

func fbmFunc(data runtime.IData, f float64, b bool, m decimal.Decimal) (string, int64, error) {
	return fmt.Sprintf("%v*%v*%v", f, b, m), 100, nil
}

type myData struct {
	Env    []interface{}
	Params map[string]interface{}
}

func (data myData) GetEnv() []interface{} {
	return data.Env
}

func (data myData) GetParam(name string) interface{} {
	return data.Params[name]
}

func printUsage() {
	fmt.Printf("usage: %s [compile | run] filename\n", os.Args[0])
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 {
		printUsage()
	}

	arg1 := os.Args[1]

	if arg1 == "compile" {
		inputFilename := os.Args[2]
		filenameSplit := strings.Split(inputFilename, ".")
		outputFilename := filenameSplit[0] + ".bvm"
		Compile(inputFilename, outputFilename)
		os.Exit(0)
	}

	if arg1 == "run" {
		bytecodeFilename := os.Args[2]
		Run(bytecodeFilename)
		os.Exit(0)
	}

	printUsage()
}

func Compile(inputFilename, outputFilename string) {
	if len(inputFilename) == 0 {
		fmt.Println("need filename")
		os.Exit(1)
	}

	vm := simvolio.NewVM(vmConfig)

	content, err := ioutil.ReadFile(inputFilename)
	if err != nil {
		log.Fatal("ReadFile failed:", err)
	}

	list := strings.Split(string(content), "\n")
	source := make([]string, 0, 32)

	for _, line := range list {
		source = append(source, line)
	}

	contractBody := strings.Join(source, "\r\n")

	err = vm.LoadContract(contractBody, 0)
	if err != nil {
		log.Fatal("LoadContract failed:", err)
	}

	// 序列化合约代码
	contractBuffer, err := msgpack.Marshal(vm.Contracts[0])
	if err != nil {
		fmt.Println("encode contract failed:", err)
	}

	// 序列化命名空间
	// TODO: 不在输出文件中保存namespace?
	/* 只能保存一份固定的? */
	namespaceBuffer, err := msgpack.Marshal(vm.NameSpace)
	if err != nil {
		fmt.Println("encode namespace failed:")
	}

	// 打开编译输出文件
	outputFile, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0664)
	if err != nil {
		fmt.Println("open file for write failed:", err)
		os.Exit(1)
	}

	// 写入合约代码长度
	contractBufferLen := len(contractBuffer)
	lenBuffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuffer, uint32(contractBufferLen))
	n, err := outputFile.Write(lenBuffer)
	if n != 4 || err != nil {
		fmt.Println("write contract code length failed:", err, n)
		os.Exit(1)
	}

	// 写入合约代码
	n, err = outputFile.Write(contractBuffer)
	if n != contractBufferLen || err != nil {
		fmt.Println("write contract code failed:", err, n)
		os.Exit(1)
	}

	// 写入命名空间长度
	namespaceBufferLen := len(namespaceBuffer)
	lenBuffer = make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuffer, uint32(namespaceBufferLen))
	n, err = outputFile.Write(lenBuffer)
	if n != 4 || err != nil {
		fmt.Println("write namespace length failed:", err, n)
		os.Exit(1)
	}

	// 写入命名空间
	n, err = outputFile.Write(namespaceBuffer)
	if n != namespaceBufferLen || err != nil {
		fmt.Println("write namespace code failed:", err, n)
		os.Exit(1)
	}

	err = outputFile.Close()
	if err != nil {
		fmt.Println("close output file failed:", err)
		os.Exit(1)
	}
}

func Run(bytecodeFilename string) {
	vm := simvolio.NewVM(vmConfig)

	bytecodeBody, err := ioutil.ReadFile(bytecodeFilename)
	if err != nil {
		fmt.Println("read bytecode file failed")
		os.Exit(1)
	}

	contractLen := binary.LittleEndian.Uint32(bytecodeBody[:4])
	contractBuf := bytecodeBody[4 : 4+contractLen]
	namespaceLen := binary.LittleEndian.Uint32(bytecodeBody[4+contractLen : 4+4+contractLen])
	namespaceBuf := bytecodeBody[4+4+contractLen : 4+4+contractLen+namespaceLen]

	var cnt runtime.Contract
	err = msgpack.Unmarshal(contractBuf, &cnt)
	if err != nil {
		fmt.Println("unmarshal contract failed:", err)
		os.Exit(1)
	}

	var nameSpace map[string]uint32
	err = msgpack.Unmarshal(namespaceBuf, &nameSpace)
	if err != nil {
		fmt.Println("unmarshal namespace failed:", err)
		os.Exit(1)
	}

	//fmt.Println(cnt)
	//fmt.Println(nameSpace)

	vm.NameSpace = nameSpace
	vm.Contracts = append(vm.Contracts, &cnt)
	ind := uint32(len(vm.Contracts) - 1)
	vm.NameSpace[cnt.Name] = ind

	// 指定给合约的参数， key是参数名称
	data := myData{
		Env: []interface{}{7, 1, `0122afcd34`},
		Params: map[string]interface{}{
			`pInt`:   "123",
			`pStr`:   `OK`,
			`pMoney`: `32562365237623`,
			`pBool`:  `false`,
			`pFloat`: `23.834`,
			`pBytes`: `31325f`,
			`bBytes`: []byte{33, 39, 0x5b, 0},
			`fFile`:  types.FileInit(`myfile.txt`, `text`, []byte{45, 47, 00, 32}),

			`s1`: `s1s1s1s1s1s1s1`,
		},
	}

	contract0 := vm.Contracts[0]

	result, gas, err := vm.Run(contract0, data)
	if err != nil {
		log.Fatal("vm.Run failed:", err)
	}

	fmt.Printf("\nvm.Run: [result: %s], [gas: %d], [error: %v]\n", result, gas, err)

	os.Exit(0)
}
