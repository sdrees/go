// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// A Type represents a type of Go.
// All types implement the Type interface.
type Type interface {
	// Underlying returns the underlying type of a type
	// w/o following forwarding chains. Only used by
	// client packages (here for backward-compatibility).
	Underlying() Type

	// String returns a string representation of a type.
	String() string
}

// top represents the top of the type lattice.
// It is the underlying type of a type parameter that
// can be satisfied by any type (ignoring methods),
// because its type constraint contains no restrictions
// besides methods.
type top struct{}

// theTop is the singleton top type.
var theTop = &top{}

func (t *top) Underlying() Type { return t }
func (t *top) String() string   { return TypeString(t, nil) }

// under returns the true expanded underlying type.
// If it doesn't exist, the result is Typ[Invalid].
// under must only be called when a type is known
// to be fully set up.
func under(t Type) Type {
	// TODO(gri) is this correct for *Union?
	if n := asNamed(t); n != nil {
		return n.under()
	}
	return t
}

// optype returns a type's operational type. Except for
// type parameters, the operational type is the same
// as the underlying type (as returned by under). For
// Type parameters, the operational type is determined
// by the corresponding type bound's type list. The
// result may be the bottom or top type, but it is never
// the incoming type parameter.
func optype(typ Type) Type {
	if t := asTypeParam(typ); t != nil {
		// If the optype is typ, return the top type as we have
		// no information. It also prevents infinite recursion
		// via the asTypeParam converter function. This can happen
		// for a type parameter list of the form:
		// (type T interface { type T }).
		// See also issue #39680.
		if a := t.Bound().typeSet().types; a != nil && a != typ {
			// If we have a union with a single entry, ignore
			// any tilde because under(~t) == under(t).
			if u, _ := a.(*Union); u != nil && u.NumTerms() == 1 {
				a = u.types[0]
			}
			if a != typ {
				// a != typ and a is a type parameter => under(a) != typ, so this is ok
				return under(a)
			}
		}
		return theTop
	}
	return under(typ)
}

// Converters
//
// A converter must only be called when a type is
// known to be fully set up. A converter returns
// a type's operational type (see comment for optype)
// or nil if the type argument is not of the
// respective type.

func asBasic(t Type) *Basic {
	op, _ := optype(t).(*Basic)
	return op
}

func asArray(t Type) *Array {
	op, _ := optype(t).(*Array)
	return op
}

func asSlice(t Type) *Slice {
	op, _ := optype(t).(*Slice)
	return op
}

func asStruct(t Type) *Struct {
	op, _ := optype(t).(*Struct)
	return op
}

func asPointer(t Type) *Pointer {
	op, _ := optype(t).(*Pointer)
	return op
}

func asSignature(t Type) *Signature {
	op, _ := optype(t).(*Signature)
	return op
}

// If the argument to asInterface, asNamed, or asTypeParam is of the respective type
// (possibly after expanding an instance type), these methods return that type.
// Otherwise the result is nil.

func asInterface(t Type) *Interface {
	op, _ := optype(t).(*Interface)
	return op
}

func asNamed(t Type) *Named {
	e, _ := expand(t).(*Named)
	return e
}

func asTypeParam(t Type) *TypeParam {
	u, _ := under(t).(*TypeParam)
	return u
}
