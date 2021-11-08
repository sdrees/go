// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package nistec implements the NIST P elliptic curves from FIPS 186-4.
//
// This package uses fiat-crypto for its backend field arithmetic (not math/big)
// and exposes constant-time, heap allocation-free, byte slice-based safe APIs.
// Group operations use modern and safe complete addition formulas. The point at
// infinity is handled and encoded according to SEC 1, Version 2.0, and invalid
// curve points can't be represented.
package nistec

import (
	"crypto/elliptic/internal/fiat"
	"crypto/subtle"
	"errors"
)

var p521B, _ = new(fiat.P521Element).SetBytes([]byte{
	0x00, 0x51, 0x95, 0x3e, 0xb9, 0x61, 0x8e, 0x1c, 0x9a, 0x1f, 0x92, 0x9a,
	0x21, 0xa0, 0xb6, 0x85, 0x40, 0xee, 0xa2, 0xda, 0x72, 0x5b, 0x99, 0xb3,
	0x15, 0xf3, 0xb8, 0xb4, 0x89, 0x91, 0x8e, 0xf1, 0x09, 0xe1, 0x56, 0x19,
	0x39, 0x51, 0xec, 0x7e, 0x93, 0x7b, 0x16, 0x52, 0xc0, 0xbd, 0x3b, 0xb1,
	0xbf, 0x07, 0x35, 0x73, 0xdf, 0x88, 0x3d, 0x2c, 0x34, 0xf1, 0xef, 0x45,
	0x1f, 0xd4, 0x6b, 0x50, 0x3f, 0x00})

var p521G, _ = NewP521Point().SetBytes([]byte{0x04,
	0x00, 0xc6, 0x85, 0x8e, 0x06, 0xb7, 0x04, 0x04, 0xe9, 0xcd, 0x9e, 0x3e,
	0xcb, 0x66, 0x23, 0x95, 0xb4, 0x42, 0x9c, 0x64, 0x81, 0x39, 0x05, 0x3f,
	0xb5, 0x21, 0xf8, 0x28, 0xaf, 0x60, 0x6b, 0x4d, 0x3d, 0xba, 0xa1, 0x4b,
	0x5e, 0x77, 0xef, 0xe7, 0x59, 0x28, 0xfe, 0x1d, 0xc1, 0x27, 0xa2, 0xff,
	0xa8, 0xde, 0x33, 0x48, 0xb3, 0xc1, 0x85, 0x6a, 0x42, 0x9b, 0xf9, 0x7e,
	0x7e, 0x31, 0xc2, 0xe5, 0xbd, 0x66, 0x01, 0x18, 0x39, 0x29, 0x6a, 0x78,
	0x9a, 0x3b, 0xc0, 0x04, 0x5c, 0x8a, 0x5f, 0xb4, 0x2c, 0x7d, 0x1b, 0xd9,
	0x98, 0xf5, 0x44, 0x49, 0x57, 0x9b, 0x44, 0x68, 0x17, 0xaf, 0xbd, 0x17,
	0x27, 0x3e, 0x66, 0x2c, 0x97, 0xee, 0x72, 0x99, 0x5e, 0xf4, 0x26, 0x40,
	0xc5, 0x50, 0xb9, 0x01, 0x3f, 0xad, 0x07, 0x61, 0x35, 0x3c, 0x70, 0x86,
	0xa2, 0x72, 0xc2, 0x40, 0x88, 0xbe, 0x94, 0x76, 0x9f, 0xd1, 0x66, 0x50})

const p521ElementLength = 66

// P521Point is a P-521 point. The zero value is NOT valid.
type P521Point struct {
	// The point is represented in projective coordinates (X:Y:Z),
	// where x = X/Z and y = Y/Z.
	x, y, z *fiat.P521Element
}

// NewP521Point returns a new P521Point representing the point at infinity point.
func NewP521Point() *P521Point {
	return &P521Point{
		x: new(fiat.P521Element),
		y: new(fiat.P521Element).One(),
		z: new(fiat.P521Element),
	}
}

// NewP521Generator returns a new P521Point set to the canonical generator.
func NewP521Generator() *P521Point {
	return (&P521Point{
		x: new(fiat.P521Element),
		y: new(fiat.P521Element),
		z: new(fiat.P521Element),
	}).Set(p521G)
}

// Set sets p = q and returns p.
func (p *P521Point) Set(q *P521Point) *P521Point {
	p.x.Set(q.x)
	p.y.Set(q.y)
	p.z.Set(q.z)
	return p
}

// SetBytes sets p to the compressed, uncompressed, or infinity value encoded in
// b, as specified in SEC 1, Version 2.0, Section 2.3.4. If the point is not on
// the curve, it returns nil and an error, and the receiver is unchanged.
// Otherwise, it returns p.
func (p *P521Point) SetBytes(b []byte) (*P521Point, error) {
	switch {
	// Point at infinity.
	case len(b) == 1 && b[0] == 0:
		return p.Set(NewP521Point()), nil

	// Uncompressed form.
	case len(b) == 1+2*p521ElementLength && b[0] == 4:
		x, err := new(fiat.P521Element).SetBytes(b[1 : 1+p521ElementLength])
		if err != nil {
			return nil, err
		}
		y, err := new(fiat.P521Element).SetBytes(b[1+p521ElementLength:])
		if err != nil {
			return nil, err
		}
		if err := p521CheckOnCurve(x, y); err != nil {
			return nil, err
		}
		p.x.Set(x)
		p.y.Set(y)
		p.z.One()
		return p, nil

	// Compressed form
	case len(b) == 1+p521ElementLength && b[0] == 0:
		return nil, errors.New("unimplemented") // TODO(filippo)

	default:
		return nil, errors.New("invalid P521 point encoding")
	}
}

func p521CheckOnCurve(x, y *fiat.P521Element) error {
	// x³ - 3x + b.
	x3 := new(fiat.P521Element).Square(x)
	x3.Mul(x3, x)

	threeX := new(fiat.P521Element).Add(x, x)
	threeX.Add(threeX, x)

	x3.Sub(x3, threeX)
	x3.Add(x3, p521B)

	// y² = x³ - 3x + b
	y2 := new(fiat.P521Element).Square(y)

	if x3.Equal(y2) != 1 {
		return errors.New("P521 point not on curve")
	}
	return nil
}

// Bytes returns the uncompressed or infinity encoding of p, as specified in
// SEC 1, Version 2.0, Section 2.3.3. Note that the encoding of the point at
// infinity is shorter than all other encodings.
func (p *P521Point) Bytes() []byte {
	// This function is outlined to make the allocations inline in the caller
	// rather than happen on the heap.
	var out [133]byte
	return p.bytes(&out)
}

func (p *P521Point) bytes(out *[133]byte) []byte {
	if p.z.IsZero() == 1 {
		return append(out[:0], 0)
	}

	zinv := new(fiat.P521Element).Invert(p.z)
	xx := new(fiat.P521Element).Mul(p.x, zinv)
	yy := new(fiat.P521Element).Mul(p.y, zinv)

	buf := append(out[:0], 4)
	buf = append(buf, xx.Bytes()...)
	buf = append(buf, yy.Bytes()...)
	return buf
}

// Add sets q = p1 + p2, and returns q. The points may overlap.
func (q *P521Point) Add(p1, p2 *P521Point) *P521Point {
	// Complete addition formula for a = -3 from "Complete addition formulas for
	// prime order elliptic curves" (https://eprint.iacr.org/2015/1060), §A.2.

	t0 := new(fiat.P521Element).Mul(p1.x, p2.x) // t0 := X1 * X2
	t1 := new(fiat.P521Element).Mul(p1.y, p2.y) // t1 := Y1 * Y2
	t2 := new(fiat.P521Element).Mul(p1.z, p2.z) // t2 := Z1 * Z2
	t3 := new(fiat.P521Element).Add(p1.x, p1.y) // t3 := X1 + Y1
	t4 := new(fiat.P521Element).Add(p2.x, p2.y) // t4 := X2 + Y2
	t3.Mul(t3, t4)                              // t3 := t3 * t4
	t4.Add(t0, t1)                              // t4 := t0 + t1
	t3.Sub(t3, t4)                              // t3 := t3 - t4
	t4.Add(p1.y, p1.z)                          // t4 := Y1 + Z1
	x3 := new(fiat.P521Element).Add(p2.y, p2.z) // X3 := Y2 + Z2
	t4.Mul(t4, x3)                              // t4 := t4 * X3
	x3.Add(t1, t2)                              // X3 := t1 + t2
	t4.Sub(t4, x3)                              // t4 := t4 - X3
	x3.Add(p1.x, p1.z)                          // X3 := X1 + Z1
	y3 := new(fiat.P521Element).Add(p2.x, p2.z) // Y3 := X2 + Z2
	x3.Mul(x3, y3)                              // X3 := X3 * Y3
	y3.Add(t0, t2)                              // Y3 := t0 + t2
	y3.Sub(x3, y3)                              // Y3 := X3 - Y3
	z3 := new(fiat.P521Element).Mul(p521B, t2)  // Z3 := b * t2
	x3.Sub(y3, z3)                              // X3 := Y3 - Z3
	z3.Add(x3, x3)                              // Z3 := X3 + X3
	x3.Add(x3, z3)                              // X3 := X3 + Z3
	z3.Sub(t1, x3)                              // Z3 := t1 - X3
	x3.Add(t1, x3)                              // X3 := t1 + X3
	y3.Mul(p521B, y3)                           // Y3 := b * Y3
	t1.Add(t2, t2)                              // t1 := t2 + t2
	t2.Add(t1, t2)                              // t2 := t1 + t2
	y3.Sub(y3, t2)                              // Y3 := Y3 - t2
	y3.Sub(y3, t0)                              // Y3 := Y3 - t0
	t1.Add(y3, y3)                              // t1 := Y3 + Y3
	y3.Add(t1, y3)                              // Y3 := t1 + Y3
	t1.Add(t0, t0)                              // t1 := t0 + t0
	t0.Add(t1, t0)                              // t0 := t1 + t0
	t0.Sub(t0, t2)                              // t0 := t0 - t2
	t1.Mul(t4, y3)                              // t1 := t4 * Y3
	t2.Mul(t0, y3)                              // t2 := t0 * Y3
	y3.Mul(x3, z3)                              // Y3 := X3 * Z3
	y3.Add(y3, t2)                              // Y3 := Y3 + t2
	x3.Mul(t3, x3)                              // X3 := t3 * X3
	x3.Sub(x3, t1)                              // X3 := X3 - t1
	z3.Mul(t4, z3)                              // Z3 := t4 * Z3
	t1.Mul(t3, t0)                              // t1 := t3 * t0
	z3.Add(z3, t1)                              // Z3 := Z3 + t1

	q.x.Set(x3)
	q.y.Set(y3)
	q.z.Set(z3)
	return q
}

// Double sets q = p + p, and returns q. The points may overlap.
func (q *P521Point) Double(p *P521Point) *P521Point {
	// Complete addition formula for a = -3 from "Complete addition formulas for
	// prime order elliptic curves" (https://eprint.iacr.org/2015/1060), §A.2.

	t0 := new(fiat.P521Element).Square(p.x)    // t0 := X ^ 2
	t1 := new(fiat.P521Element).Square(p.y)    // t1 := Y ^ 2
	t2 := new(fiat.P521Element).Square(p.z)    // t2 := Z ^ 2
	t3 := new(fiat.P521Element).Mul(p.x, p.y)  // t3 := X * Y
	t3.Add(t3, t3)                             // t3 := t3 + t3
	z3 := new(fiat.P521Element).Mul(p.x, p.z)  // Z3 := X * Z
	z3.Add(z3, z3)                             // Z3 := Z3 + Z3
	y3 := new(fiat.P521Element).Mul(p521B, t2) // Y3 := b * t2
	y3.Sub(y3, z3)                             // Y3 := Y3 - Z3
	x3 := new(fiat.P521Element).Add(y3, y3)    // X3 := Y3 + Y3
	y3.Add(x3, y3)                             // Y3 := X3 + Y3
	x3.Sub(t1, y3)                             // X3 := t1 - Y3
	y3.Add(t1, y3)                             // Y3 := t1 + Y3
	y3.Mul(x3, y3)                             // Y3 := X3 * Y3
	x3.Mul(x3, t3)                             // X3 := X3 * t3
	t3.Add(t2, t2)                             // t3 := t2 + t2
	t2.Add(t2, t3)                             // t2 := t2 + t3
	z3.Mul(p521B, z3)                          // Z3 := b * Z3
	z3.Sub(z3, t2)                             // Z3 := Z3 - t2
	z3.Sub(z3, t0)                             // Z3 := Z3 - t0
	t3.Add(z3, z3)                             // t3 := Z3 + Z3
	z3.Add(z3, t3)                             // Z3 := Z3 + t3
	t3.Add(t0, t0)                             // t3 := t0 + t0
	t0.Add(t3, t0)                             // t0 := t3 + t0
	t0.Sub(t0, t2)                             // t0 := t0 - t2
	t0.Mul(t0, z3)                             // t0 := t0 * Z3
	y3.Add(y3, t0)                             // Y3 := Y3 + t0
	t0.Mul(p.y, p.z)                           // t0 := Y * Z
	t0.Add(t0, t0)                             // t0 := t0 + t0
	z3.Mul(t0, z3)                             // Z3 := t0 * Z3
	x3.Sub(x3, z3)                             // X3 := X3 - Z3
	z3.Mul(t0, t1)                             // Z3 := t0 * t1
	z3.Add(z3, z3)                             // Z3 := Z3 + Z3
	z3.Add(z3, z3)                             // Z3 := Z3 + Z3

	q.x.Set(x3)
	q.y.Set(y3)
	q.z.Set(z3)
	return q
}

// Select sets q to p1 if cond == 1, and to p2 if cond == 0.
func (q *P521Point) Select(p1, p2 *P521Point, cond int) *P521Point {
	q.x.Select(p1.x, p2.x, cond)
	q.y.Select(p1.y, p2.y, cond)
	q.z.Select(p1.z, p2.z, cond)
	return q
}

// ScalarMult sets p = scalar * q, and returns p.
func (p *P521Point) ScalarMult(q *P521Point, scalar []byte) *P521Point {
	// table holds the first 16 multiples of q. The explicit newP521Point calls
	// get inlined, letting the allocations live on the stack.
	var table = [16]*P521Point{
		NewP521Point(), NewP521Point(), NewP521Point(), NewP521Point(),
		NewP521Point(), NewP521Point(), NewP521Point(), NewP521Point(),
		NewP521Point(), NewP521Point(), NewP521Point(), NewP521Point(),
		NewP521Point(), NewP521Point(), NewP521Point(), NewP521Point(),
	}
	for i := 1; i < 16; i++ {
		table[i].Add(table[i-1], q)
	}

	// Instead of doing the classic double-and-add chain, we do it with a
	// four-bit window: we double four times, and then add [0-15]P.
	t := NewP521Point()
	p.Set(NewP521Point())
	for _, byte := range scalar {
		p.Double(p)
		p.Double(p)
		p.Double(p)
		p.Double(p)

		for i := uint8(0); i < 16; i++ {
			cond := subtle.ConstantTimeByteEq(byte>>4, i)
			t.Select(table[i], t, cond)
		}
		p.Add(p, t)

		p.Double(p)
		p.Double(p)
		p.Double(p)
		p.Double(p)

		for i := uint8(0); i < 16; i++ {
			cond := subtle.ConstantTimeByteEq(byte&0b1111, i)
			t.Select(table[i], t, cond)
		}
		p.Add(p, t)
	}

	return p
}
