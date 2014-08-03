package router

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"
)

func TestCreateRegExp(t *testing.T) {

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
		r, _ := createRegexp(test.input)
		if r != test.reg {
			t.Error("Expected ", test.reg, " got ", r)
		}
	}

	for _, test := range testPairs {
		_, p := createRegexp(test.input)
		if !reflect.DeepEqual(test.params, p) {
			t.Error("Expected ", test.params, " got ", p)
		}
	}
}

func TestMakeRequestHandler(t *testing.T) {

	type testPair struct {
		input    string
		expected RequestHandler
	}

	handleFunc := func(w http.ResponseWriter, req *http.Request) {}

	testPairs := []testPair{
		{input: "/hello",
			expected: RequestHandler{
				Path:       "/hello",
				ParamNames: make([]string, 0),
				Regex:      regexp.MustCompile(`^\/hello$`),
				Tokenized:  false,
				Handler:    handleFunc,
			},
		},
		{input: "/hello/world",
			expected: RequestHandler{
				Path:       "/hello/world",
				ParamNames: make([]string, 0),
				Regex:      regexp.MustCompile(`^\/hello\/world$`),
				Tokenized:  false,
				Handler:    handleFunc,
			},
		},
		{input: "/hello/:world",
			expected: RequestHandler{
				Path:       "/hello/:world",
				ParamNames: []string{"world"},
				Regex:      regexp.MustCompile(`^\/hello\/([^\/]+)$`),
				Tokenized:  true,
				Handler:    handleFunc,
			},
		},
		{input: "/hello/and/goodmorning",
			expected: RequestHandler{
				Path:       "/hello/and/goodmorning",
				ParamNames: make([]string, 0),
				Regex:      regexp.MustCompile(`^\/hello\/and\/goodmorning$`),
				Tokenized:  false,
				Handler:    handleFunc,
			},
		},
		{input: "/hello/:and/good/:morning",
			expected: RequestHandler{
				Path:       "/hello/:and/good/:morning",
				ParamNames: []string{"and", "morning"},
				Regex:      regexp.MustCompile(`^\/hello\/([^\/]+)\/good\/([^\/]+)$`),
				Tokenized:  true,
				Handler:    handleFunc,
			},
		},
	}

	for _, test := range testPairs {
		requestHandler := makeRequestHandler(test.input, handleFunc)
		if !isRequestHandlerDeepEqual(&test.expected, requestHandler) {
			t.Error("Expected ", test.expected, " got ", requestHandler)
		}
	}
}

func TestMatches(t *testing.T) {

	type testPair struct {
		path           string
		expectedMatch  bool
		expectedParams map[string]string
	}

	handler := func(w http.ResponseWriter, req *http.Request) {}
	requestHandler := makeRequestHandler("/hello", handler)

	testPairs := []testPair{
		{"/hello", true, make(map[string]string)},
		{"/hello/", false, make(map[string]string)},
		{"/helloo", false, make(map[string]string)},
		{"/helo", false, make(map[string]string)},
		{"/hello/something", false, make(map[string]string)},
	}

	for _, test := range testPairs {
		isAMatch, withParams := requestHandler.Matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}

	// Second...
	requestHandler = makeRequestHandler("/hello/world", handler)

	testPairs = []testPair{
		{"/hello", false, make(map[string]string)},
		{"/hello/", false, make(map[string]string)},
		{"/hello/world", true, make(map[string]string)},
		{"/helloo/world", false, make(map[string]string)},
		{"/hello/world/", false, make(map[string]string)},
		{"/hello/something", false, make(map[string]string)},
	}

	for _, test := range testPairs {
		isAMatch, withParams := requestHandler.Matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}

	// Third...
	requestHandler = makeRequestHandler("/hello/:world", handler)

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
		isAMatch, withParams := requestHandler.Matches(test.path)
		if isAMatch != test.expectedMatch {
			t.Error("Expected ", test.expectedMatch, " got ", isAMatch, " for path ", test.path)
		}
		if !reflect.DeepEqual(test.expectedParams, withParams) {
			t.Error("Expected ", test.expectedParams, " got ", withParams, " for path ", test.path)
		}
	}

	// Fourth...
	requestHandler = makeRequestHandler("/hello/:world/and/:goodmorning", handler)

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
		isAMatch, withParams := requestHandler.Matches(test.path)
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

	handler := func(w http.ResponseWriter, req *http.Request) {}

	// Add one
	router.registerRequestHandler("GET", "/hello", handler)

	if requestHandler := router.routes["GET"][0]; len(router.routes["GET"]) != 1 ||
		requestHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.registerRequestHandler("GET", "/hello/world", handler)

	if requestHandler := router.routes["GET"][1]; len(router.routes["GET"]) != 2 ||
		requestHandler.Path != "/hello/world" {
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

	handler := func(w http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Get("/hello", handler)

	if requestHandler := router.routes["GET"][0]; len(router.routes["GET"]) != 1 ||
		requestHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Get("/hello/world", handler)

	if requestHandler := router.routes["GET"][1]; len(router.routes["GET"]) != 2 ||
		requestHandler.Path != "/hello/world" {
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

	handler := func(w http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Post("/hello", handler)

	if requestHandler := router.routes["POST"][0]; len(router.routes["POST"]) != 1 ||
		requestHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Post("/hello/world", handler)

	if requestHandler := router.routes["POST"][1]; len(router.routes["POST"]) != 2 ||
		requestHandler.Path != "/hello/world" {
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

	handler := func(w http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Put("/hello", handler)

	if requestHandler := router.routes["PUT"][0]; len(router.routes["PUT"]) != 1 ||
		requestHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Put("/hello/world", handler)

	if requestHandler := router.routes["PUT"][1]; len(router.routes["PUT"]) != 2 ||
		requestHandler.Path != "/hello/world" {
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

	handler := func(w http.ResponseWriter, req *http.Request) {}

	// Add one
	router.Delete("/hello", handler)

	if requestHandler := router.routes["DELETE"][0]; len(router.routes["DELETE"]) != 1 ||
		requestHandler.Path != "/hello" {
		t.Error("Expected a first request handler to be registered")
	}

	// Add one more
	router.Delete("/hello/world", handler)

	if requestHandler := router.routes["DELETE"][1]; len(router.routes["DELETE"]) != 2 ||
		requestHandler.Path != "/hello/world" {
		t.Error("Expected a second request handler to be registered")
	}
}

// End-to-end ish
// --------------------------------

// This test is going to be more end-to-end ish
func TestServeHTTP(t *testing.T) {
	router := NewRouter()

	indexHandler := func(w http.ResponseWriter, req *http.Request) {
		params, _ := Params(req)
		if !reflect.DeepEqual(params, make(map[string]string)) {
			t.Error("Params do not watch")
		}
		w.Write([]byte("index"))
	}

	listHandler := func(w http.ResponseWriter, req *http.Request) {
		params, _ := Params(req)
		if !reflect.DeepEqual(params, make(map[string]string)) {
			t.Error("Params do not watch")
		}
		w.Write([]byte("list"))
	}

	userDetailHandler := func(w http.ResponseWriter, req *http.Request) {
		params, _ := Params(req)
		w.Write([]byte("user detail " + params["userid"]))
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

// Helpers....
// ---------------------------------

func isRequestHandlerDeepEqual(first *RequestHandler, second *RequestHandler) bool {
	if first.Path != second.Path ||
		!reflect.DeepEqual(first.ParamNames, second.ParamNames) ||
		!reflect.DeepEqual(first.Regex, second.Regex) ||
		first.Tokenized != second.Tokenized ||
		reflect.ValueOf(first.Handler) != reflect.ValueOf(second.Handler) {
		return false
	}
	return true
}
