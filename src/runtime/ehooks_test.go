// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime_test

import (
	"internal/testenv"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestExitHooks(t *testing.T) {
	bmodes := []string{""}
	// Race detector is not supported everywhere -- limit to just
	// linux/amd64 to keep things simple. Note the HasCGO() test
	// below; this is to prevent the test running if CGO_ENABLED=0
	// is in effect.
	if !testing.Short() && runtime.GOARCH == "amd64" &&
		runtime.GOOS == "linux" && testenv.HasCGO() {
		bmodes = append(bmodes, "-race")
	}
	for _, bmode := range bmodes {
		scenarios := []struct {
			mode     string
			expected string
			musthave string
		}{
			{
				mode:     "simple",
				expected: "bar foo",
				musthave: "",
			},
			{
				mode:     "goodexit",
				expected: "orange apple",
				musthave: "",
			},
			{
				mode:     "badexit",
				expected: "blub blix",
				musthave: "",
			},
			{
				mode:     "panics",
				expected: "",
				musthave: "fatal error: internal error: exit hook invoked panic",
			},
			{
				mode:     "callsexit",
				expected: "",
				musthave: "fatal error: internal error: exit hook invoked exit",
			},
		}

		exe, err := buildTestProg(t, "testexithooks", bmode)
		if err != nil {
			t.Fatal(err)
		}

		bt := ""
		if bmode != "" {
			bt = " bmode: " + bmode
		}
		for _, s := range scenarios {
			cmd := exec.Command(exe, []string{"-mode", s.mode}...)
			out, _ := cmd.CombinedOutput()
			outs := strings.ReplaceAll(string(out), "\n", " ")
			outs = strings.TrimSpace(outs)
			if s.expected != "" {
				if s.expected != outs {
					t.Logf("raw output: %q", outs)
					t.Errorf("failed%s mode %s: wanted %q got %q", bt,
						s.mode, s.expected, outs)
				}
			} else if s.musthave != "" {
				if !strings.Contains(outs, s.musthave) {
					t.Logf("raw output: %q", outs)
					t.Errorf("failed mode %s: output does not contain %q",
						s.mode, s.musthave)
				}
			} else {
				panic("badly written scenario")
			}
		}
	}
}
