// asmcheck

// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codegen

func andn64(x, y int64) int64 {
	// amd64/v3:"ANDNQ"
	return x &^ y
}

func andn32(x, y int32) int32 {
	// amd64/v3:"ANDNL"
	return x &^ y
}

func blsi64(x int64) int64 {
	// amd64/v3:"BLSIQ"
	return x & -x
}

func blsi32(x int32) int32 {
	// amd64/v3:"BLSIL"
	return x & -x
}

func blsmsk64(x int64) int64 {
	// amd64/v3:"BLSMSKQ"
	return x ^ (x - 1)
}

func blsmsk32(x int32) int32 {
	// amd64/v3:"BLSMSKL"
	return x ^ (x - 1)
}

func blsr64(x int64) int64 {
	// amd64/v3:"BLSRQ"
	return x & (x - 1)
}

func blsr32(x int32) int32 {
	// amd64/v3:"BLSRL"
	return x & (x - 1)
}

func sarx64(x, y int64) int64 {
	// amd64/v3:"SARXQ"
	return x >> y
}

func sarx32(x, y int32) int32 {
	// amd64/v3:"SARXL"
	return x >> y
}

func shlrx64(x []uint64, i int, s uint64) uint64 {
	// amd64/v3: `SHRXQ\t[A-Z]+[0-9]*, \([A-Z]+[0-9]*\)\([A-Z]+[0-9]*\*8\), [A-Z]+[0-9]*`
	s = x[i] >> i
	// amd64/v3: `SHLXQ\t[A-Z]+[0-9]*, 8\([A-Z]+[0-9]*\)\([A-Z]+[0-9]*\*8\), [A-Z]+[0-9]*`
	s = x[i+1] << s
	return s
}

func shlrx32(x []uint32, i int, s uint32) uint32 {
	// amd64/v3: `SHRXL\t[A-Z]+[0-9]*, \([A-Z]+[0-9]*\)\([A-Z]+[0-9]*\*4\), [A-Z]+[0-9]*`
	s = x[i] >> i
	// amd64/v3: `SHLXL\t[A-Z]+[0-9]*, 4\([A-Z]+[0-9]*\)\([A-Z]+[0-9]*\*4\), [A-Z]+[0-9]*`
	s = x[i+1] << s
	return s
}
