// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types2

import (
	"cmd/compile/internal/syntax"
	"fmt"
	"go/version"
	"internal/goversion"
)

// A goVersion is a Go language version string of the form "go1.%d"
// where d is the minor version number. goVersion strings don't
// contain release numbers ("go1.20.1" is not a valid goVersion).
type goVersion string

// asGoVersion returns v as a goVersion (e.g., "go1.20.1" becomes "go1.20").
// If v is not a valid Go version, the result is the empty string.
func asGoVersion(v string) goVersion {
	return goVersion(version.Lang(v))
}

// isValid reports whether v is a valid Go version.
func (v goVersion) isValid() bool {
	return v != ""
}

// cmp returns -1, 0, or +1 depending on whether x < y, x == y, or x > y,
// interpreted as Go versions.
func (x goVersion) cmp(y goVersion) int {
	return version.Compare(string(x), string(y))
}

var (
	// Go versions that introduced language changes
	go1_9  = asGoVersion("go1.9")
	go1_13 = asGoVersion("go1.13")
	go1_14 = asGoVersion("go1.14")
	go1_17 = asGoVersion("go1.17")
	go1_18 = asGoVersion("go1.18")
	go1_20 = asGoVersion("go1.20")
	go1_21 = asGoVersion("go1.21")
	go1_22 = asGoVersion("go1.22")
	go1_23 = asGoVersion("go1.23")

	// current (deployed) Go version
	go_current = asGoVersion(fmt.Sprintf("go1.%d", goversion.Version))
)

// allowVersion reports whether the given package is allowed to use version v.
func (check *Checker) allowVersion(pkg *Package, at poser, v goVersion) bool {
	// We assume that imported packages have all been checked,
	// so we only have to check for the local package.
	if pkg != check.pkg {
		return true
	}

	// If no explicit file version is specified,
	// fileVersion corresponds to the module version.
	var fileVersion goVersion
	if pos := at.Pos(); pos.IsKnown() {
		// We need version.Lang below because file versions
		// can be (unaltered) Config.GoVersion strings that
		// may contain dot-release information.
		fileVersion = asGoVersion(check.versions[base(pos)])
	}
	return !fileVersion.isValid() || fileVersion.cmp(v) >= 0
}

// verifyVersionf is like allowVersion but also accepts a format string and arguments
// which are used to report a version error if allowVersion returns false. It uses the
// current package.
func (check *Checker) verifyVersionf(at poser, v goVersion, format string, args ...interface{}) bool {
	if !check.allowVersion(check.pkg, at, v) {
		check.versionErrorf(at, v, format, args...)
		return false
	}
	return true
}

// base finds the underlying PosBase of the source file containing pos,
// skipping over intermediate PosBase layers created by //line directives.
// The positions must be known.
func base(pos syntax.Pos) *syntax.PosBase {
	assert(pos.IsKnown())
	b := pos.Base()
	for {
		bb := b.Pos().Base()
		if bb == nil || bb == b {
			break
		}
		b = bb
	}
	return b
}
