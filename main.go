package gox

import (
	"errors"
	"github.com/caleb-sideras/gox/utils"
	"github.com/gorilla/mux"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	DIR            = "/"
	GO_EXT         = ".go"
	HTML_EXT       = ".html"
	PAGE           = "page"
	INDEX          = "index"
	METADATA       = "metadata"
	DATA           = "data"
	RENDER         = "render"
	HANDLER        = "handler"
	BODY           = "-body"
	PAGE_BODY      = PAGE + BODY
	PAGE_FILE      = PAGE + HTML_EXT
	INDEX_FILE     = INDEX + HTML_EXT
	METADATA_FILE  = METADATA + HTML_EXT
	DATA_FILE      = DATA + GO_EXT
	RENDER_FILE    = RENDER + GO_EXT
	HANDLER_FILE   = HANDLER + GO_EXT
	PAGE_BODY_FILE = PAGE_BODY + HTML_EXT
)

var FILE_CHECK_LIST = map[string]bool{
	DATA_FILE:     true,
	RENDER_FILE:   true,
	HANDLER_FILE:  true,
	INDEX_FILE:    true,
	PAGE_FILE:     true,
	METADATA_FILE: true,
}

type Gox struct {
	OutputDir string
}

func NewGox(outputDir string) *Gox {
	return &Gox{
		OutputDir: outputDir,
	}
}

func (g *Gox) Build(startDir string, packageDir string) {
	dirFiles, err := walkDirectoryStructure(startDir)
	if err != nil {
		log.Fatalf("error walking the path %v: %v", startDir, err)
	}

	for k, v := range dirFiles {
		log.Println("Directory:", k)
		for ext, files := range v {
			log.Println("  ", ext)
			for file := range files {
				log.Println("   -", file)
			}
		}
	}

	// used for generated output
	var routes []string
	var renderCustomFunctions []string
	var renderDefaultFunctions []string
	var handlerDefaultFunctions []string
	imports := utils.NewStringSet()

	for dir, files := range dirFiles {
		if len(files) > 0 {
			dataPath := filepath.Join(dir, DATA_FILE)
			pagePath := filepath.Join(dir, PAGE_FILE)
			renderPath := filepath.Join(dir, RENDER_FILE)
			handlerPath := filepath.Join(dir, HANDLER_FILE)

			var goFiles map[string]bool
			if _, ok := files[GO_EXT]; ok {
				goFiles = make(map[string]bool)
				goFiles = files[GO_EXT]
			}

			var htmlFiles map[string]bool
			if _, ok := files[HTML_EXT]; ok {
				htmlFiles = make(map[string]bool)
				htmlFiles = files[HTML_EXT]
			}

			ndir := removeDirWithUnderscorePostfix(dir)
			leafNode := filepath.Base(ndir)
			leafPath := ndir[5:]

			if _, ok := goFiles[dataPath]; ok {

				hasExpData, pkName, err := hasExportedDataVariable(dataPath)
				if err != nil {
					panic(err)
				}

				if hasExpData && pkName != "" {
					imports.Add(`"` + packageDir + filepath.Dir(dataPath) + `"`)
					routes = append(routes, formatDataVariable(pkName, leafPath, htmlFiles))
				}
			} else if _, ok := htmlFiles[pagePath]; ok {
				var fileDst string
				for k := range htmlFiles {
					if filepath.Base(k) == "page.html" {
						fileDst = removeDirWithUnderscorePostfix(k)[5:]
						break
					}
				}
				if fileDst == "" {
					log.Panicln("Please provide a data.go &/or page.html for directory:", dir)
				}

				err := utils.RenderGeneric(fileDst, g.OutputDir, utils.MapKeysToSlice(htmlFiles), struct{}{}, "")
				if err != nil {
					panic(err)
				}

			}

			if _, ok := goFiles[renderPath]; ok {
				expFns, pkName, err := getExportedFuctions(renderPath)
				if err != nil {
					panic(err)
				}
				if expFns != nil && pkName != "" {
					imports.Add(`"` + packageDir + filepath.Dir(renderPath) + `"`)
					for _, expFn := range expFns {
						if strings.HasSuffix(expFn, "_") {
							renderCustomFunctions = append(renderCustomFunctions, formatCustomFunction(pkName, expFn))
						} else {
							renderDefaultFunctions = append(renderDefaultFunctions, formatDefaultFunction(pkName, expFn, leafNode, "Render"))
						}
					}
				}
			}

			if _, ok := goFiles[handlerPath]; ok {
				expFns, pkName, err := getExportedFuctions(handlerPath)
				if err != nil {
					panic(err)
				}
				if expFns != nil && pkName != "" {
					imports.Add(`"` + packageDir + filepath.Dir(handlerPath) + `"`)
					for _, expFn := range expFns {
						if !strings.HasSuffix(expFn, "_") {
							handlerDefaultFunctions = append(handlerDefaultFunctions, formatDefaultFunction(pkName, expFn, leafNode, "Handler"))
						}
					}
				}
			}
		}
	}

	err = generateCode(imports, routes, renderCustomFunctions, renderDefaultFunctions, handlerDefaultFunctions)
	if err != nil {
		panic(err)
	}

	err = g.renderStaticFiles()
	if err != nil {
		panic(err)
	}
}

func (g *Gox) Run(r *mux.Router, port string, servePath string) {

	http.Handle("/", r)
	http.Handle(servePath, http.StripPrefix(servePath, http.FileServer(http.Dir(g.OutputDir))))

	g.handleRoutes(r)

	log.Fatal(http.ListenAndServe(port, nil))
}

// handleRoutes() binds Mux handlers to user defined functions, and creates default handlers to serve static pages
func (g *Gox) handleRoutes(r *mux.Router) {
	log.Println("----------------------CUSTOM HANDLERS----------------------")
	for _, route := range HandlerDefaultList {
		log.Println(route.Path + DIR)
		r.HandleFunc(route.Path+"{slash:/?}", route.Handler)
	}
	log.Println("---------------------STATIC PAGE HANDLERS-----------------------")
	for route := range RenderRouteList {
		log.Println(route + DIR)
		if route == "" {
			route = "/"
		}
		r.HandleFunc(route+"{slash:/?}",
			func(w http.ResponseWriter, r *http.Request) {
				if utils.IsHtmxRequest(r) {
					log.Println("Serving static page body:", filepath.Join(g.OutputDir, r.URL.Path, PAGE_BODY_FILE))
					http.ServeFile(w, r, filepath.Join(g.OutputDir, r.URL.Path, PAGE_BODY_FILE))
				} else {
					log.Println("Serving static page:", filepath.Join(g.OutputDir, r.URL.Path, PAGE_FILE))
					http.ServeFile(w, r, filepath.Join(g.OutputDir, r.URL.Path, PAGE_FILE))
				}
			},
		)
	}
	log.Println("---------------------STATIC COMPONENT HANDLERS-----------------------")
	for _, route := range RenderDefaultList {
		log.Println(route.Path + DIR)
		r.HandleFunc(route.Path+"{slash:/?}",
			func(w http.ResponseWriter, r *http.Request) {
				log.Println("Serving static component:", filepath.Join(g.OutputDir, r.URL.Path, PAGE_FILE))
				http.ServeFile(w, r, filepath.Join(g.OutputDir, r.URL.Path, PAGE_FILE))
			},
		)
	}
}

// RenderStaticFiles() renders all static files defined by the user
// Render hierarchy IF duplicate path: Custom Render -> Default Render -> Page Render
// Returns a map of all rendered paths
func (g *Gox) renderStaticFiles() error {

	for path, data := range RenderRouteList {
		err := utils.RenderGeneric(filepath.Join(path, PAGE_FILE), g.OutputDir, append(data.GeneratedTemplates, data.PageData.Templates...), data.PageData.Content, "")
		if err != nil {
			return err
		}
		err = utils.RenderGeneric(filepath.Join(path, PAGE_BODY_FILE), g.OutputDir, append(data.GeneratedTemplates, data.PageData.Templates...), data.PageData.Content, "body")
		if err != nil {
			return err
		}
	}

	for _, renderDefault := range RenderDefaultList {
		content, tmpls, tmplExec := renderDefault.Handler()
		err := utils.RenderGeneric(filepath.Join(renderDefault.Path, PAGE_FILE), g.OutputDir, tmpls, content, tmplExec)
		if err != nil {
			return err
		}
	}

	for _, renderCustom := range RenderCustomList {
		err := renderCustom.Handler()
		if err != nil {
			return err
		}
	}

	return nil
}

func walkDirectoryStructure(startDir string) (map[string]map[string]map[string]bool, error) {

	result := make(map[string]map[string]map[string]bool)

	err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && strings.HasPrefix(info.Name(), "_") {
			return filepath.SkipDir
		}

		if info.IsDir() && path != startDir {
			files := make(map[string]map[string]bool)

			filepath.Walk(path, func(innerPath string, innerInfo os.FileInfo, innerErr error) error {

				if innerInfo.IsDir() && strings.HasPrefix(innerInfo.Name(), "_") {
					return filepath.SkipDir
				}

				ext := filepath.Ext(innerPath)
				if !innerInfo.IsDir() && filepath.Dir(innerPath) == path && FILE_CHECK_LIST[filepath.Base(innerPath)] {
					if _, exists := files[ext]; !exists {
						files[ext] = make(map[string]bool)
					}
					files[ext][innerPath] = true
				}
				return nil
			})

			currDir := path
			for {
				indexFile := filepath.Join(currDir, INDEX_FILE)
				if _, err := os.Stat(indexFile); !os.IsNotExist(err) {
					if _, ok := files[filepath.Ext(indexFile)]; !ok {
						files[filepath.Ext(indexFile)] = make(map[string]bool)
					}
					files[filepath.Ext(indexFile)][indexFile] = true
					break
				}
				currDir = filepath.Dir(currDir)
				if currDir == "." || currDir == "/" {
					return errors.New("MISSING: " + INDEX_FILE)
				}
			}

			result[path] = files
		}
		return nil
	})

	return result, err
}

func formatDefaultFunction(pkName string, fnName string, leafPath string, rootPath string) string {
	if fnName == rootPath {
		return `{"/` + leafPath + `", ` + pkName + `.` + fnName + `},`
	} else {
		return `{"/` + leafPath + `/` + strings.ToLower(fnName) + `", ` + pkName + `.` + fnName + `},`
	}
}

func formatCustomFunction(pkName string, fnName string) string {
	return `{` + pkName + `.` + fnName + `},`
}

func formatDataVariable(pkName string, leafPath string, dirHtmlFiles map[string]bool) string {
	var additionalTemplates []string
	for file := range dirHtmlFiles {
		if filepath.Base(file) == INDEX_FILE {
			additionalTemplates = append([]string{`"` + file + `",`}, additionalTemplates...)
		} else {

			additionalTemplates = append(additionalTemplates, `"`+file+`",`)
		}
	}
	return `"` + leafPath + `": {PageData:` + pkName + `.` + "Data" + `, GeneratedTemplates: []string{` + strings.Join(additionalTemplates, " ") + `},},`
}

func getAstVals(path string) (*ast.File, error) {
	_, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func getExportedFuctions(path string) ([]string, string, error) {

	node, err := getAstVals(path)
	if err != nil {
		return nil, "", err
	}

	var pkName string
	var expFns []string
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.File:
			pkName = x.Name.Name
		case *ast.FuncDecl:
			if x.Name.IsExported() {
				expFns = append(expFns, x.Name.Name)
			}
		}
		return true
	})
	return expFns, pkName, nil
}

func hasExportedDataVariable(path string) (bool, string, error) {

	node, err := getAstVals(path)
	if err != nil {
		return false, "", err
	}

	hasData := false
	var pkName string
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.File:
			pkName = x.Name.Name
		case *ast.GenDecl:
			if x.Tok == token.VAR {
				for _, spec := range x.Specs {
					vspec := spec.(*ast.ValueSpec)
					if vspec.Names[0].Name == "Data" {
						hasData = true
						break
					}
				}
			}
		}
		return true
	})
	return hasData, pkName, nil
}

func generateCode(imports utils.StringSet, routes []string, renderCustomFunctions []string, renderDefaultFunctions []string, exportedHandlerFunctions []string) error {

	code := `
			// Code generated by gox; DO NOT EDIT.
			package gox
			import (
				` + imports.Join("\n\t") + `
			)

			var RenderRouteList = map[string]GeneratedPageData{
				` + strings.Join(routes, "\n\t") + `
			}

			var RenderCustomList = []RenderCustom{
				` + strings.Join(renderCustomFunctions, "\n\t") + `
			}

			var RenderDefaultList = []RenderDefault{
				` + strings.Join(renderDefaultFunctions, "\n\t") + `
			}

			var HandlerDefaultList = []HandlerDefault{
				` + strings.Join(exportedHandlerFunctions, "\n\t") + `
			}
			`
	log.Println(code)

	err := ioutil.WriteFile("../gox/generated.go", []byte(code), 0644)
	if err != nil {
		return err
	}
	return nil
}

func removeDirWithUnderscorePostfix(path string) string {
	segments := strings.Split(path, "/")
	var output []string
	if len(segments) == 0 {
		return path
	}
	for _, segment := range segments {
		if !strings.HasSuffix(segment, "_") {
			output = append(output, segment)
		}
	}

	return filepath.Join(output...)
}
