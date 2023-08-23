package gox

import (
	"github.com/caleb-sideras/gox/utils"
	"net/http"
)

type GeneratedPageData struct {
	utils.PageData
	GeneratedTemplates []string
}
type HandlerDefaultFunc func(http.ResponseWriter, *http.Request)
type HandlerDefault struct {
	Path    string
	Handler HandlerDefaultFunc
}

type RenderCustomFunc func() error
type RenderCustom struct {
	Handler RenderCustomFunc
}

type RenderDefaultFunc func() (interface{}, []string, string)
type RenderDefault struct {
	Path    string
	Handler RenderDefaultFunc
}
