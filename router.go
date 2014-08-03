// Some utility functions to be refactored into their own packages.
//
// We want an api like:
// var router := NewRouter()
// router.Get("path", handler)
// router.Post("path", handler)
// router.Put("path", handler)
// router.Delete("path", handler)
//
// Once a user tries to register a path/handler we do
//   - analyse the path
package router

import (
	"net/http"
	"regexp"
	"strings"
)

// Stores
// ----------------------

// Store to keep track of the request parameters
var paramsStore = make(map[*http.Request]map[string]string)

// Router
// ----------------------

// A Router to register paths and requesthandlers to.
// There should be only one per application.
type Router struct {
	routes map[string][]*requestHandler
}

// NewRouter creates a router and returns a pointer to it so
// you can start registering routes.
//
// Dont forget to call `router.Handle(pattern)` to actually use
// the router.
func NewRouter() (router *Router) {
	router = new(Router)

	router.routes = map[string][]*requestHandler{
		"GET":    make([]*requestHandler, 0),
		"POST":   make([]*requestHandler, 0),
		"PUT":    make([]*requestHandler, 0),
		"DELETE": make([]*requestHandler, 0),
	}
	return
}

// Register a GET path to be handled.
func (router *Router) Get(path string, handler http.HandlerFunc) {
	router.registerRequestHandler("GET", path, handler)
}

// Register a POST path to be handled.
func (router *Router) Post(path string, handler http.HandlerFunc) {
	router.registerRequestHandler("POST", path, handler)
}

// Register a PUT path to be handled.
func (router *Router) Put(path string, handler http.HandlerFunc) {
	router.registerRequestHandler("PUT", path, handler)
}

// Register a DELETE path to be handled.
func (router *Router) Delete(path string, handler http.HandlerFunc) {
	router.registerRequestHandler("DELETE", path, handler)
}

// Handle registers the router for the given pattern in the DefaultServeMux.
// The documentation for ServeMux explains how patterns are matched.
//
// This just delegetes to `http.Handle()` internally.
//
// Most of the times, you just want to do `router.Handle("/")`.
func (router *Router) Handle(pattern string) {
	http.Handle(pattern, router)
}

// Needed by go to actually start handling the registered routes.
// You don't need to call this yourself.
func (router *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// For each of the registered routes for this request method...
	for _, reqHandler := range router.routes[req.Method] {
		// Only when the route matches...
		if isAMatch, withParams := reqHandler.matches(req.URL.Path); isAMatch {
			// Capture the route params
			paramsStore[req] = withParams
			// Fire the handler
			reqHandler.Handler(res, req)
			// Clean up
			delete(paramsStore, req)
			break
		}
	}
}

// Helper function to actually register the requestHandler on the router
func (router *Router) registerRequestHandler(method string, path string, handler http.HandlerFunc) {
	reqHandler := makeRequestHandler(path, handler)
	router.routes[method] = append(router.routes[method], reqHandler)
}

// Exported helper funcs
// ---------------------------

// Access the request parameters for a given request
func Params(req *http.Request) (reqParams map[string]string, ok bool) {
	reqParams, ok = paramsStore[req]
	return
}

// Private helper funcs
// ---------------------------

// Some paths use tokens like "/user/:userid" where "userid" is the token.
//
// This function builds a string to be compiled as a regexp to match those
// paths and returns the names of the parameters found in the route.
func buildRegexpFor(path string) (regexpPath string, withParamNames []string) {
	parts := strings.Split(path, "/")
	items := make([]string, 0)
	withParamNames = make([]string, 0)
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			nameOnly := strings.Trim(part, ":")
			withParamNames = append(withParamNames, nameOnly)
			items = append(items, `([^\/]+)`)
		} else {
			items = append(items, part)
		}
	}
	regexpPath = "^" + strings.Join(items, `\/`) + "$"
	return
}

// Creates the requestHandler struct from the given path
func makeRequestHandler(path string, handler http.HandlerFunc) (reqHandler *requestHandler) {
	regexpPath, withParamNames := buildRegexpFor(path)

	reqHandler = &requestHandler{
		Path:       path,
		ParamNames: withParamNames,
		Regex:      regexp.MustCompile(regexpPath),
		Tokenized:  len(withParamNames) != 0,
		Handler:    handler,
	}
	return
}

// RequestHandler
// --------------------------------

// RequestHandler stores info to evaluate if a route can be
// matched, for which params and which handlerFunc to dispatch.
type requestHandler struct {
	Path       string
	ParamNames []string
	Regex      *regexp.Regexp
	Tokenized  bool
	Handler    http.HandlerFunc
}

// requestHandler.matches checks if the given handler matches the given given string.
//
// It will also return to which uservalues the params evaluate for this path.
func (reqHandler *requestHandler) matches(path string) (isAMatch bool, withParams map[string]string) {
	withParams = make(map[string]string)
	isAMatch = false

	// Compare strings only when we know the path registered
	// does not contain tokens
	if !reqHandler.Tokenized {
		isAMatch = reqHandler.Path == path
		return
	}

	// Compare via regex when the path does contain tokens
	matches := reqHandler.Regex.FindAllStringSubmatch(path, -1)
	// Only try to find the params if we have a match
	if isAMatch = len(matches) != 0; isAMatch {
		for i, paramName := range reqHandler.ParamNames {
			withParams[paramName] = matches[0][i+1]
		}
	}
	return
}
