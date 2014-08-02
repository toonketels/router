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
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var params map[*http.Request]map[string]string

type RequestHandler struct {
	Path       string
	ParamNames []string
	Regex      *regexp.Regexp
	Tokenized  bool
	Handler    http.HandlerFunc
}

// A Router to register paths and requesthandlers to.
// There should be only one per application.
type Router struct {
	routes       map[string][]*RequestHandler
	getRoutes    map[string]http.HandlerFunc
	postRoutes   map[string]http.HandlerFunc
	putRoutes    map[string]http.HandlerFunc
	deleteRoutes map[string]http.HandlerFunc
}

// NewRouter creates a router, starts handling those routes and
// returns a pointer to it.
func NewRouter() (router *Router) {
	router = new(Router)
	router.getRoutes = make(map[string]http.HandlerFunc)
	router.postRoutes = make(map[string]http.HandlerFunc)
	router.putRoutes = make(map[string]http.HandlerFunc)
	router.deleteRoutes = make(map[string]http.HandlerFunc)

	router.routes = map[string][]*RequestHandler{
		"GET":    make([]*RequestHandler, 0),
		"POST":   make([]*RequestHandler, 0),
		"PUT":    make([]*RequestHandler, 0),
		"DELETE": make([]*RequestHandler, 0),
	}

	params = make(map[*http.Request]map[string]string)

	http.Handle("/", router)
	return
}

func Params(req *http.Request) (reqParams map[string]string, ok bool) {
	reqParams, ok = params[req]
	return
}

// Register a GET path to be handled.
func (router *Router) Get(path string, handler http.HandlerFunc) {
	router.getRoutes[path] = handler
}

// Register a POST path to be handled.
func (router *Router) Post(path string, handler http.HandlerFunc) {
	router.postRoutes[path] = handler
}

// Register a PUT path to be handled.
func (router *Router) Put(path string, handler http.HandlerFunc) {
	router.putRoutes[path] = handler
}

// Register a DELETE path to be handled.
func (router *Router) Delete(path string, handler http.HandlerFunc) {
	router.deleteRoutes[path] = handler
}

// Private API to start handling the registered routes.
func (router *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Method, req.URL)

	if req.Method == "GET" {
		if handle, ok := router.getRoutes[req.URL.Path]; ok {
			// params[req] = map[string]string{
			// 	"userid": "34",
			// }
			params[req] = FindParams(req.URL.Path)
			handle(res, req)
			fmt.Println("utils params", params)
			delete(params, req)
		}

		res.Write([]byte("You issued a GET request"))
	}

	if req.Method == "POST" {
		if handle, ok := router.postRoutes[req.URL.Path]; ok {
			handle(res, req)
		}
		res.Write([]byte("You issued a POST request"))
	}

	if req.Method == "PUT" {
		if handle, ok := router.putRoutes[req.URL.Path]; ok {
			handle(res, req)
		}
		res.Write([]byte("You issued a PUT request"))
	}

	if req.Method == "DELETE" {
		if handle, ok := router.deleteRoutes[req.URL.Path]; ok {
			handle(res, req)
		}
		res.Write([]byte("You issued a DELETE request"))
	}

	res.Write([]byte("Yes we did it"))
}

func FindParams(path string) map[string]string {
	parts := strings.Split(path, "/")
	items := make(map[string]string)
	for i, value := range parts {
		if strings.HasPrefix(value, ":") {
			trimmed := strings.Trim(value, ":")
			items[trimmed] = trimmed + string(i)
		}
	}
	return items
}

func findParamNames(path string) []string {
	parts := strings.Split(path, "/")
	items := make([]string, 0)
	for _, value := range parts {
		if strings.HasPrefix(value, ":") {
			trimmed := strings.Trim(value, ":")
			items = append(items, trimmed)
		}
	}
	return items
}

func createRegexp(path string) (string, []string) {
	parts := strings.Split(path, "/")
	items := make([]string, 0)
	params := make([]string, 0)
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			nameOnly := strings.Trim(part, ":")
			params = append(params, nameOnly)
			items = append(items, `([^\/]+)`)
		} else {
			items = append(items, part)
		}
	}
	regStr := "^" + strings.Join(items, `\/`) + "$"
	return regStr, params
}

func isTokenized(path string) bool {
	return strings.Contains(path, ":")
}

func (router *Router) registerRequestHandler(method string, path string, handler http.HandlerFunc) {
	requestHandler := makeRequestHandler(path, handler)
	router.routes[method] = append(router.routes[method], requestHandler)
}

// Creates the RequestHandler struct from the given path
func makeRequestHandler(path string, handler http.HandlerFunc) (requestHandler *RequestHandler) {
	regStr, params := createRegexp(path)

	requestHandler = &RequestHandler{
		Path:       path,
		ParamNames: params,
		Regex:      regexp.MustCompile(regStr),
		Tokenized:  len(params) != 0,
		Handler:    handler,
	}
	return
}

func (requestHandler *RequestHandler) Matches(path string) bool {
	if !requestHandler.Tokenized {
		return requestHandler.Path == path
	}
	matches := requestHandler.Regex.FindAllStringSubmatch(path, -1)
	return len(matches) != 0
}
