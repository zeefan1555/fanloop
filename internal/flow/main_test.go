package flow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zeefan1555/fanloop/internal/testfixture"
)

var testControllerRelease string

func TestMain(main *testing.M) {
	if err := testfixture.CanonicalTemp(); err != nil {
		panic(err)
	}
	root, err := os.MkdirTemp("", "fanloop-runtime-test-")
	if err != nil {
		panic(err)
	}
	testControllerRelease = filepath.Join(root, "test-release")
	if err := testfixture.BuildRelease(testControllerRelease, "test"); err != nil {
		panic(err)
	}
	code := main.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}
