// Package osexitanalyzer provides a static analysis pass that reports direct
// calls to [os.Exit] inside the main function of package main.
//
// # Rationale
//
// Calling os.Exit directly from main bypasses all deferred functions, which
// makes orderly shutdown (closing files, flushing logs, releasing connections)
// impossible. Instead of calling os.Exit, the recommended pattern is to return
// an error from a sub-function and let main (or the framework) handle it, or
// to use [log.Fatal] / [log.Fatalf] when a fatal condition is truly
// unrecoverable.
//
// # What is flagged
//
//	package main
//
//	import "os"
//
//	func main() {
//	    os.Exit(1) // ← flagged
//	}
//
// # What is NOT flagged
//
//   - os.Exit calls inside any function other than main (helper functions,
//     init, etc.)
//   - os.Exit calls in packages other than main
//   - log.Fatal / log.Fatalf (not direct os.Exit calls at the AST level)
//   - Generated files (files containing "Code generated … DO NOT EDIT.")
//     such as the test runner main produced by `go test`
package osexitanalyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the os.Exit-in-main checker.
// It reports any call expression that resolves to os.Exit inside the top-level
// main function of a non-generated file in package main.
var Analyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "reports os.Exit calls in the main function of package main",
	Run:  run,
}

// run is the entry point for the analysis pass.
func run(pass *analysis.Pass) (interface{}, error) {
	// Only interested in the main package.
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Skip files marked as generated (e.g., `go test` test runner stubs).
		// Generated files are identified by the standard marker comment
		// "Code generated … DO NOT EDIT." described in https://go.dev/s/generatedcode.
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil || fn.Body == nil {
				continue
			}

			// Walk the body of main looking for call expressions.
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// Use type information to resolve the callee — this correctly handles
				// import aliases (e.g. import myos "os").
				obj, ok := pass.TypesInfo.Uses[sel.Sel]
				if !ok {
					return true
				}

				fn, ok := obj.(*types.Func)
				if !ok {
					return true
				}

				pkg := fn.Pkg()
				if pkg != nil && pkg.Path() == "os" && fn.Name() == "Exit" {
					pass.Reportf(call.Pos(), "direct os.Exit call in main function of main package is prohibited; use log.Fatal or return an error instead")
				}

				return true
			})
		}
	}

	return nil, nil
}
