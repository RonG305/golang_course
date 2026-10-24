// Lesson 19: packages, modules and visibility.
//
// A PACKAGE is a directory of .go files sharing a name. A MODULE is a tree of
// packages with a go.mod at its root - the unit of versioning and dependency.
//
// This module is `golangcourse` (see go.mod), so the package in
// ./lessons/19_packages_modules/mathx is imported as
// "golangcourse/lessons/19_packages_modules/mathx".
//
// Run: go run ./lessons/19_packages_modules
package main

import (
	"fmt"

	// Import path = module path + directory path. The last element is usually,
	// but not necessarily, the package NAME - the name comes from the source.
	"golangcourse/lessons/19_packages_modules/mathx"
)

func main() {
	// Only EXPORTED identifiers (capitalised) are reachable from here.
	fmt.Println("Add:", mathx.Add(2, 3))
	fmt.Println("Mean:", mathx.Mean([]float64{1, 2, 3, 4}))
	fmt.Println("Version:", mathx.Version)

	// mathx.normalise is unexported: referencing it would not compile.
	// _ = mathx.normalise

	acc := mathx.NewAccumulator()
	acc.Add(5)
	acc.Add(7)
	fmt.Println("Accumulator total:", acc.Total())

	fmt.Println(`
--- Module commands worth memorising ---
  go mod init github.com/you/project   create a module (use your real repo path)
  go mod tidy                          add missing deps, drop unused ones
  go get example.com/pkg@v1.2.3        add or upgrade one dependency
  go get -u ./...                      upgrade everything (minor+patch)
  go list -m all                       show the full dependency graph
  go mod why example.com/pkg           explain why a dep is needed
  go mod vendor                        copy deps into ./vendor
  go build ./...                       build every package
  go test ./...                        test every package
  go run ./cmd/server                  run one main package
  go doc fmt.Println                   read docs offline
  gofmt -w .                           format (non-negotiable in Go)
  go vet ./...                         catch suspicious code

--- Layout rules that matter in big projects ---
  cmd/<name>/      one directory per binary, each with a tiny main()
  internal/        importable ONLY by this module: the compiler enforces it
  pkg/             optional, for code you intend others to import
  Package names are short, lower-case, no underscores: user, httpapi, store.
  Avoid util/common/helpers - name packages after what they provide.
  Import cycles are a COMPILE ERROR, which pushes you toward clean layering.`)
}
