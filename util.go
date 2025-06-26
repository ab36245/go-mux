package mux

import "fmt"

func readNumber(bytes []byte) (uint, []byte, error) {
	var n uint = 0
	s := 0
	for i := range bytes {
		b := uint(bytes[i])
		n |= (b & 0x7f) << s
		s += 7
		if b&0x80 == 0x00 {
			return n, bytes[i+1:], nil
		}
	}
	return 0, nil, fmt.Errorf("incomplete number")
}

func writeNumber(n uint) []byte {
	bytes := make([]byte, 0, 4)
	for {
		b := byte(n & 0x7F)
		n >>= 7
		if n > 0 {
			b |= 0x80
		}
		bytes = append(bytes, b)
		if n == 0 {
			break
		}
	}
	return bytes
}
