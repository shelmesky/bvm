package runtime

import (
	"fmt"
	"strings"
)

// Replace replaces old substrings to new substrings
func Replace(rt *Runtime, s, old, new int64) int64 {
	rt.Strings = append(rt.Strings, strings.Replace(rt.Strings[s], rt.Strings[old],
		rt.Strings[new], -1))
	return int64(len(rt.Strings) - 1)
}

// Split splits the input string to array
func Split(rt *Runtime, input, sep int64) int64 {
	out := strings.Split(rt.Strings[input], rt.Strings[sep])
	result := make([]int64, len(out))
	for i, val := range out {
		rt.Strings = append(rt.Strings, val)
		result[i] = int64(len(rt.Strings) - 1)
	}
	rt.Objects = append(rt.Objects, result)
	return int64(len(rt.Objects) - 1)
}

// Substr returns a substring with the specified offset and length
func Substr(rt *Runtime, in, off, length int64) (int64, error) {
	var rin []rune
	rin = []rune(rt.Strings[in])
	rlen := int64(len(rin))
	if length < 0 {
		length = -length
		off -= length
	}
	if off < 0 || off >= rlen || off+length > rlen {
		return 0, fmt.Errorf(errInvalidParam)
	}
	if length == 0 {
		length = rlen - off
	}
	rt.Strings = append(rt.Strings, string(rin[off:off+length]))
	return int64(len(rt.Strings) - 1), nil
}

// Contains returns true if s contains substr string
func Contains(rt *Runtime, s, substr int64) int64 {
	if strings.Contains(rt.Strings[s], rt.Strings[substr]) {
		return 1
	}
	return 0
}

// Join is joining input with separator
func Join(rt *Runtime, arrid int64, sep int64) int64 {
	var ret string
	for i, item := range rt.Objects[arrid].([]int64) {
		if i > 0 {
			ret += rt.Strings[sep]
		}
		ret += rt.Strings[item]
	}
	rt.Strings = append(rt.Strings, ret)
	return int64(len(rt.Strings) - 1)
}

func HasPrefix(rt *Runtime, s, prefix int64) int64 {
	if strings.HasPrefix(rt.Strings[s], rt.Strings[prefix]) {
		return 1
	}
	return 0
}

func TrimSpace(rt *Runtime, s int64) int64 {
	rt.Strings = append(rt.Strings, strings.TrimSpace(rt.Strings[s]))
	return int64(len(rt.Strings) - 1)
}

func ToLower(rt *Runtime, s int64) int64 {
	rt.Strings = append(rt.Strings, strings.ToLower(rt.Strings[s]))
	return int64(len(rt.Strings) - 1)
}

func ToUpper(rt *Runtime, s int64) int64 {
	rt.Strings = append(rt.Strings, strings.ToUpper(rt.Strings[s]))
	return int64(len(rt.Strings) - 1)
}
