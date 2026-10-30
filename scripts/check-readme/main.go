// Command check-readme compiles every README program and validates shell syntax.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if err := check(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func check() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "easyslip-readme-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/readme\n\ngo 1.26\n\nrequire github.com/EASYSLIP-CO-LTD/sdk-go v0.0.0\nreplace github.com/EASYSLIP-CO-LTD/sdk-go => "+filepath.ToSlash(root)+"\n"), 0600); err != nil {
		return err
	}
	fence := regexp.MustCompile("(?ms)^```([^\\n]*)\\n(.*?)^```[ \\t]*$")
	count, programs := 0, 0
	for fileIndex, name := range []string{"README.md", "README.th.md", "examples/README.md"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		for index, block := range fence.FindAllStringSubmatch(string(raw), -1) {
			count++
			switch block[1] {
			case "go":
				if !strings.HasPrefix(block[2], "package main\n") {
					return fmt.Errorf("%s block %d is not a standalone program", name, index)
				}
				programs++
				path := filepath.Join(directory, fmt.Sprintf("snippet%d_%d", fileIndex, index))
				if err := os.Mkdir(path, 0700); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(path, "main.go"), []byte(block[2]), 0600); err != nil {
					return err
				}
			case "sh":
				command := exec.Command("sh", "-n")
				command.Stdin = strings.NewReader(block[2])
				command.Stderr = os.Stderr
				if err := command.Run(); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unchecked language %q in %s", block[1], name)
			}
		}
	}
	if programs == 0 {
		return fmt.Errorf("no Go README programs found")
	}
	for _, args := range [][]string{{"vet", "./..."}, {"build", "./..."}} {
		command := exec.Command("go", args...)
		command.Dir = directory
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}
	fmt.Printf("README check: %d blocks checked; %d standalone Go programs\n", count, programs)
	return nil
}
