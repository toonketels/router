package router

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"
)

func TestBuildRegexpFor(t *testing.T) {

	type testPair struct {
		input  string
		reg    string
		params []string
	}

	testPairs := []testPair{
		{"/hello", `^\/hello$`, make([]string, 0)},
		{"/hello/world", `^\/hello\/world$`, make([]string, 0)},
		{"/hello/:world", `^\/hello\/([^\/]+)$`, []string{"world"}},
		{"/hello/and/goodmorning", `^\/hello\/and\/goodmorning$`, make([]string, 0)},
		{"/hello/:and/good/:morning", `^\/hello\/([^\/]+)\/good\/([^\/]+)$`, []string{"and", "morning"}},
	}

	for _, test := range testPairs {
		r, _ := buildRegexpFor(test.input)
		if r != test.reg {
			t.Error("Expected ", test.reg, " got ", r)
		}
	}

	for _, test := range testPairs {
		_, p := buildRegexpFor(test.input)
		if !reflect.DeepEqual(test.params, p) {
			t.Error("Expected ", test.params, " got ", p)
		}
	}
}

func TestMakeRequestHandler(t *testing.T) {

	type testPair struct {
		input    string
		expected requestHandler
	}

	aRouter := NewRouter()
	handleFunc := func(res http.ResponseWriter, req *http.Request) {}

	testPairs := []testPair{
		{input: "/hello",
			expected: requestHandler{
				Path:       "/hello",
				ParamNames: make([]string, 0),
				Regex:      regexp.MustCompile(`^\/hello$`),
				Tokenized:  false,
				Handlers:   []http.HandlerFunc{handleFunc},
			},
		},
		{input: "/hello/world",
			expected: requestHandler{
				Path:       "/hello/world",
				ParamNames: make([]string, 0),
				Regex:      regexp.MustCompile(`^\/hello\/world$`),
				Tokenized:  false,
				Handlers:   []http.HandlerFunc{handleFunc},
			},
		},
		{input: "/hello/:world",
			expected: requestHandler{
				Path:       "/hello/:world",
				ParamNames: []string{"world"},
				Regex:      regexp.MustCompile(`^\/hello\/([^\/]+)$`),
				Tokenized:  true,
				Handlers:   []http.HandlerFunc{handleFunc},
			},
		},
		{input: "/hello/and/goodmorning",
			expected: requestHandler{
				Path:       "/hello/and/goodmorning",
				ParamNames: make([]string, 0),
				Regex:      regexp.MustCompile(`^\/hello\/and\/goodmorning$`),
				Tokenized:  false,
				Handlers:   []http.HandlerFunc{handleFunc},
			},
		},
		{input: "/hello/:and/good/:morning",
			expected: requestHandler{
				Path:       "/hello/:and/good/:morning",
				ParamNames: []string{"and", "morning"},
				Regex:      regexp.MustCompile(`^\/hello\/([^\/]+)\/good\/([^\/]+)$`),
				Tokenized:  true,
				Handlers:   []http.HandlerFunc{handleFunc},
			},
		},
	}

	for _, test := range testPairs {
		reqHandler := aRouter.makeRequestHandler(test.input, handleFunc)
		if !isRequestHandlerDeepEqual(&test.expected, reqHandler) {
			t.Error("Expected ", test.expected, " got ", reqHandler)
		}
	}
}

func TestMatches(t *testing.T) {

	type testPair struct {
		path           string
		expectedMatch  bool
		expectedParams map[string]string
	}

	aRouter := NewRouter()
	handler := func(res http.ResponseWriter, req *http.Request) {}
	reqHandler := aRouter.makeRequestHandler("/hello", handler)

	testPairs := []testPair{
		{"/hello", true, make(map[string]string)},
		{"/hello/", false, make(map[string]string)},
		{"/helloo", false, make(map[string]string)},
		{"/helo", false, make(map[string]string)},
		{"/hello/something", false, make(map[string]string)},
	}

	for _, test := range testPairs {
		isAMatch, withParams := reqHandler.matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}

	// Second...
	reqHandler = aRouter.makeRequestHandler("/hello/world", handler)

	testPairs = []testPair{
		{"/hello", false, make(map[string]string)},
		{"/hello/", false, make(map[string]string)},
		{"/hello/world", true, make(map[string]string)},
		{"/helloo/world", false, make(map[string]string)},
		{"/hello/world/", false, make(map[string]string)},
		{"/hello/something", false, make(map[string]string)},
	}

	for _, test := range testPairs {
		isAMatch, withParams := reqHandler.matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}

	// Third...
	reqHandler = aRouter.makeRequestHandler("/hello/:world", handler)

	testPairs = []testPair{
		{"/hello", false, make(map[string]string)},
		{"/hello/", false, make(map[string]string)},
		{"/hello/world", true, map[string]string{"world": "world"}},
		{"/hello/:world", true, map[string]string{"world": ":world"}},
		{"/hello/14", true, map[string]string{"world": "14"}},
		{"/hello/15/", false, make(map[string]string)},
		{"/hello/15/something", false, make(map[string]string)},
	}

	for _, test := range testPairs {
		isAMatch, withParams := reqHandler.matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}

	// Fourth...
	reqHandler = aRouter.makeRequestHandler("/hello/:world/and/:goodmorning", handler)

	testPairs = []testPair{
		{"/hello", false, make(map[string]string)},
		{"/hello/:world/and/:goodmorning", true, map[string]string{"world": ":world", "goodmorning": ":goodmorning"}},
		{"/hello/12/and/54", true, map[string]string{"world": "12", "goodmorning": "54"}},
		{"/hello/16/and/something-else", true, map[string]string{"world": "16", "goodmorning": "something-else"}},
		{"/hello/:world/and/:goodmorning/", false, make(map[string]string)},
		{"/hello/12/and/54/", false, make(map[string]string)},
		{"/hello/16/and/something-else/", false, make(map[string]string)},
		{"/hello/:world/and/:goodmorning/456", false, make(map[string]string)},
	}

	for _, test := range testPairs {
		isAMatch, withParams := reqHandler.matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}
}

func TestRegisterRequestHandler(t *testing.T) {
	router := NewRouter()

	// There should be no handlers registered
	if len(router.routes["GET"]) > 0 ||
		len(router.routes["POST"]) > 0 ||
		len(router.routes["PUT"]) > 0 ||
		len(router.routes["DELETE"]) > 0 {
		t.Error("No handlers should be registered yet")
	}

	handler := func(res http.ResponseWriter, req *http.Request) {}

	// Add one
	router.registerRequestHandler("GET", "/hello", handler)

	if reqHandler := router.routes["GET"][0]; len(router.routes["GET"]) != 1 ||
		reqHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.registerRequestHandler("GET", "/hello/world", handler)

	if reqHandler := router.routes["GET"][1]; len(router.routes["GET"]) != 2 ||
		reqHandler.Path != "/hello/world" {
		t.Error("Expected a first request handler to be registered")
	}
}

func TestGet(t *testing.T) {
	router := NewRouter()

	// Should not have ony get handlers
	// There should be no handlers registered
	if len(router.routes["GET"]) > 0 {
		t.Error("No GET handlers should be registered yet")
	}

	handler := func(res http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Get("/hello", handler)

	if reqHandler := router.routes["GET"][0]; len(router.routes["GET"]) != 1 ||
		reqHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Get("/hello/world", handler)

	if reqHandler := router.routes["GET"][1]; len(router.routes["GET"]) != 2 ||
		reqHandler.Path != "/hello/world" {
		t.Error("Expected a second request handler to be registered")
	}
}

func TestPost(t *testing.T) {
	router := NewRouter()

	// Should not have ony get handlers
	// There should be no handlers registered
	if len(router.routes["POST"]) > 0 {
		t.Error("No POST handlers should be registered yet")
	}

	handler := func(res http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Post("/hello", handler)

	if reqHandler := router.routes["POST"][0]; len(router.routes["POST"]) != 1 ||
		reqHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Post("/hello/world", handler)

	if reqHandler := router.routes["POST"][1]; len(router.routes["POST"]) != 2 ||
		reqHandler.Path != "/hello/world" {
		t.Error("Expected a second request handler to be registered")
	}
}

func TestPut(t *testing.T) {
	router := NewRouter()

	// Should not have ony get handlers
	// There should be no handlers registered
	if len(router.routes["PUT"]) > 0 {
		t.Error("No PUT handlers should be registered yet")
	}

	handler := func(res http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Put("/hello", handler)

	if reqHandler := router.routes["PUT"][0]; len(router.routes["PUT"]) != 1 ||
		reqHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Put("/hello/world", handler)

	if reqHandler := router.routes["PUT"][1]; len(router.routes["PUT"]) != 2 ||
		reqHandler.Path != "/hello/world" {
		t.Error("Expected a second request handler to be registered")
	}
}

func TestDelete(t *testing.T) {
	router := NewRouter()

	// Should not have ony get handlers
	// There should be no handlers registered
	if len(router.routes["DELETE"]) > 0 {
		t.Error("No DELETE handlers should be registered yet")
	}

	handler := func(res http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Delete("/hello", handler)

	if reqHandler := router.routes["DELETE"][0]; len(router.routes["DELETE"]) != 1 ||
		reqHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Delete("/hello/world", handler)

	if reqHandler := router.routes["DELETE"][1]; len(router.routes["DELETE"]) != 2 ||
		reqHandler.Path != "/hello/world" {
		t.Error("Expected a second request handler to be registered")
	}
}

// Tests registration of middlewareHandlers
func TestUse(t *testing.T) {
	aRouter := NewRouter()

	firstMReqHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("first"))
		Context(req).Next(res, req)
	}

	secondMReqHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("second"))
		Context(req).Next(res, req)
	}

	thirdMReqHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("third"))
	}

	if len(aRouter.middleware) != 0 {
		t.Error("A new router should not have middleware, we got ", len(aRouter.middleware))
	}

	aRouter.Use("/", firstMReqHandler)
	if len(aRouter.middleware) != 1 ||
		reflect.ValueOf(aRouter.middleware[0].Handle).Pointer() != reflect.ValueOf(firstMReqHandler).Pointer() ||
		aRouter.middleware[0].MountPath != "/" {
		t.Error("The middleware should have been registered to the router")
	}

	aRouter.Use("/api", secondMReqHandler)
	if len(aRouter.middleware) != 2 ||
		reflect.ValueOf(aRouter.middleware[1].Handle).Pointer() != reflect.ValueOf(secondMReqHandler).Pointer() ||
		aRouter.middleware[1].MountPath != "/api" {
		t.Error("The middleware should have been registered to the router")
	}

	aRouter.Use("/", thirdMReqHandler)
	if len(aRouter.middleware) != 3 ||
		reflect.ValueOf(aRouter.middleware[2].Handle).Pointer() != reflect.ValueOf(thirdMReqHandler).Pointer() ||
		aRouter.middleware[2].MountPath != "/" {
		t.Error("The middleware should have been registered to the router")
	}
}

// End-to-end ish
// --------------------------------

// Test if the correct handlerfuncs are dispatched for each registered path.
func TestServeHTTP(t *testing.T) {
	router := NewRouter()

	indexHandler := func(res http.ResponseWriter, req *http.Request) {
		params := Context(req).Params
		if !reflect.DeepEqual(params, make(map[string]string)) {
			t.Error("Params do not watch")
		}
		res.Write([]byte("index"))
	}

	listHandler := func(res http.ResponseWriter, req *http.Request) {
		params := Context(req).Params
		if !reflect.DeepEqual(params, make(map[string]string)) {
			t.Error("Params do not watch")
		}
		res.Write([]byte("list"))
	}

	userDetailHandler := func(res http.ResponseWriter, req *http.Request) {
		params := Context(req).Params
		res.Write([]byte("user detail " + params["userid"]))
	}

	router.Get("/", indexHandler)
	router.Get("/list", listHandler)
	router.Get("/user/:userid", userDetailHandler)

	server := httptest.NewServer(router)
	defer server.Close()

	res, _ := http.Get(server.URL)
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()

	if string(body) != "index" {
		t.Error("Expected index as response but got ", string(body))
	}

	res, _ = http.Get(server.URL + "/list")
	body, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if string(body) != "list" {
		t.Error("Expected list as response but got ", string(body))
	}

	res, _ = http.Get(server.URL + "/user/14")
	body, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if string(body) != "user detail 14" {
		t.Error("Expected 'user detail 14' as response but got ", string(body))
	}

	res, _ = http.Get(server.URL + "/user/420")
	body, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if string(body) != "user detail 420" {
		t.Error("Expected 'user detail 420' as response but got ", string(body))
	}
}

// Test if we can pass multipel handlerFuncs for a given path
// This test is going to be more end-to-end ish
func TestRegisterMultiple(t *testing.T) {
	router := NewRouter()

	firstHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("first"))
		Context(req).Next(res, req)
	}

	secondHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("second"))
		Context(req).Next(res, req)
	}

	thirdHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("third"))
	}

	router.Get("/", firstHandler, secondHandler, thirdHandler)

	server := httptest.NewServer(router)
	defer server.Close()

	res, _ := http.Get(server.URL)
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()

	// It should invoke all three handlers
	if string(body) != "firstsecondthird" {
		t.Error("Expected 'firstsecondthird' as response but got ", string(body))
	}

	// Second
	// -------------------------
	router = NewRouter()

	// It should invoke only the first two handlers
	secondHandler = func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("second"))
	}

	router.Get("/", firstHandler, secondHandler, thirdHandler)

	server = httptest.NewServer(router)
	defer server.Close()

	res, _ = http.Get(server.URL)
	body, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if string(body) != "firstsecond" {
		t.Error("Expected 'firstsecond' as response but got ", string(body))
	}
}

// Test mounting and dispatching middleware
func TestMiddleware(t *testing.T) {
	aRouter := NewRouter()

	indexReqHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("index mReqHandler"))
		Context(req).Next(res, req)
	}

	apiReqHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("api mReqHandler"))
		Context(req).Next(res, req)
	}

	firstHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("first"))
		Context(req).Next(res, req)
	}

	secondHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("second"))
		Context(req).Next(res, req)
	}

	thirdHandler := func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("third"))
	}

	// Register middleware
	aRouter.Use("/", indexReqHandler)
	aRouter.Use("/api", apiReqHandler)

	// Register routes
	aRouter.Get("/", secondHandler, thirdHandler)
	aRouter.Get("/api", firstHandler, secondHandler)

	server := httptest.NewServer(aRouter)
	defer server.Close()

	res, _ := http.Get(server.URL)
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()

	// It should invoke all three handlers
	if string(body) != "index mReqHandlersecondthird" {
		t.Error("Expected 'index mReqHandlersecondthird' as response but got ", string(body))
	}

	res, _ = http.Get(server.URL + "/api")
	body, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	if string(body) != "index mReqHandlerapi mReqHandlerfirstsecond" {
		t.Error("Expected 'index mReqHandlerapi mReqHandlerfirstsecond' as response but got ", string(body))
	}
}

// Helpers....
// ---------------------------------

func isRequestHandlerDeepEqual(first *requestHandler, second *requestHandler) bool {
	if first.Path != second.Path ||
		!reflect.DeepEqual(first.ParamNames, second.ParamNames) ||
		!reflect.DeepEqual(first.Regex, second.Regex) ||
		first.Tokenized != second.Tokenized ||
		!isHandlersSliceDeepEqual(first.Handlers, second.Handlers) {
		return false
	}
	return true
}

func isHandlersSliceDeepEqual(first []http.HandlerFunc, second []http.HandlerFunc) bool {
	if len(first) != len(second) {
		return false
	}
	for i, fun := range first {
		if reflect.ValueOf(fun) != reflect.ValueOf(second[i]) {
			return false
		}
	}
	return true
}
