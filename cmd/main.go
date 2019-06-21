package main

import (
	"encoding/json"
	"fmt"
	simvolio "github.com/shelmesky/bvm"
	"github.com/shelmesky/bvm/runtime"
	"github.com/shelmesky/bvm/types"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

func main() {
	filename := os.Args[1]
	if len(filename) == 0 {
		fmt.Println("need filename")
		os.Exit(1)
	}

	vm := simvolio.NewVM(simvolio.VMSettings{
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
	})

	content, err := ioutil.ReadFile(filename)
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

	buffer, err := json.Marshal(contract0)
	if err != nil {
		fmt.Println("encode contract failed:", err)
	}

	err = ioutil.WriteFile("test.bvm", buffer, 0664)
	if err != nil {
		fmt.Println()
	}

	result, gas, err := vm.Run(contract0, data)
	if err != nil {
		log.Fatal("vm.Run failed:", err)
	}

	fmt.Printf("\nvm.Run: [result: %s], [gas: %d], [error: %v]\n", result, gas, err)

	os.Exit(0)
}
