package analyzer

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoAnalyzer implements the Analyzer interface for Go
type GoAnalyzer struct{}

// NewGoAnalyzer creates a new Go analyzer
func NewGoAnalyzer() *GoAnalyzer {
	return &GoAnalyzer{}
}

// Language returns the language this analyzer supports
func (a *GoAnalyzer) Language() Language {
	return LanguageGo
}

// Analyze analyzes Go source code
func (a *GoAnalyzer) Analyze(ctx context.Context, content []byte) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Language:  LanguageGo,
		Functions: []Function{},
		Classes:   []Class{},
		Imports:   []Import{},
		Comments:  []Comment{},
		Errors:    []ParseError{},
	}

	// Parse the Go source code
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		// Try to extract line number from error
		if syntaxErr, ok := err.(interface{ Pos() token.Pos }); ok {
			pos := fset.Position(syntaxErr.Pos())
			result.Errors = append(result.Errors, ParseError{
				Message: err.Error(),
				Line:    pos.Line,
				Column:  pos.Column,
			})
		} else {
			result.Errors = append(result.Errors, ParseError{
				Message: err.Error(),
				Line:    0,
				Column:  0,
			})
		}
		return result, nil // Return partial result with errors
	}

	result.AST = node

	// Extract comments
	for _, commentGroup := range node.Comments {
		comment := Comment{
			Text:      commentGroup.Text(),
			StartLine: fset.Position(commentGroup.Pos()).Line,
			EndLine:   fset.Position(commentGroup.End()).Line,
			IsBlock:   len(commentGroup.List) > 1,
		}
		result.Comments = append(result.Comments, comment)
	}

	// Extract imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		result.Imports = append(result.Imports, Import{
			Package: importPath,
			Alias:   alias,
			Line:    fset.Position(imp.Pos()).Line,
		})
	}

	// Walk the AST to extract functions and types
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			function := a.extractFunction(x, fset, result.Comments)
			if x.Recv != nil {
				// This is a method, add it to the appropriate struct/type
				a.addMethodToClass(result, x, function, fset)
			} else {
				// This is a standalone function
				result.Functions = append(result.Functions, function)
			}

		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						class := a.extractType(typeSpec, x, fset, result.Comments)
						if class != nil {
							result.Classes = append(result.Classes, *class)
						}
					}
				}
			}
		}
		return true
	})

	// Calculate complexity for functions
	for i := range result.Functions {
		result.Functions[i].Complexity = a.calculateComplexity(result.Functions[i], node)
	}

	// Calculate complexity for methods in classes
	for i := range result.Classes {
		for j := range result.Classes[i].Methods {
			result.Classes[i].Methods[j].Complexity = a.calculateComplexity(result.Classes[i].Methods[j], node)
		}
	}

	return result, nil
}

// extractFunction extracts function information from AST
func (a *GoAnalyzer) extractFunction(fn *ast.FuncDecl, fset *token.FileSet, comments []Comment) Function {
	function := Function{
		Name:       fn.Name.Name,
		StartLine:  fset.Position(fn.Pos()).Line,
		EndLine:    fset.Position(fn.End()).Line,
		Parameters: []Parameter{},
		IsPublic:   ast.IsExported(fn.Name.Name),
		IsTest:     strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark"),
	}

	// Extract documentation
	if fn.Doc != nil {
		function.Documentation = fn.Doc.Text()
	} else {
		function.Documentation = ExtractDocumentation(comments, function.StartLine)
	}

	// Extract parameters
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			paramType := a.typeToString(field.Type)
			for _, name := range field.Names {
				function.Parameters = append(function.Parameters, Parameter{
					Name: name.Name,
					Type: paramType,
				})
			}
		}
	}

	// Extract return type
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		var returnTypes []string
		for _, result := range fn.Type.Results.List {
			returnTypes = append(returnTypes, a.typeToString(result.Type))
		}
		function.ReturnType = strings.Join(returnTypes, ", ")
	}

	return function
}

// extractType extracts type information (struct, interface, etc.)
func (a *GoAnalyzer) extractType(typeSpec *ast.TypeSpec, genDecl *ast.GenDecl, fset *token.FileSet, comments []Comment) *Class {
	class := &Class{
		Name:       typeSpec.Name.Name,
		StartLine:  fset.Position(typeSpec.Pos()).Line,
		EndLine:    fset.Position(typeSpec.End()).Line,
		Methods:    []Function{},
		Properties: []Property{},
		IsPublic:   ast.IsExported(typeSpec.Name.Name),
	}

	// Extract documentation
	if genDecl.Doc != nil {
		class.Documentation = genDecl.Doc.Text()
	} else {
		class.Documentation = ExtractDocumentation(comments, class.StartLine)
	}

	// Determine type and extract properties
	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		class.Type = "struct"
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldType := a.typeToString(field.Type)
				if len(field.Names) == 0 {
					// Embedded field
					class.Properties = append(class.Properties, Property{
						Name:     fieldType,
						Type:     fieldType,
						IsPublic: ast.IsExported(fieldType),
					})
				} else {
					for _, name := range field.Names {
						class.Properties = append(class.Properties, Property{
							Name:     name.Name,
							Type:     fieldType,
							IsPublic: ast.IsExported(name.Name),
						})
					}
				}
			}
		}

	case *ast.InterfaceType:
		class.Type = "interface"
		// Interface methods are handled separately

	default:
		// Type alias or other type definition
		class.Type = "type"
	}

	return class
}

// addMethodToClass adds a method to the appropriate class
func (a *GoAnalyzer) addMethodToClass(result *AnalysisResult, fn *ast.FuncDecl, method Function, fset *token.FileSet) {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return
	}

	// Get receiver type name
	var receiverType string
	switch t := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		receiverType = t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			receiverType = ident.Name
		}
	}

	// Find the class and add the method
	for i := range result.Classes {
		if result.Classes[i].Name == receiverType {
			result.Classes[i].Methods = append(result.Classes[i].Methods, method)
			return
		}
	}
}

// typeToString converts an AST expression to a string representation
func (a *GoAnalyzer) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + a.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + a.typeToString(t.Elt)
		}
		return "[...]" + a.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + a.typeToString(t.Key) + "]" + a.typeToString(t.Value)
	case *ast.ChanType:
		return "chan " + a.typeToString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.SelectorExpr:
		return a.typeToString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// calculateComplexity calculates cyclomatic complexity for a function
func (a *GoAnalyzer) calculateComplexity(fn Function, file *ast.File) int {
	complexity := 1 // Base complexity

	// Find the function node in the AST
	var funcNode *ast.FuncDecl
	ast.Inspect(file, func(n ast.Node) bool {
		if f, ok := n.(*ast.FuncDecl); ok && f.Name.Name == fn.Name {
			funcNode = f
			return false
		}
		return true
	})

	if funcNode == nil {
		return complexity
	}

	// Count decision points
	ast.Inspect(funcNode, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// init registers the Go analyzer
func init() {
	RegisterAnalyzer(LanguageGo, NewGoAnalyzer())
}