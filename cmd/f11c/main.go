package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"f11/internal/f11"
)

type opts struct {
	emit string
	out  string
	in   string
}

func parseArgs(args []string) (opts, error) {
	o := opts{emit: "exe"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--emit":
			if i+1 >= len(args) {
				return o, fmt.Errorf("--emit requires a value")
			}
			i++
			o.emit = args[i]
		case strings.HasPrefix(a, "--emit="):
			o.emit = strings.TrimPrefix(a, "--emit=")
		case a == "-o":
			if i+1 >= len(args) {
				return o, fmt.Errorf("-o requires a path")
			}
			i++
			o.out = args[i]
		case a == "-h" || a == "--help":
			return o, fmt.Errorf("help")
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("unknown option %s", a)
		default:
			if o.in != "" {
				return o, fmt.Errorf("multiple input files: %s and %s", o.in, a)
			}
			o.in = a
		}
	}
	if o.in == "" {
		return o, fmt.Errorf("missing input file")
	}
	switch o.emit {
	case "check", "llvm", "obj", "exe":
	default:
		return o, fmt.Errorf("unknown emit mode %q", o.emit)
	}
	return o, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: f11c [--emit check|llvm|obj|exe] [-o path] file.f11")
}

func main() {
	o, err := parseArgs(os.Args[1:])
	if err != nil {
		usage()
		if err.Error() != "help" {
			fmt.Fprintln(os.Stderr, "f11c:", err)
		}
		if err.Error() == "help" {
			os.Exit(0)
		}
		os.Exit(2)
	}
	b, err := os.ReadFile(o.in)
	die(err)
	ts, err := f11.Lex(string(b))
	die(err)
	p, err := f11.Parse(ts)
	die(err)
	die(f11.Check(p))
	if o.emit == "check" {
		fmt.Println("ok")
		return
	}
	ir, err := f11.EmitLLVM(p)
	die(err)
	base := strings.TrimSuffix(filepath.Base(o.in), filepath.Ext(o.in))
	if o.out == "" {
		switch o.emit {
		case "llvm":
			o.out = base + ".ll"
		case "obj":
			o.out = base + ".o"
		default:
			o.out = base
		}
	}
	if o.emit == "llvm" {
		die(os.WriteFile(o.out, []byte(ir), 0644))
		return
	}
	tmp, err := os.CreateTemp("", "f11-*.ll")
	die(err)
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	_, err = tmp.WriteString(ir)
	if err == nil {
		err = tmp.Close()
	}
	die(err)
	args := []string{}
	if o.emit == "obj" {
		args = append(args, "-c")
	}
	args = append(args, tmpName, "-o", o.out)
	cmd := exec.Command("clang", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	die(cmd.Run())
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "f11c:", err)
		os.Exit(1)
	}
}
