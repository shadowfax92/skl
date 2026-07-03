package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandDefaultsToShellEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")

	cmd := *initCmd
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatalf("init RunE: %v", err)
	}
	if got := out.String(); !strings.Contains(got, `dir="$(command skl cd "$@")"`) {
		t.Fatalf("init output should include zsh cd wrapper:\n%s", got)
	}
}

func TestShellInitScriptWrapsCDAndDelegatesOtherCommands(t *testing.T) {
	script, err := shellInitScript("bash")
	if err != nil {
		t.Fatalf("shellInitScript: %v", err)
	}

	required := []string{
		"skl() {",
		`dir="$(command skl cd "$@")"`,
		`builtin cd "$dir"`,
		`printf '%s\n' "$dir"`,
		`command skl "$@"`,
	}
	for _, want := range required {
		if !strings.Contains(script, want) {
			t.Fatalf("shell init missing %q:\n%s", want, script)
		}
	}
}

func TestShellInitScriptRejectsUnsupportedShell(t *testing.T) {
	if _, err := shellInitScript("fish"); err == nil {
		t.Fatalf("fish should be rejected until it has explicit integration")
	}
}

func TestShellNameFromEnvHandlesEmptyShell(t *testing.T) {
	if got := shellNameFromEnv(""); got != "" {
		t.Fatalf("empty shell path should stay empty, got %q", got)
	}
}

func TestInitCommandRejectsExtraArgs(t *testing.T) {
	cmd := *initCmd
	if err := cmd.Args(&cmd, []string{"zsh", "extra"}); err == nil {
		t.Fatalf("init should reject extra args")
	}
}

func TestZshWrapperPreservesPathOutputWhenNotTTY(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}

	fakeBin := t.TempDir()
	target := filepath.Join(t.TempDir(), "library root")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	fakeSkl := filepath.Join(fakeBin, "skl")
	body := "#!/bin/sh\nif [ \"$1\" = cd ]; then printf '%s\\n' \"$SKL_FAKE_PATH\"; exit 0; fi\nprintf 'delegated:%s\\n' \"$*\"\n"
	if err := os.WriteFile(fakeSkl, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake skl: %v", err)
	}

	script, err := shellInitScript("zsh")
	if err != nil {
		t.Fatalf("shellInitScript: %v", err)
	}

	startDir := t.TempDir()
	resolvedStartDir, err := filepath.EvalSymlinks(startDir)
	if err != nil {
		t.Fatalf("resolve start dir: %v", err)
	}
	initFile := filepath.Join(t.TempDir(), "skl-init.zsh")
	if err := os.WriteFile(initFile, []byte(script), 0o644); err != nil {
		t.Fatalf("write init script: %v", err)
	}

	program := fmt.Sprintf("source %q\nbefore=$PWD\nout=$(skl cd)\nprintf 'OUT=%%s\\nPWD=%%s\\nBEFORE=%%s\\n' \"$out\" \"$PWD\" \"$before\"\n", initFile)
	cmd := exec.Command("zsh", "-fc", program)
	cmd.Dir = startDir
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+":"+os.Getenv("PATH"), "SKL_FAKE_PATH="+target)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh wrapper failed: %v\n%s", err, output)
	}

	got := string(output)
	if !strings.Contains(got, "OUT="+target+"\n") {
		t.Fatalf("wrapper should print path when stdout is not a tty:\n%s", got)
	}
	if !strings.Contains(got, "PWD="+resolvedStartDir+"\n") {
		t.Fatalf("wrapper should not cd in command substitution:\n%s", got)
	}
}
