package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGuidesUseIdenticalStandalonePrograms(t *testing.T) {
	root := filepath.Join("..", "..")
	fence := regexp.MustCompile("(?ms)^```go\\n(.*?)^```")
	english, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	thai, err := os.ReadFile(filepath.Join(root, "README.th.md"))
	if err != nil {
		t.Fatal(err)
	}
	left, right := fence.FindAllStringSubmatch(string(english), -1), fence.FindAllStringSubmatch(string(thai), -1)
	if len(left) != 13 || len(right) != len(left) {
		t.Fatalf("program count: %d / %d", len(left), len(right))
	}
	for index := range left {
		if left[index][1] != right[index][1] {
			t.Fatalf("program %d differs between locales", index)
		}
	}
}

func TestQuickStartAcceptsOnlyCheckedPayment(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	program := regexp.MustCompile("(?ms)^```go\\n(.*?)^```").FindStringSubmatch(string(readme))[1]
	directory := t.TempDir()
	source, binary := filepath.Join(directory, "main.go"), filepath.Join(directory, "verify-payment")
	if err := os.WriteFile(source, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", binary, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile guide: %s %v", out, err)
	}
	if err := os.WriteFile(filepath.Join(directory, "slip.png"), []byte{0x89, 0x50, 0x4e, 0x47}, 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		fields   map[string]any
		accepted bool
	}{
		{"checked", map[string]any{"isDuplicate": false, "isAmountMatched": true, "matchedAccount": map[string]any{"type": "BANK"}}, true},
		{"duplicate", map[string]any{"isDuplicate": true, "isAmountMatched": true, "matchedAccount": map[string]any{}}, false},
		{"duplicate_unknown", map[string]any{"isAmountMatched": true, "matchedAccount": map[string]any{}}, false},
		{"wrong_amount", map[string]any{"isDuplicate": false, "isAmountMatched": false, "matchedAccount": map[string]any{}}, false},
		{"amount_unknown", map[string]any{"isDuplicate": false, "matchedAccount": map[string]any{}}, false},
		{"wrong_account", map[string]any{"isDuplicate": false, "isAmountMatched": true, "matchedAccount": nil}, false},
		{"account_unknown", map[string]any{"isDuplicate": false, "isAmountMatched": true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fields["amountInSlip"] = 150.0
			c.fields["rawSlip"] = map[string]any{"transRef": "example-transfer", "date": "2026-10-29", "sender": map[string]any{"account": map[string]any{"name": map[string]any{"th": "Example"}}}, "receiver": map[string]any{"account": map[string]any{"name": map[string]any{"th": "Shop"}}}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseMultipartForm(5 << 20); err != nil {
					t.Error(err)
				}
				if r.MultipartForm != nil {
					defer r.MultipartForm.RemoveAll()
				}
				for _, field := range []string{"matchAccount", "checkDuplicate"} {
					if r.FormValue(field) != "true" {
						t.Errorf("%s not enabled", field)
					}
				}
				if r.FormValue("matchAmount") != "150" {
					t.Errorf("unexpected order amount %s", r.FormValue("matchAmount"))
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"success": true, "data": c.fields}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			command := exec.Command(binary)
			command.Dir = directory
			command.Env = append(os.Environ(), "EASYSLIP_API_KEY=example-test-key", "EASYSLIP_V2_BASE_URL="+server.URL, "EASYSLIP_V1_BASE_URL="+server.URL)
			out, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("guide failed: %s %v", out, err)
			}
			if strings.Contains(string(out), "Payment checks passed") != c.accepted {
				t.Fatalf("wrong payment decision: %s", out)
			}
		})
	}
}
