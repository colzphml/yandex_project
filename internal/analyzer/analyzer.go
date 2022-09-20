// Package analyzer проверяет на вызов os.Exit в функции main пакетов main
package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// OsExitAnalyzer описывает возвращаемый анализатор
var OsExitAnalyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "check for os.Exit in main() and package main()",
	Run:  run,
}

// Фуункция run проходит по объявлениям в пакете main в поисках функции main.
// После ищет внутри main вызовы os.Exit через цепочку FuncDecl->ExprStmt->CallExpr->SelectorExpr и сверяет вызов функции с "os.Exit"
func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		// Эта бурда мне не нравится, как обойтись без нее???
		// Если будем просто inspect, то можем напороться на os.Exit в другой функции, которая вызывается в main
		if f.Name.Name == "main" {
			// Get all top-level declarations
			for _, decl := range f.Decls {
				// Check if top-level decl is a Func
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					// Check if function is main
					if funcDecl.Name.Name == "main" {
						// Iterate all elements in main
						for _, l := range funcDecl.Body.List {
							// Check elements is a ExprStmt
							switch exprStmt := l.(type) {
							case *ast.ExprStmt:
								// Check if ExprStmt is a CallExpr
								if call, ok := exprStmt.X.(*ast.CallExpr); ok {
									// Check if CallExpr is a SelectorExpr
									if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
										// Final get expression
										if first, ok := fun.X.(*ast.Ident); ok {
											result := first.Name + "." + fun.Sel.Name
											if result == "os.Exit" {
												pass.Reportf(first.NamePos, "call os.Exit in main() in package main")
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil, nil
}
