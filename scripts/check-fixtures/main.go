package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	raw, err := os.ReadFile("fixtures.sha256.json")
	if err != nil {
		panic(err)
	}
	var expected map[string]string
	if err = json.Unmarshal(raw, &expected); err != nil {
		panic(err)
	}
	tracked, err := exec.Command("git", "ls-files", "fixtures").Output()
	if err != nil {
		panic(err)
	}
	entries := strings.Fields(string(tracked))
	if len(entries) != 127 || len(expected) != 127 {
		panic("Expected 127 tracked fixtures")
	}
	for _, path := range entries {
		raw, err = os.ReadFile(filepath.Clean(path))
		if err != nil {
			panic(err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != expected[path] {
			panic("Fixture checksum mismatch: " + path)
		}
	}
	fmt.Println("127 tracked fixtures: SHA-256 PASS")
}
