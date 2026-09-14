package registry_test

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"context-rail/internal/projectregistry"
)

func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			relative, relativeErr := filepath.Rel(root, path)
			if relativeErr != nil {
				return relativeErr
			}
			paths = append(paths, relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, relative := range paths {
		content, err := filepath.Abs(filepath.Join(root, relative))
		if err != nil {
			t.Fatal(err)
		}
		data, err := readFile(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprintf(hash, "%s\x00", relative)
		_, _ = hash.Write(data)
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func TestImportDoesNotMutateConsumerFixtures(t *testing.T) {
	roots := fixtureRoots(t)
	before := make(map[string]string, len(roots))
	for _, root := range roots {
		before[root] = treeDigest(t, root)
	}
	if _, err := projectregistry.Import(roots); err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		if after := treeDigest(t, root); after != before[root] {
			t.Fatalf("fixture %s changed: before %s after %s", root, before[root], after)
		}
	}
}
