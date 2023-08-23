package utils

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type PageData struct {
	Content   interface{}
	Templates []string
}

func RenderGeneric[T any](filePath string, outputDir string, templates []string, v T, templateExec string) error {
	tmpl := template.Must(template.ParseFiles(templates...))
	file, err := CreateFile(filePath, outputDir)
	if err != nil {
		return err
	}

	err = WriteToFile(templateExec, file, tmpl, v)
	return err
}

func CreateFile(filePath string, outputDir string) (*os.File, error) {
	dir := filepath.Dir(outputDir + filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	file, err := os.Create(filepath.Join(outputDir, filePath))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func WriteToFile[T any](templateExec string, file *os.File, tmpl *template.Template, v T) error {
	defer file.Close()

	var err error
	if templateExec == "" {
		err = tmpl.Execute(file, v)
	} else {
		err = tmpl.ExecuteTemplate(file, templateExec, v)
	}
	return err
}

func HandleGeneric[T any](templates []string, v T, w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.ParseFiles(templates...))
	var err error

	if IsHtmxRequest(r) {
		err = tmpl.ExecuteTemplate(w, "body", v)
	} else {
		err = tmpl.Execute(w, v)
	}

	if err != nil {
		log.Fatalf("execution: %s", err)
	}
}

func IsHtmxRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

type Render struct{}

// DefaultRender enforces the return type required for GoX's default render handler. The name and relative path of your function will be the route.
// - d: A struct containing the data you want executed in your template.
// - t: A list of strings, where each string represents represents the path to a .html file you want executed.
// - x: A string that indicates the template you want executed. Use "" for no template execution
func (e Render) DefaultRender(d interface{}, t []string, x string) (interface{}, []string, string) {
	return d, t, x
}

// StringSet represents a collection of unique strings.
type StringSet map[string]struct{}

// New creates a new StringSet.
func NewStringSet() StringSet {
	return make(StringSet)
}

// Add inserts the item into the set.
func (s StringSet) Add(item string) {
	s[item] = struct{}{}
}

// Remove deletes the item from the set.
func (s StringSet) Remove(item string) {
	delete(s, item)
}

// Contains checks if the item is in the set.
func (s StringSet) Contains(item string) bool {
	_, exists := s[item]
	return exists
}

// Elements returns the elements of the set as a slice of strings.
func (s StringSet) Elements() []string {
	elements := make([]string, 0, len(s))
	for key := range s {
		elements = append(elements, key)
	}
	return elements
}

// Join concatenates the elements of the set using the provided separator.
func (s StringSet) Join(separator string) string {
	return strings.Join(s.Elements(), separator)
}
func MapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
