package sortorder

import (
	"fmt"
	"strings"
)

// Base62Digits is the digit set used by the fractional-indexing algorithm.
// Compatible with the npm fractional-indexing package.
const Base62Digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func midpoint(a, b string) string {
	zero := Base62Digits[0]
	if b != "" && a >= b {
		panic(fmt.Sprintf("%s >= %s", a, b))
	}
	if len(a) > 0 && a[len(a)-1] == zero {
		panic("trailing zero")
	}
	if len(b) > 0 && b[len(b)-1] == zero {
		panic("trailing zero")
	}
	if b != "" {
		n := 0
		for n < len(b) {
			aChar := zero
			if n < len(a) {
				aChar = a[n]
			}
			if aChar == b[n] {
				n++
				continue
			}
			break
		}
		if n > 0 {
			aSlice := ""
			if n < len(a) {
				aSlice = a[n:]
			}
			return b[:n] + midpoint(aSlice, b[n:])
		}
	}
	digitA := 0
	if len(a) > 0 {
		digitA = strings.IndexByte(Base62Digits, a[0])
	}
	digitB := len(Base62Digits)
	if b != "" {
		digitB = strings.IndexByte(Base62Digits, b[0])
	}
	if digitB-digitA > 1 {
		midDigit := (digitA + digitB + 1) / 2
		return string(Base62Digits[midDigit])
	}
	if len(b) > 1 {
		return b[:1]
	}
	aRest := ""
	if len(a) > 1 {
		aRest = a[1:]
	}
	return string(Base62Digits[digitA]) + midpoint(aRest, "")
}

func getIntegerLength(head byte) int {
	if head >= 'a' && head <= 'z' {
		return int(head-'a') + 2
	}
	if head >= 'A' && head <= 'Z' {
		return int('Z'-head) + 2
	}
	panic(fmt.Sprintf("invalid order key head: %c", head))
}

func getIntegerPart(key string) string {
	intLen := getIntegerLength(key[0])
	if intLen > len(key) {
		panic(fmt.Sprintf("invalid order key: %s", key))
	}
	return key[:intLen]
}

func validateInteger(x string) {
	if len(x) != getIntegerLength(x[0]) {
		panic(fmt.Sprintf("invalid integer part of order key: %s", x))
	}
}

func validateOrderKey(key string) {
	if key == "A"+strings.Repeat(string(Base62Digits[0]), 26) {
		panic(fmt.Sprintf("invalid order key: %s", key))
	}
	i := getIntegerPart(key)
	f := key[len(i):]
	if len(f) > 0 && f[len(f)-1] == Base62Digits[0] {
		panic(fmt.Sprintf("invalid order key: %s", key))
	}
}

func incrementInteger(x string) (string, bool) {
	validateInteger(x)
	head := x[0]
	digs := []byte(x[1:])
	carry := true
	for i := len(digs) - 1; carry && i >= 0; i-- {
		d := strings.IndexByte(Base62Digits, digs[i]) + 1
		if d == len(Base62Digits) {
			digs[i] = Base62Digits[0]
		} else {
			digs[i] = Base62Digits[d]
			carry = false
		}
	}
	if carry {
		if head == 'Z' {
			return "a" + string(Base62Digits[0]), true
		}
		if head == 'z' {
			return "", false
		}
		h := head + 1
		if h > 'a' {
			digs = append(digs, Base62Digits[0])
		} else {
			digs = digs[:len(digs)-1]
		}
		return string(h) + string(digs), true
	}
	return string(head) + string(digs), true
}

func decrementInteger(x string) (string, bool) {
	validateInteger(x)
	head := x[0]
	digs := []byte(x[1:])
	borrow := true
	for i := len(digs) - 1; borrow && i >= 0; i-- {
		d := strings.IndexByte(Base62Digits, digs[i]) - 1
		if d == -1 {
			digs[i] = Base62Digits[len(Base62Digits)-1]
		} else {
			digs[i] = Base62Digits[d]
			borrow = false
		}
	}
	if borrow {
		if head == 'a' {
			return "Z" + string(Base62Digits[len(Base62Digits)-1]), true
		}
		if head == 'A' {
			return "", false
		}
		h := head - 1
		if h < 'Z' {
			digs = append(digs, Base62Digits[len(Base62Digits)-1])
		} else {
			digs = digs[:len(digs)-1]
		}
		return string(h) + string(digs), true
	}
	return string(head) + string(digs), true
}

// GenerateKeyBetween generates a fractional-indexing key between a and b.
// Use empty string to represent no bound (start/end of range).
func GenerateKeyBetween(a, b string) (string, error) {
	if a != "" {
		validateOrderKey(a)
	}
	if b != "" {
		validateOrderKey(b)
	}
	if a != "" && b != "" && a >= b {
		return "", fmt.Errorf("%s >= %s", a, b)
	}
	if a == "" {
		if b == "" {
			return "a" + string(Base62Digits[0]), nil
		}
		ib := getIntegerPart(b)
		fb := b[len(ib):]
		if ib == "A"+strings.Repeat(string(Base62Digits[0]), 26) {
			return ib + midpoint("", fb), nil
		}
		if ib < b {
			return ib, nil
		}
		res, ok := decrementInteger(ib)
		if !ok {
			return "", fmt.Errorf("cannot decrement any more")
		}
		return res, nil
	}
	if b == "" {
		ia := getIntegerPart(a)
		fa := a[len(ia):]
		i, ok := incrementInteger(ia)
		if !ok {
			return ia + midpoint(fa, ""), nil
		}
		return i, nil
	}
	ia := getIntegerPart(a)
	fa := a[len(ia):]
	ib := getIntegerPart(b)
	fb := b[len(ib):]
	if ia == ib {
		return ia + midpoint(fa, fb), nil
	}
	i, ok := incrementInteger(ia)
	if !ok {
		return "", fmt.Errorf("cannot increment any more")
	}
	if i < b {
		return i, nil
	}
	return ia + midpoint(fa, ""), nil
}

// GenerateNKeysBetween generates n keys between a and b.
// Use empty string to represent no bound.
func GenerateNKeysBetween(a, b string, n int) ([]string, error) {
	if n == 0 {
		return []string{}, nil
	}
	if n == 1 {
		key, err := GenerateKeyBetween(a, b)
		if err != nil {
			return nil, err
		}
		return []string{key}, nil
	}
	if b == "" {
		c, err := GenerateKeyBetween(a, b)
		if err != nil {
			return nil, err
		}
		result := []string{c}
		for i := 0; i < n-1; i++ {
			c, err = GenerateKeyBetween(c, b)
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
		return result, nil
	}
	if a == "" {
		c, err := GenerateKeyBetween(a, b)
		if err != nil {
			return nil, err
		}
		result := []string{c}
		for i := 0; i < n-1; i++ {
			c, err = GenerateKeyBetween(a, c)
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
		return result, nil
	}
	mid := n / 2
	c, err := GenerateKeyBetween(a, b)
	if err != nil {
		return nil, err
	}
	left, err := GenerateNKeysBetween(a, c, mid)
	if err != nil {
		return nil, err
	}
	right, err := GenerateNKeysBetween(c, b, n-mid-1)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, n)
	result = append(result, left...)
	result = append(result, c)
	result = append(result, right...)
	return result, nil
}
