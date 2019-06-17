package runtime

import (
	"encoding/hex"
	//	"fmt"
	//	"strings"
)

// LenBytes returns the length of the bytes
func LenBytes(rt *Runtime, i int64) int64 {
	return int64(len(rt.Objects[i].([]byte)))
}

// BytesStr converts a string to bytes
func BytesStr(rt *Runtime, i int64) int64 {
	rt.Objects = append(rt.Objects, []byte(rt.Strings[i]))
	return int64(len(rt.Objects) - 1)
}

// HexBytes encodes buf to hex string
func HexBytes(rt *Runtime, i int64) int64 {
	rt.Strings = append(rt.Strings, hex.EncodeToString(rt.Objects[i].([]byte)))
	return int64(len(rt.Strings) - 1)
}

// UnHexBytes decodes hex string to the bytes
func UnHexBytes(rt *Runtime, i int64) (int64, error) {
	b, err := hex.DecodeString(rt.Strings[i])
	if err != nil {
		return 0, err
	}
	rt.Objects = append(rt.Objects, b)
	return int64(len(rt.Objects) - 1), nil
}
