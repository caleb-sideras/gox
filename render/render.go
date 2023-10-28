package render

import "html/template"

// The GoX Router will render and serve the full page
// - Templates: A list of strings, where each string represents represents the path to a .html file you want executed.
// - Content: A struct containing the data you want executed in your template.
// - Name: A string that indicates the name of the template you want executed. Use "" for no template execution
type StaticF struct {
	Templates []string
	Content   interface{}
	Name      string
}
type StaticFFunc func() StaticF

// The GoX Router will render both the body or full page and serve based on state
// - Templates: A list of strings, where each string represents represents the path to a .html file you want executed.
// - Content: A struct containing the data you want executed in your template.
type DynamicF struct {
	Templates []string
	Content   interface{}
}
type DynamicFFunc func() DynamicF

// The GoX Router will render and serve the full page
// - Template: Template object used for your page
// - Content: A struct containing the data you want executed in your template.
// - Name: A string that indicates the name of the template you want executed. Use "" for no template execution
type StaticT struct {
	Template *template.Template
	Content  interface{}
	Name     string
}
type StaticTFunc func() StaticT

// The GoX Router will render both the body or full page and serve based on state
// - Templates: Additional templates you want parsed with your page
// - Content: A struct containing the data you want executed in your template.
// - Template: Template object used for your page
type DynamicT struct {
	Templates []string
	Content   interface{}
	Template  *template.Template
}
type DynamicTFunc func() DynamicT
