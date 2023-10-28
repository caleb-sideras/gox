package utils

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type RenderCustomFunc func() error
type RenderCustom struct {
	Handler RenderCustomFunc
}

func RenderFile[T any](filePath string, outputDir string, templates []string, v T, templateExec string) error {
	tmpl := template.Must(template.ParseFiles(templates...))
	return RenderTemplate(filePath, outputDir, tmpl, v, templateExec)
}

func RenderTemplate[T any](filePath string, outputDir string, tmpl *template.Template, v T, templateExec string) error {
	file, err := CreateFile(filePath, outputDir)
	if err != nil {
		return err
	}

	err = WriteToFile(templateExec, file, tmpl, v)
	return err
}

func RenderFileTemplateIndex[T any](filePath string, outputDir string, index string, tmpls []string, tmpl *template.Template, v T) error {
	parentTmpl := template.Must(template.ParseFiles(index))

	_, err := parentTmpl.ParseFiles(tmpls...)
	if err != nil {
		log.Fatal(err)
	}

	_, err = parentTmpl.New("page").Parse(tmpl.Tree.Root.String())
	if err != nil {
		log.Fatal(err)
	}

	return RenderTemplate(filePath, outputDir, parentTmpl, v, "")
}

func RenderFileTemplatePage[T any](filePath string, outputDir string, tmpls []string, tmpl *template.Template, v T) error {
	_, err := tmpl.ParseFiles(tmpls...)
	if err != nil {
		log.Fatal(err)
	}
	return RenderTemplate(filePath, outputDir, tmpl, v, "")
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

func GetHtmxRequestURL(r *http.Request) string {
	return r.Header.Get("HX-Current-URL")
}

func IsHxBoosted(r *http.Request) bool {
	if r.Header.Get("HX-Boosted") == "true" {
		return true
	} else {
		return false
	}
}

func LastElementOfURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if u.Path == "" || u.Path == "/" {
		return "/", nil
	}

	return u.Path, nil
}
func GenerateETag(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf(`"%x"`, hash)
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
