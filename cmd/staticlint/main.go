// Package main implements a multichecker static analysis tool for the
// url-shortener project.
//
// # Running the multichecker
//
// Build and run against the entire module:
//
//	go run ./cmd/staticlint/... ./...
//
// Or build a binary and use it as a standalone linter:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// To analyse a specific package:
//
//	./staticlint github.com/avc-dev/url-shortener/internal/handler
//
// Flags follow the standard go/analysis convention (-<analyzer>.<flag>=value).
// Use -help to list all flags.
//
// # Included analyzers
//
// ## Standard passes (golang.org/x/tools/go/analysis/passes)
//
// These are the same checks bundled with `go vet` plus several additional
// passes shipped as part of the Go tools module:
//
//   - appends       — detects append(s) with a single argument (no values added)
//   - asmdecl       — checks consistency between Go assembly stubs and implementations
//   - assign        — detects useless assignments whose result is immediately overwritten
//   - atomic        — checks for common mistakes using sync/atomic
//   - atomicalign   — ensures 64-bit fields are properly aligned for atomic access
//   - bools         — detects redundant or impossible boolean expressions
//   - buildtag      — verifies //go:build and +build constraints are well-formed
//   - cgocall       — checks for unsafe cgo pointer rules
//   - composite     — reports unkeyed composite literals
//   - copylock      — detects types containing sync.Locker being copied by value
//   - deepequalerrors — flags reflect.DeepEqual called on error values
//   - defers        — detects common mistakes in defer statements
//   - directive     — checks well-formedness of known //go: directives
//   - errorsas      — verifies the second argument to errors.As is a non-nil pointer
//   - fieldalignment — detects structs that could be made smaller by reordering fields
//   - framepointer  — reports assembly functions that clobber the frame pointer
//   - hostport      — checks for invalid host:port string construction
//   - httpmux       — checks for incorrect net/http ServeMux patterns
//   - httpresponse  — checks for mistakes handling HTTP responses
//   - ifaceassert   — detects impossible interface-to-interface type assertions
//   - loopclosure   — checks for references to loop variables from closures
//   - lostcancel    — checks that context cancellation functions are called
//   - nilfunc       — reports comparisons of functions to nil
//   - nilness       — checks for redundant or impossible nil comparisons
//   - printf        — checks consistency of Printf format strings and arguments
//   - reflectvaluecompare — flags Value.Addr comparisons using == or reflect.DeepEqual
//   - shadow        — checks for shadowed variables
//   - shift         — checks for shifts that exceed the width of the integer
//   - sigchanyzer   — detects misuse of os/signal.Notify
//   - slog          — checks for mismatched key-value pairs in log/slog calls
//   - sortslice     — checks the sort.Slice call has a non-nil function argument
//   - stdmethods    — checks for misspelled signatures of standard interfaces
//   - stdversion    — reports uses of too-new standard library symbols
//   - stringintconv — flags conversions from int to string
//   - structtag     — checks struct tags are well-formed
//   - testinggoroutine — detects calls to Fatal from a test goroutine
//   - tests         — checks signatures of test/benchmark/example functions
//   - timeformat    — checks for incorrect uses of time.Format and time.Parse
//   - unmarshal     — reports passing non-pointer or non-interface to Unmarshal
//   - unreachable   — checks for unreachable code
//   - unsafeptr     — checks for invalid conversions of uintptr to unsafe.Pointer
//   - unusedresult  — checks for unused results of calls to pure functions
//   - unusedwrite   — checks for unused writes to struct fields and arrays
//   - waitgroup     — detects misuse of sync.WaitGroup
//
// ## Staticcheck SA class (honnef.co/go/tools/staticcheck)
//
// All SA (staticcheck) analyzers are included. These find real bugs with a
// very low false-positive rate. Examples include:
//
//   - SA1000 — invalid regular expressions
//   - SA1006 — Printf with dynamic first argument
//   - SA2002 — Called testing.T.Fatal from a goroutine in a test
//   - SA4006 — A value assigned to a variable is never read
//   - SA5001 — defer on a nil function value
//   - SA9003 — Empty body in if/else branch
//
// ## Staticcheck S1 class (honnef.co/go/tools/simple)
//
// Code simplification suggestions. All S1 analyzers are included. Examples:
//
//   - S1000 — Use a plain channel send/receive instead of select with a single case
//   - S1016 — Use a type conversion instead of manually copying struct fields
//   - S1039 — Unnecessary use of fmt.Sprintf
//
// ## Staticcheck ST1 class (honnef.co/go/tools/stylecheck)
//
// Style enforcement. All ST1 analyzers are included. Examples:
//
//   - ST1003 — Incorrectly named identifiers (e.g. Id → ID)
//   - ST1017 — Don't use Yoda conditions
//   - ST1020 — Exported functions and types should have documentation comments
//
// ## Public third-party analyzers
//
// Two additional widely-used community analyzers are included:
//
//   - bodyclose (github.com/timakin/bodyclose) — detects when the body of an
//     http.Response is not closed. Forgetting to close the response body leaks
//     the underlying TCP connection and prevents it from being reused.
//
//   - nilerr (github.com/gostaticanalysis/nilerr) — detects code that returns
//     a nil error even though a non-nil error variable is in scope, which
//     almost always indicates a copy-paste bug.
//
// ## Custom analyzer
//
//   - osexitcheck — forbids direct calls to os.Exit inside the main function of
//     package main. Such calls bypass all deferred functions, preventing orderly
//     shutdown. Use log.Fatal or return an error from a sub-function instead.
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/waitgroup"

	lintanalysis "honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"

	"github.com/avc-dev/url-shortener/cmd/staticlint/osexitanalyzer"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// ── Standard passes ───────────────────────────────────────────────────
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		framepointer.Analyzer,
		hostport.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		waitgroup.Analyzer,

		// ── Public third-party analyzers ──────────────────────────────────────
		bodyclose.Analyzer,
		nilerr.Analyzer,

		// ── Custom analyzer ───────────────────────────────────────────────────
		osexitanalyzer.Analyzer,
	}

	// Append SA analyzers (all checks from staticcheck.io SA class).
	analyzers = append(analyzers, lintSlice(staticcheck.Analyzers)...)

	// Append S1 analyzers (code simplification, "simple" class).
	analyzers = append(analyzers, lintSlice(simple.Analyzers)...)

	// Append ST1 analyzers (style enforcement, "stylecheck" class).
	// This satisfies the requirement for at least one non-SA class from staticcheck.io.
	analyzers = append(analyzers, lintSlice(stylecheck.Analyzers)...)

	multichecker.Main(analyzers...)
}

// lintSlice converts []*lint.Analyzer (staticcheck's wrapper type) to the
// []*analysis.Analyzer slice expected by multichecker.Main.
func lintSlice(in []*lintanalysis.Analyzer) []*analysis.Analyzer {
	out := make([]*analysis.Analyzer, len(in))
	for i, a := range in {
		out[i] = a.Analyzer
	}
	return out
}
