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
var requestContextStore = make(map[*http.Request]*RequestContext)

// Router
// ----------------------

// A Router to register paths and requestHandlers to.
//
// Set a custom NotFoundHandler if you want to override go's default one.
//
// There can be multiple per application, if so, don't forget to pass a
// different pattern to `router.Handle()`.
type Router struct {
	routes map[string][]*requestHandler
	// Specify a custom NotFoundHandler
	NotFoundHandler http.HandlerFunc
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
func (router *Router) Get(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("GET", path, handlers...)
}

// Register a POST path to be handled.
func (router *Router) Post(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("POST", path, handlers...)
}

// Register a PUT path to be handled.
func (router *Router) Put(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("PUT", path, handlers...)
}

// Register a DELETE path to be handled.
func (router *Router) Delete(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("DELETE", path, handlers...)
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
	unMatched := true

	// For each of the registered routes for this request method...
	for _, reqHandler := range router.routes[req.Method] {
		// Only when the route matches...
		if isAMatch, withParams := reqHandler.matches(req.URL.Path); isAMatch {
			unMatched = false

			// Capture the route params
			paramsStore[req] = withParams
			// Create a RequestContext
			contxt := new(RequestContext)
			requestContextStore[req] = contxt
			// Call the handlers
			dispatchHandlers(reqHandler, res, req, contxt)
			// Clean up
			delete(paramsStore, req)
			delete(requestContextStore, req)
			break
		}
	}

	// Nothing found...
	if unMatched {
		router.notFound(res, req)
	}
}

// Helper function to actually register the requestHandler on the router
func (router *Router) registerRequestHandler(method string, path string, handlers ...http.HandlerFunc) {
	reqHandler := makeRequestHandler(path, handlers...)
	router.routes[method] = append(router.routes[method], reqHandler)
}

// Helper function to dispatch the correct NotFoundHanler.
func (router *Router) notFound(res http.ResponseWriter, req *http.Request) {
	if router.NotFoundHandler != nil {
		router.NotFoundHandler(res, req)
	} else {
		http.NotFound(res, req)
	}
}

// Exported helper funcs
// ---------------------------

func getPreHandlers(handlers []http.HandlerFunc) (preHandlers []http.HandlerFunc) {
	preHandlers = make([]http.HandlerFunc, len(handlers)-1)
	copy(preHandlers, handlers)
	return
}

// Access the request parameters for a given request
func Params(req *http.Request) (reqParams map[string]string, ok bool) {
	reqParams, ok = paramsStore[req]
	return
}

// Private helper funcs
// ---------------------------

// Attaches the handlers to the context and starts dispatching the first one.
func dispatchHandlers(reqHandler *requestHandler, res http.ResponseWriter, req *http.Request, cntxt *RequestContext) {
	cntxt.handlers = reqHandler.Handlers
	cntxt.Next(res, req)
}

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
func makeRequestHandler(path string, handlers ...http.HandlerFunc) (reqHandler *requestHandler) {
	regexpPath, withParamNames := buildRegexpFor(path)

	reqHandler = &requestHandler{
		Path:       path,
		ParamNames: withParamNames,
		Regex:      regexp.MustCompile(regexpPath),
		Tokenized:  len(withParamNames) != 0,
		Handlers:   handlers,
	}
	return
}

// Context
// --------------------------------

// RequestContext contains data related to the current request
type RequestContext struct {
	Error          error
	Final          bool
	handlers       []http.HandlerFunc
	currentHandler int
}

// Context returns a pointer to the RequestContext for the current request.
func Context(req *http.Request) (cntxt *RequestContext) {
	cntxt = requestContextStore[req]
	return
}

// RequestContext.Next() allows a http.HandleFunc to invoke the next HandleFunc.
// This is useful when multiple HandleFuncs are registered for a given path
// and allows the creation and use of `middleware`.
func (cntxt *RequestContext) Next(res http.ResponseWriter, req *http.Request) {
	var handler http.HandlerFunc
	if len(cntxt.handlers) < cntxt.currentHandler+1 {
		handler = func(res http.ResponseWriter, req *http.Request) {}
	} else {
		handler = cntxt.handlers[cntxt.currentHandler]
	}
	cntxt.currentHandler++
	handler(res, req)
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
	Handlers   []http.HandlerFunc
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
