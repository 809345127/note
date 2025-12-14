package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run impl_to_intf.go <file>:<line>:<col>")
		os.Exit(1)
	}

	pos := os.Args[1]
	parts := strings.Split(pos, ":")
	if len(parts) != 3 {
		fmt.Println("Invalid position format. Use <file>:<line>:<col>")
		os.Exit(1)
	}

	filePath := parts[0]
	var line, col int
	fmt.Sscanf(parts[1], "%d", &line)
	fmt.Sscanf(parts[2], "%d", &col)

	if line == 0 || col == 0 {
		fmt.Println("Invalid line or column number")
		os.Exit(1)
	}

	// Get the method signature at the given position
	methodSig, methodName, err := getMethodSignature(filePath, line, col)
	if err != nil {
		log.Fatalf("Error getting method signature: %v", err)
	}

	fmt.Printf("🔍 Found method: %s\n", methodSig)
	fmt.Printf("📛 Method name: %s\n", methodName)

	// Find all interfaces in the workspace
	interfaces, err := findAllInterfaces(".")
	if err != nil {
		log.Fatalf("Error finding interfaces: %v", err)
	}

	fmt.Printf("Found %d interfaces\n", len(interfaces))

	// Find interfaces that the method implements
	var matches []Interface
	for _, iface := range interfaces {
		for _, method := range iface.Methods {
			// Compare the method signatures, ignoring the receiver type
			if matchSignatures(method.Signature, methodSig) {
				// Check if the method name matches exactly
				if method.Name == funcName(methodSig) {
					matches = append(matches, iface)
					fmt.Printf("\n✅ Match found!\n")
					fmt.Printf("📝 Interface: %s\n", iface.Name)
					fmt.Printf("📁 File: %s\n", iface.File)
					fmt.Printf("📍 Line: %d\n", iface.Line)
					fmt.Printf("🔧 Method: %s\n", method.Name)
					fmt.Printf("📌 Method Line: %d\n", method.Line)

					// Output in JSON format for VSCode extension to parse
					fmt.Printf("JSON_START{\"file\":\"%s\",\"line\":%d,\"methodLine\":%d,\"interface\":\"%s\",\"method\":\"%s\"}JSON_END\n", iface.File, iface.Line, method.Line, iface.Name, method.Name)
					return
				}
			}
		}
	}

	if len(matches) == 0 {
		fmt.Println("\n❌ No matching interface found.")
	}

	// Add a helper message
	fmt.Println("\n💡 Tip: Use 'Ctrl+Click' on the file path to open it in VSCode.")

}

// Method represents a method signature with its location
type Method struct {
	Name      string
	Signature string
	Line      int
}

// Interface represents an interface with its methods
type Interface struct {
	Name    string
	File    string
	Line    int
	Methods []Method
}

// getMethodSignature extracts the method signature at the given position
func getMethodSignature(filePath string, line, col int) (string, string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return "", "", err
	}

	var methodSig, methodName string
	var found bool

	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}

		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Check if this function declaration contains the given position
		funcPos := fset.Position(funcDecl.Pos())
		funcEnd := fset.Position(funcDecl.End())

		if funcPos.Line > line || (funcPos.Line == line && funcPos.Column > col) {
			return true
		}

		if funcEnd.Line < line || (funcEnd.Line == line && funcEnd.Column < col) {
			return true
		}

		// Found the function, extract its signature and name
		methodName = funcDecl.Name.Name
		methodSig = getFuncSignature(funcDecl)
		found = true
		return false
	})

	if !found {
		return "", "", fmt.Errorf("no method found at position %d:%d", line, col)
	}

	return methodSig, methodName, nil
}

// getFuncSignature returns the string representation of a function signature
func getFuncSignature(funcDecl *ast.FuncDecl) string {
	// Build the signature: func (<receiver>) <name>(<params>) <return type>
	var sig strings.Builder

	// Start with "func"
	sig.WriteString("func ")

	// Add receiver if present
	if funcDecl.Recv != nil {
		sig.WriteString("(")
		for i, field := range funcDecl.Recv.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			// Just use the type, not the name
			for j, _ := range field.Names {
				if j > 0 {
					sig.WriteString(", ")
				}
				if j == 0 {
					sig.WriteString(exprToString(field.Type))
				}
			}
		}
		sig.WriteString(") ")
	}

	// Add function name
	sig.WriteString(funcDecl.Name.Name)

	// Add parameters
	sig.WriteString("(")
	if funcDecl.Type.Params != nil {
		for i, param := range funcDecl.Type.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			// Just use the types, not the names
			for j, _ := range param.Names {
				if j > 0 {
					sig.WriteString(", ")
				}
				// For multiple names with the same type, just list the type once
				if j == 0 {
					sig.WriteString(exprToString(param.Type))
				}
			}
		}
	}
	sig.WriteString(")")

	// Add return types
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString("(")
		}
		for i, result := range funcDecl.Type.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			// Just use the types, not the names
			for j, _ := range result.Names {
				if j > 0 {
					sig.WriteString(", ")
				}
				// For multiple names with the same type, just list the type once
				if j == 0 {
					sig.WriteString(exprToString(result.Type))
				}
			}
		}
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// exprToString converts an AST expression to a string representation
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.ChanType:
		return "chan " + exprToString(e.Value)
	case *ast.FuncType:
		// This is a function type, we need to handle it specially
		var sig strings.Builder
		sig.WriteString("func(")
		if e.Params != nil {
			for i, param := range e.Params.List {
				if i > 0 {
					sig.WriteString(", ")
				}
				for j, _ := range param.Names {
					if j > 0 {
						sig.WriteString(", ")
					}
					if j == 0 {
						sig.WriteString(exprToString(param.Type))
					}
				}
			}
		}
		sig.WriteString(")")
		if e.Results != nil && len(e.Results.List) > 0 {
			sig.WriteString(" ")
			if len(e.Results.List) > 1 {
				sig.WriteString("(")
			}
			for i, result := range e.Results.List {
				if i > 0 {
					sig.WriteString(", ")
				}
				for j, _ := range result.Names {
					if j > 0 {
						sig.WriteString(", ")
					}
					if j == 0 {
						sig.WriteString(exprToString(result.Type))
					}
				}
			}
			if len(e.Results.List) > 1 {
				sig.WriteString(")")
			}
		}
		return sig.String()
	default:
		// For unknown types, just return a placeholder
		return fmt.Sprintf("<%T>", e)
	}
}

// matchSignatures compares two method signatures, ignoring the receiver type
func matchSignatures(sig1, sig2 string) bool {
	// Remove the receiver part from both signatures
	cleanSig1 := removeReceiver(sig1)
	cleanSig2 := removeReceiver(sig2)
	return cleanSig1 == cleanSig2
}

// removeReceiver removes the receiver part from a method signature
func removeReceiver(sig string) string {
	if !strings.HasPrefix(sig, "func (") {
		return sig
	}

	// Find the end of the receiver
	endReceiver := strings.Index(sig, ") ")
	if endReceiver == -1 {
		return sig
	}

	// Return the signature without the receiver
	return "func " + sig[endReceiver+2:]
}

// funcName extracts the function name from a method signature
func funcName(sig string) string {
	// Remove the receiver part
	cleanSig := removeReceiver(sig)
	if !strings.HasPrefix(cleanSig, "func (") {
		// This is a function, not a method
		// Find the function name
		start := 5 // len("func ")
		end := strings.Index(cleanSig[start:], "(")
		if end == -1 {
			return ""
		}
		return cleanSig[start : start+end]
	}

	// This is a method with receiver
	start := 6 // len("func (")
	// Find the end of the receiver
	endReceiver := strings.Index(cleanSig[start:], ") ")
	if endReceiver == -1 {
		return ""
	}

	// Find the function name after the receiver
	nameStart := start + endReceiver + 2 // +2 for ") "
	nameEnd := strings.Index(cleanSig[nameStart:], "(")
	if nameEnd == -1 {
		return ""
	}

	return cleanSig[nameStart : nameStart+nameEnd]
}

// findAllInterfaces finds all interfaces in the given directory
func findAllInterfaces(dir string) ([]Interface, error) {
	fset := token.NewFileSet()

	var interfaces []Interface

	// Walk through all Go files in the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files and generated files
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/mocks/") {
			return nil
		}

		// Convert relative path to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		file, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			ifaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				return true
			}

			// Found an interface, extract its methods
			var methods []Method
			if ifaceType.Methods != nil {
				for _, field := range ifaceType.Methods.List {
					if len(field.Names) == 0 {
						// This is an embedded interface, skip it
						continue
					}

					methodName := field.Names[0].Name
					methodLine := fset.Position(field.Pos()).Line

					// Extract the method signature
					var sig strings.Builder
					sig.WriteString("func (")

					// For interface methods, the receiver is the interface type itself
					sig.WriteString(typeSpec.Name.Name)
					sig.WriteString(") ")
					sig.WriteString(methodName)

					// Add parameters
					funcType, ok := field.Type.(*ast.FuncType)
					if !ok {
						return true
					}

					if funcType.Params != nil {
						sig.WriteString("(")
						for i, param := range funcType.Params.List {
							if i > 0 {
								sig.WriteString(", ")
							}
							for j, _ := range param.Names {
								if j > 0 {
									sig.WriteString(", ")
								}
								if j == 0 {
									sig.WriteString(exprToString(param.Type))
								}
							}
						}
						sig.WriteString(")")
					}

					// Add return types
					if funcType.Results != nil && len(funcType.Results.List) > 0 {
						sig.WriteString(" ")
						if len(funcType.Results.List) > 1 {
							sig.WriteString("(")
						}
						for i, result := range funcType.Results.List {
							if i > 0 {
								sig.WriteString(", ")
							}
							for j, _ := range result.Names {
								if j > 0 {
									sig.WriteString(", ")
								}
								if j == 0 {
									sig.WriteString(exprToString(result.Type))
								}
							}
						}
						if len(funcType.Results.List) > 1 {
							sig.WriteString(")")
						}
					}

					methods = append(methods, Method{
						Name:      methodName,
						Signature: sig.String(),
						Line:      methodLine,
					})
				}
			}

			interfaces = append(interfaces, Interface{
				Name:    typeSpec.Name.Name,
				File:    absPath,
				Line:    fset.Position(typeSpec.Pos()).Line,
				Methods: methods,
			})

			return true
		})

		return nil
	})

	return interfaces, err
}
