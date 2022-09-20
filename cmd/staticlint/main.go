// Package main staticlint используется для анализа кода с помощью сторонних (и 1 собственного анализатора):
// analysis/passes + staticcheck.io (SA* и S1*) + errwrap + testableexamples
//
// Для запуска необходимо выполнить следующее:
//
// 1. Перейти в директорию с main.go файлов из состава staticlint
//
// 2. Выполнить go build -o mychecker
//
// 3. Запустить ./mychecker <директория с файлами для проверки>, например ./mychecker ./... проверит все файлы из текущего каталога и подкаталогов
package main

import (
	"strings"

	"github.com/colzphml/yandex_project/internal/analyzer"
	"github.com/fatih/errwrap/errwrap"
	"github.com/maratori/testableexamples/pkg/testableexamples"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/staticcheck"
)

// main - основная функция работы мультичекера.
func main() {
	mychecks := []*analysis.Analyzer{
		asmdecl.Analyzer,               // Package asmdecl defines an Analyzer that reports mismatches between assembly files and Go declarations.
		assign.Analyzer,                // Package assign defines an Analyzer that detects useless assignments.
		atomic.Analyzer,                // Package atomic defines an Analyzer that checks for common mistakes using the sync/atomic package.
		atomicalign.Analyzer,           // Package atomicalign defines an Analyzer that checks for non-64-bit-aligned arguments to sync/atomic functions.
		bools.Analyzer,                 // Package bools defines an Analyzer that detects common mistakes involving boolean operators.
		buildssa.Analyzer,              // Package buildssa defines an Analyzer that constructs the SSA representation of an error-free package and returns the set of all functions within it.
		buildtag.Analyzer,              // Package buildtag defines an Analyzer that checks build tags.
		cgocall.Analyzer,               // Package cgocall defines an Analyzer that detects some violations of the cgo pointer passing rules.
		composite.Analyzer,             // Package composite defines an Analyzer that checks for unkeyed composite literals.
		copylock.Analyzer,              // Package copylock defines an Analyzer that checks for locks erroneously passed by value.
		ctrlflow.Analyzer,              // Package ctrlflow is an analysis that provides a syntactic control-flow graph (CFG) for the body of a function.
		deepequalerrors.Analyzer,       // Package deepequalerrors defines an Analyzer that checks for the use of reflect.DeepEqual with error values.
		errorsas.Analyzer,              // The errorsas package defines an Analyzer that checks that the second argument to errors.As is a pointer to a type implementing error.
		fieldalignment.Analyzer,        // Package fieldalignment defines an Analyzer that detects structs that would use less memory if their fields were sorted.
		findcall.Analyzer,              // Package findcall defines an Analyzer that serves as a trivial example and test of the Analysis API.
		framepointer.Analyzer,          // Package framepointer defines an Analyzer that reports assembly code that clobbers the frame pointer before saving it.
		httpresponse.Analyzer,          // Package httpresponse defines an Analyzer that checks for mistakes using HTTP responses.
		ifaceassert.Analyzer,           // Package ifaceassert defines an Analyzer that flags impossible interface-interface type assertions.
		inspect.Analyzer,               // Package inspect defines an Analyzer that provides an AST inspector (golang.org/x/tools/go/ast/inspector.Inspector) for the syntax trees of a package.
		loopclosure.Analyzer,           // Package loopclosure defines an Analyzer that checks for references to enclosing loop variables from within nested functions.
		lostcancel.Analyzer,            // Package lostcancel defines an Analyzer that checks for failure to call a context cancellation function.
		nilfunc.Analyzer,               // Package nilfunc defines an Analyzer that checks for useless comparisons against nil.
		nilness.Analyzer,               // Package nilness inspects the control-flow graph of an SSA function and reports errors such as nil pointer dereferences and degenerate nil pointer comparisons.
		pkgfact.Analyzer,               // The pkgfact package is a demonstration and test of the package fact mechanism.
		printf.Analyzer,                // Package printf defines an Analyzer that checks consistency of Printf format strings and arguments.
		reflectvaluecompare.Analyzer,   // Package reflectvaluecompare defines an Analyzer that checks for accidentally using == or reflect.DeepEqual to compare reflect.Value values.
		shadow.Analyzer,                // Package shadow defines an Analyzer that checks for shadowed variables.
		shift.Analyzer,                 // Package shift defines an Analyzer that checks for shifts that exceed the width of an integer.
		sigchanyzer.Analyzer,           // Package sigchanyzer defines an Analyzer that detects misuse of unbuffered signal as argument to signal.Notify.
		sortslice.Analyzer,             // Package sortslice defines an Analyzer that checks for calls to sort.Slice that do not use a slice type as first argument.
		stdmethods.Analyzer,            // Package stdmethods defines an Analyzer that checks for misspellings in the signatures of methods similar to well-known interfaces.
		stringintconv.Analyzer,         // Package stringintconv defines an Analyzer that flags type conversions from integers to strings.
		structtag.Analyzer,             // Package structtag defines an Analyzer that checks struct field tags are well formed.
		testinggoroutine.Analyzer,      // Package testinggoroutine report calls to (*testing.T).Fatal from goroutines started by a test.
		tests.Analyzer,                 // Package tests defines an Analyzer that checks for common mistaken usages of tests and examples.
		unmarshal.Analyzer,             // The unmarshal package defines an Analyzer that checks for passing non-pointer or non-interface types to unmarshal and decode functions.
		unreachable.Analyzer,           // Package unreachable defines an Analyzer that checks for unreachable code.
		unsafeptr.Analyzer,             // Package unsafeptr defines an Analyzer that checks for invalid conversions of uintptr to unsafe.Pointer.
		unusedresult.Analyzer,          // Package unusedresult defines an analyzer that checks for unused results of calls to certain pure functions.
		unusedwrite.Analyzer,           // Package unusedwrite checks for unused writes to the elements of a struct or array object.
		usesgenerics.Analyzer,          // Package usesgenerics defines an Analyzer that checks for usage of generic features added in Go 1.18.
		errwrap.Analyzer,               // Package errwrap defines an Analyzer that rewrites error statements to use the new wrapping/unwrapping functionality
		testableexamples.NewAnalyzer(), // Package testableexamples checks if examples are testable.
		analyzer.OsExitAnalyzer,        // Package analyzer test if os.Exit calls in main function of package main
	}
	for _, v := range staticcheck.Analyzers {
		if strings.Contains(v.Analyzer.Name, "SA") || strings.Contains(v.Analyzer.Name, "S1") {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	multichecker.Main(
		mychecks...,
	)
}
