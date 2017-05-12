package main

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/charlievieth/buildutil"
)

var (
	sizes types.Sizes

	knownOS    = makeMap(buildutil.KnownOSList())
	knownArch  = makeMap(buildutil.KnownArchList())
	fset       = token.NewFileSet()
	errorCount = 0

	// TODO: add option for AllErrors
	parserMode parser.Mode
)

func makeMap(a []string) map[string]bool {
	m := make(map[string]bool, len(a))
	for _, s := range a {
		m[s] = true
	}
	return m
}

func updateGOOS(ctxt *build.Context, tags map[string]bool) string {
	if tags[ctxt.GOOS] {
		return ctxt.GOOS
	}
	for os, ok := range tags {
		if !knownOS[os] {
			continue
		}
		if ok && ctxt.GOOS != os {
			return os
		}
		if !ok && ctxt.GOOS == os && runtime.GOOS != ctxt.GOOS {
			if tags[runtime.GOOS] {
				return runtime.GOOS
			}
		}
	}
	return ctxt.GOOS
}

func initSizes() {
	wordSize := 8
	maxAlign := 8
	switch build.Default.GOARCH {
	case "386", "arm":
		wordSize = 4
		maxAlign = 4
		// add more cases as needed
	}
	sizes = &types.StdSizes{WordSize: int64(wordSize), MaxAlign: int64(maxAlign)}
}

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	if list, ok := err.(scanner.ErrorList); ok {
		errorCount += len(list)
		return
	}
	errorCount++
}

// parse may be called concurrently
func parse(filename string, src interface{}) (*ast.File, error) {
	return parser.ParseFile(fset, filename, src, parserMode)
}

func parseStdin() (*ast.File, error) {
	src, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return parse("<standard input>", src)
}

func parseFiles(filenames []string) ([]*ast.File, error) {
	files := make([]*ast.File, len(filenames))

	type parseResult struct {
		file *ast.File
		err  error
	}

	out := make(chan parseResult)
	for _, filename := range filenames {
		go func(filename string) {
			file, err := parse(filename, nil)
			out <- parseResult{file, err}
		}(filename)
	}

	for i := range filenames {
		res := <-out
		if res.err != nil {
			return nil, res.err // leave unfinished goroutines hanging
		}
		files[i] = res.file
	}

	return files, nil
}

func parseDir(ctxt *build.Context, dirname string, allFiles bool) ([]*ast.File, error) {
	pkginfo, err := ctxt.ImportDir(dirname, 0)
	if _, nogo := err.(*build.NoGoError); err != nil && !nogo {
		return nil, err
	}
	filenames := append(pkginfo.GoFiles, pkginfo.CgoFiles...)
	if allFiles {
		filenames = append(filenames, pkginfo.TestGoFiles...)
	}

	// complete file names
	for i, filename := range filenames {
		filenames[i] = filepath.Join(dirname, filename)
	}

	return parseFiles(filenames)
}

func checkPkgFiles(files []*ast.File) {
	compiler := "gc"
	type bailout struct{}
	conf := types.Config{
		FakeImportC: true,
		Error: func(err error) {
			if errorCount >= 10 {
				panic(bailout{})
			}
			report(err)
		},
		Importer: importer.For(compiler, nil),
		Sizes:    sizes,
	}

	defer func() {
		switch p := recover().(type) {
		case nil, bailout:
			// normal return or early exit
		default:
			// re-panic
			panic(p)
		}
	}()

	const path = "pkg" // any non-empty string will do for now
	conf.Check(path, fset, files, nil)
}

func updateContext(ctxt *build.Context, filename string) {
	tags := make(map[string]bool)
	if buildutil.GoodOSArchFile(ctxt, filename, tags) {
		// check for build tags
		if b, err := ioutil.ReadFile(filename); err == nil {
			if buildutil.ShouldBuild(ctxt, b, tags) {
				return
			}
		}
	}
}

func main() {
	initSizes()

	allErrors, _ := strconv.ParseBool(os.Getenv("GOSUBL_ALL_ERRORS"))
	_ = allErrors

	filename := os.Getenv("GOSUBL_LINT_FILENAME")
	allFiles := strings.HasSuffix(filename, "_test.go")

	var dirname string
	if filename != "" {
		dirname = filepath.Dir(filename)
	} else {
		var err error
		dirname, err = os.Getwd()
		if err != nil {
			report(err)
			os.Exit(2)
		}
	}

	ctxt := build.Default

	files, err := parseDir(&ctxt, dirname, allFiles)
	if err != nil {
		report(err)
		os.Exit(2)
	}
	checkPkgFiles(files)
	if errorCount > 0 {
		os.Exit(2)
	}
}
