package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/ast/astutil"
)

type EventData struct {
	Name    string
	Comment string
}

// To be executed with an []EventData
var tmpl string = `|Event Type|Description|
|---|---|
{{- range . }}
|{{.Name}}|{{.Comment}}|
{{- end }}
`

func main() {
	eventTypes := make(map[string]struct{})
	gofiles := []*ast.File{}
	eventData := []EventData{}

	// Parse Go source files
	if err := filepath.Walk(path.Join("..", ".."), func(pth string, i fs.FileInfo, _ error) error {
		if !strings.HasSuffix(i.Name(), ".go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), pth, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing Go source files: %v", err)
			os.Exit(1)
		}
		gofiles = append(gofiles, f)
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error walking gravitational/teleport: %v", err)
		os.Exit(1)
	}

	// We will traverse the AST of each Go file twice: once to collect types of
	// audit events that are used in apievents.Metadata declarations, and
	// again to see where those audit event types are declared. In the second
	// traversal, we'll collect the string values of those event types along
	// their godoc comments.

	// First walk through the AST: collect types of audit events.
	// We identify audit event types by instances where a field named
	// "Metadata" is assigned to a composite literal with type
	// "Metadata". Further, that Metadata composite literal has a
	// field called "Type".
	for _, f := range gofiles {

		for _, d := range f.Decls {
			astutil.Apply(d, func(c *astutil.Cursor) bool {
				// We're looking for a KeyValueExpr
				// "Metadata: apievents.Metadata{}"
				if kv, ok := c.Node().(*ast.KeyValueExpr); ok {

					if ki, ok := kv.Key.(*ast.Ident); !ok || ki.Name != "Metadata" {
						// This can't be the Metadata field of an audit
						// event, since it's not an identifier named "Metadata"
						return true
					}

					// The value of the KeyValueExpression must be an
					// apievents.Metadata struct
					if vl, ok := kv.Value.(*ast.CompositeLit); ok {
						if vt, ok := vl.Type.(*ast.SelectorExpr); ok && vt.Sel.Name == "Metadata" {

							// The type of the composite literal must come
							// from the "apievents" package.
							if vtx, ok := vt.X.(*ast.Ident); !ok || vtx.Name != "apievents" {
								return true
							}

							// Go through the fields of the composite literal
							// and collect the type to use later
							for _, el := range vl.Elts {
								elkv, ok := el.(*ast.KeyValueExpr)
								if !ok {
									continue
								}
								elkvk, ok := elkv.Key.(*ast.Ident)
								if !ok {
									continue
								}

								// We have an audit event type, so save it
								// for our second walk through the AST.
								if elkvk.Name == "Type" {
									elkvv, ok := elkv.Value.(*ast.SelectorExpr)
									if !ok {
										continue
									}
									eventTypes[elkvv.Sel.Name] = struct{}{}
								}
							}
						}
					}
				}
				return true
			}, nil)
		}

	}

	// Second walk through the AST: find definitions of audit event
	// types by comparing them to the audit event types we collected
	// in the first walk. Gather the comments.
	for _, f := range gofiles {
		for _, d := range f.Decls {
			astutil.Apply(d, func(c *astutil.Cursor) bool {
				// Look through all declarations and find those that match
				// the identifiers we have collected.
				val, ok := c.Node().(*ast.ValueSpec)
				if !ok {
					return true
				}
				for _, n := range val.Names {
					if _, y := eventTypes[n.Name]; y {
						tx := strings.Trim(val.Doc.Text(), "\n")
						typ := strings.ReplaceAll(val.Values[0].(*ast.BasicLit).Value, "\"", "`")
						if strings.HasPrefix(tx, n.Name) {
							tx = strings.ReplaceAll(tx, n.Name, typ)
						}
						eventData = append(eventData, EventData{
							Name:    typ,
							Comment: tx,
						})
					}
				}
				return true
			}, nil)
		}
	}

	tt, err := template.New("table").Parse(tmpl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing the audit event reference template: %v", err)
		os.Exit(1)
	}
	if err := tt.Execute(os.Stdout, eventData); err != nil {
		fmt.Fprintf(os.Stderr, "error executing the audit event reference template: %v", err)
		os.Exit(1)
	}
}
