package runtime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/shelmesky/bvm/types"
)

func JSONDecode(rt *Runtime, s int64) (int64, error) {
	var err error
	input := []byte(rt.Strings[s])
	var v map[string]interface{}
	if err = json.Unmarshal(input, &v); err != nil {
		return 0, err
	}
	ret := types.ConvertMap(v)
	rt.Objects = append(rt.Objects, ret)
	return int64(len(rt.Objects) - 1), nil
}

// JSONEncodeIdent converts object to json string
func JSONEncodeIndent(rt *Runtime, s, ind int64) (int64, error) {
	input := rt.Objects[s]
	indent := rt.Strings[ind]
	rv := reflect.ValueOf(input)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct && reflect.TypeOf(input).String() != `*types.Map` {
		return 0, fmt.Errorf(errTypeJSON)
	}
	var (
		b   []byte
		err error
	)
	if len(indent) == 0 {
		b, err = json.Marshal(input)
	} else {
		b, err = json.MarshalIndent(input, ``, indent)
	}
	if err != nil {
		return 0, err
	}
	out := string(b)
	out = strings.Replace(out, `\u003c`, `<`, -1)
	out = strings.Replace(out, `\u003e`, `>`, -1)
	out = strings.Replace(out, `\u0026`, `&`, -1)

	rt.Strings = append(rt.Strings, out)
	return int64(len(rt.Strings) - 1), nil
}

// JSONEncode converts object to json string
func JSONEncode(rt *Runtime, obj int64) (int64, error) {
	return JSONEncodeIndent(rt, obj, 0)
}

func IsExists(rt *Runtime, obj, key int64) int64 {
	omap := rt.Objects[obj].(*types.Map)
	if _, found := omap.Get(rt.Strings[key]); found {
		return 1
	}
	return 0
}

func IsString(rt *Runtime, obj, key int64) int64 {
	omap := rt.Objects[obj].(*types.Map)
	if val, found := omap.Get(rt.Strings[key]); found {
		if _, ok := val.(string); ok {
			return 1
		}
	}
	return 0
}

func IsArray(rt *Runtime, obj, key int64) int64 {
	omap := rt.Objects[obj].(*types.Map)
	if val, found := omap.Get(rt.Strings[key]); found {
		if _, ok := val.([]interface{}); ok {
			return 1
		}
	}
	return 0
}

func IsMap(rt *Runtime, obj, key int64) int64 {
	omap := rt.Objects[obj].(*types.Map)
	if val, found := omap.Get(rt.Strings[key]); found {
		if _, ok := val.(*types.Map); ok {
			return 1
		}
	}
	return 0
}

func GetString(rt *Runtime, obj, key int64) int64 {
	omap := rt.Objects[obj].(*types.Map)
	if val, found := omap.Get(rt.Strings[key]); found {
		rt.Strings = append(rt.Strings, fmt.Sprint(val))
		return int64(len(rt.Strings) - 1)
	}
	return 0
}

func GetArray(rt *Runtime, obj, key int64) int64 {
	ret := make([]int64, 0, 8)
	omap := rt.Objects[obj].(*types.Map)
	if val, found := omap.Get(rt.Strings[key]); found {
		switch v := val.(type) {
		case []interface{}:
			for _, item := range v {
				rt.Strings = append(rt.Strings, fmt.Sprint(item))
				ret = append(ret, int64(len(rt.Strings)-1))
			}
		default:
			ret = append(ret, GetString(rt, obj, key))
		}
	}
	rt.Objects = append(rt.Objects, ret)
	return int64(len(rt.Objects) - 1)
}

func GetMap(rt *Runtime, obj, key int64) int64 {
	ret := make(map[string]int64)
	omap := rt.Objects[obj].(*types.Map)
	if val, found := omap.Get(rt.Strings[key]); found {

		switch v := val.(type) {
		case *types.Map:
			for _, ikey := range v.Keys() {
				item, _ := v.Get(ikey)
				rt.Strings = append(rt.Strings, fmt.Sprint(item))
				ret[ikey] = int64(len(rt.Strings) - 1)
			}
		}
	}
	rt.Objects = append(rt.Objects, ret)
	return int64(len(rt.Objects) - 1)
}
