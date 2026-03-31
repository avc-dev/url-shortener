package osexitanalyzer_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/avc-dev/url-shortener/cmd/staticlint/osexitanalyzer"
)

func testDataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestOsExitInMain(t *testing.T) {
	// exitcheck — package where main calls os.Exit directly: one diagnostic expected.
	analysistest.Run(t, testDataDir(), osexitanalyzer.Analyzer, "exitcheck")
}

func TestOsExitNotInMain(t *testing.T) {
	// noerror — os.Exit is only called from a helper, not main: no diagnostics expected.
	analysistest.Run(t, testDataDir(), osexitanalyzer.Analyzer, "noerror")
}
