package data

import (
	"net/http"
)

type Page struct {
	Content   interface{}
	Templates []string
}

type PageReturn struct {
	Page
	// on the fence about error. want it to be used in the future for conditional failures (similar to next)
	Error error
}

type PageFunc func(w http.ResponseWriter, r *http.Request) PageReturn
