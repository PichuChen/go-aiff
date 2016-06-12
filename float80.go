package aiff

// https://ja.wikipedia.org/wiki/%E6%8B%A1%E5%BC%B5%E5%80%8D%E7%B2%BE%E5%BA%A6%E6%B5%AE%E5%8B%95%E5%B0%8F%E6%95%B0%E7%82%B9%E6%95%B0
// https://en.wikipedia.org/wiki/Extended_precision

import (
	// "math/big"
	// "fmt"
	"math"
)

type float80 [10]byte

func (f *float80) Float64() float64 {
	// sign : 1  => 1
	// exp  : 15 => 11
	// frac : 63 => 52
	frac := make([]byte, 8)
	expb := make([]byte, 2)
	copy(frac, []byte(f[2:]))
	copy(expb, []byte(f[:2]))
	// fmt.Printf("Debug: frac: % X\n", frac)
	// fmt.Printf("Debug: expb: % X\n", expb)
	sign := expb[0] & 0x80
	expb[0] &= 0x7F
	exp := uint16(expb[0])<<8 | uint16(expb[1])
	exp = exp - 16383 + 1023

	// fmt.Printf("Debug: exp: % d\n", exp)
	frac[0] &= 0x7F
	newfrac := uint64(frac[0])<<56 | uint64(frac[1])<<48 | uint64(frac[2])<<40 | uint64(frac[3])<<32 |
		uint64(frac[4])<<24 | uint64(frac[5])<<16 | uint64(frac[6])<<8 | uint64(frac[7])

	// fmt.Printf("Debug: newfrac: % X\n", (newfrac))
	result := uint64(sign)<<56 | uint64(exp)<<52 | newfrac>>11
	// fmt.Printf("Debug: expb: % X\n", (result))
	return math.Float64frombits(result)

}
