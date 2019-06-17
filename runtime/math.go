package runtime

import (
	"fmt"
	"math"
	"unsafe"
)

func isValidFloat(x float64) bool {
	return !(math.IsNaN(x) || math.IsInf(x, 1) || math.IsInf(x, -1))
}

// Floor returns the greatest integer value less than or equal to x
func Floor(rt *Runtime, i int64) (int64, error) {
	f := *(*float64)(unsafe.Pointer(&i))
	if f = math.Floor(f); isValidFloat(f) {
		return int64(f), nil
	}
	return 0, fmt.Errorf(errFloatResult)
}

// Log returns the natural logarithm of x
func Log(rt *Runtime, i int64) (int64, error) {
	f := *(*float64)(unsafe.Pointer(&i))
	if f = math.Log(f); isValidFloat(f) {
		return *(*int64)(unsafe.Pointer(&f)), nil
	}
	return 0, fmt.Errorf(errFloatResult)
}

// Log10 returns the decimal logarithm of x
func Log10(rt *Runtime, i int64) (int64, error) {
	f := *(*float64)(unsafe.Pointer(&i))
	if f = math.Log10(f); isValidFloat(f) {
		return *(*int64)(unsafe.Pointer(&f)), nil
	}
	return 0, fmt.Errorf(errFloatResult)
}

// Pow returns x**y, the base-x exponential of y
func Pow(rt *Runtime, x, y int64) (int64, error) {
	fx := *(*float64)(unsafe.Pointer(&x))
	fy := *(*float64)(unsafe.Pointer(&y))
	if fx = math.Pow(fx, fy); isValidFloat(fx) {
		return *(*int64)(unsafe.Pointer(&fx)), nil
	}
	return 0, fmt.Errorf(errFloatResult)
}

// Round returns the nearest integer, rounding half away from zero
func Round(rt *Runtime, x int64) (int64, error) {
	fx := *(*float64)(unsafe.Pointer(&x))
	if fx = math.Round(fx); isValidFloat(fx) {
		return int64(fx), nil
	}
	return 0, fmt.Errorf(errFloatResult)
}

// Sqrt returns the square root of x
func Sqrt(rt *Runtime, x int64) (int64, error) {
	fx := *(*float64)(unsafe.Pointer(&x))
	if fx = math.Sqrt(fx); isValidFloat(fx) {
		return *(*int64)(unsafe.Pointer(&fx)), nil
	}
	return 0, fmt.Errorf(errFloatResult)
}
