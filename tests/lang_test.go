package test

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/shelmesky/bvm"
	"github.com/shelmesky/bvm/runtime"
	"github.com/shelmesky/bvm/types"
)

type contract struct {
	Source string
	Line   int    // the start line of the contract
	Gas    int64  // expecting gas
	Result string // expecting result
}

func (cnt *contract) check(gas int64, get string) error {
	if get != cnt.Result {
		return fmt.Errorf("Line %d: get != want;\n%s !=\n%s", cnt.Line, get, cnt.Result)
	}
	if cnt.Gas > 0 && gas != cnt.Gas {
		return fmt.Errorf("Line %d: got gas %d != %d", cnt.Line, gas, cnt.Gas)
	}
	return nil
}

func (cnt *contract) checkError(err error) error {
	if err.Error() != cnt.Result {
		return fmt.Errorf("Line %d: get != want;\n%s !=\n%s", cnt.Line, err.Error(), cnt.Result)
	}
	return nil
}

func loadTest(filename string) (ret []*contract, err error) {
	var (
		input []byte
		start int
	)
	input, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	ret = make([]*contract, 0, 1000)

	list := strings.Split(string(input), "\n")
	source := make([]string, 0, 32)
	re := regexp.MustCompile(`====\s*(\d+)\s*\$(.*)`)

	for i, line := range list {
		if strings.HasPrefix(line, `====`) {
			var (
				gas    int64
				result string
			)
			if re.MatchString(line) {
				match := re.FindSubmatch([]byte(line))
				if gas, err = strconv.ParseInt(strings.TrimSpace(string(match[1])), 10, 64); err != nil {
					return
				}
				result = string(match[2])
			} else {
				result = line[4:]
			}
			ret = append(ret, &contract{
				Source: strings.Join(source, "\r\n"),
				Line:   start + 1,
				Gas:    gas,
				Result: strings.Replace(strings.TrimSpace(result), `\n`, "\n", -1),
			})
			source = source[:0]
			start = i + 1
		} else {
			source = append(source, line)
		}
	}

	return
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

func testFile(filename string) error {
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
		},
	})
	contracts, err := loadTest(filename)
	if err != nil {
		return err
	}
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
		},
	}
	for i := int64(len(contracts)) - 1; i >= 0; i-- {
		cnt := contracts[i]
		if err = vm.LoadContract(cnt.Source, i); err != nil {
			if err = cnt.checkError(err); err != nil {
				return err
			}
			continue
		}
		//		fmt.Println(`I`, cnt.Line)
		result, gas, err := vm.Run(vm.Contracts[len(vm.Contracts)-1], data)
		if err != nil {
			if err = cnt.checkError(err); err != nil {
				return err
			}
		} else if err = cnt.check(gas, result); err != nil {
			return err
		}
	}
	return nil
}

func TestLang(t *testing.T) {
	if err := testFile(`default_test`); err != nil {
		t.Error(err)
		return
	}
	t.Error(`OK`)
}

func TestHard(t *testing.T) {
	if err := testFile(`hard_test`); err != nil {
		t.Error(err)
		return
	}
}
