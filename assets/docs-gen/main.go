package main

import (
	"docs-gen/auditevents"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DocsGenerator is a function that writes to an io.Writer based on analyzing
// parsed Go source files.
type DocsGenerator func(io.Writer, []*ast.File) error

func main() {

	from := flag.String("from", "", "Path to the root of the directory tree to use for analyzing Go files")
	to := flag.String("to", "", "Path to the directory to use for writing generated docs")
	flag.Parse()

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "You must provide values for the -from and -to flags.")
		flag.Usage()
		os.Exit(1)
	}

	gofiles := []*ast.File{}

	// Parse Go source files. We will use the results for all docs generators.
	if err := filepath.Walk(*from, func(pth string, i fs.FileInfo, _ error) error {
		if !strings.HasSuffix(i.Name(), ".go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), pth, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing Go source files: %v", err)
			os.Exit(1)
		}
		gofiles = append(gofiles, f)
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error walking gravitational/teleport: %v", err)
		os.Exit(1)
	}

	// map files within the output directory to their corresponding docs generators
	generators := map[string]DocsGenerator{
		"audit-events.md": auditevents.GenerateAuditEventsTable,
	}

	for n, g := range generators {
		f, err := os.OpenFile(path.Join(*to, n), os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file %v: %v\n", n, err)
			os.Exit(1)
		}
		if err := g(f, gofiles); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating docs file %v: %v\n", n, err)
			os.Exit(1)
		}
	}

}
