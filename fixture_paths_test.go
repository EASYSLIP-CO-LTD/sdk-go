package easyslip

import (
	"os"
	"path/filepath"
)

func fixturePath(path string) string {
	root := os.Getenv("EASYSLIP_FIXTURES_DIR")
	if root == "" {
		root = "fixtures"
	}
	return filepath.Join(root, path)
}
