package router

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"
)

// func TestIsTrue(t *testing.T) {
// 	res := isTrue()
// 	fmt.Println(res)
// 	if !res {
// 		t.Error("Expected true, go ", res)
// 	}
// }

func TestIsTokenized(t *testing.T) {

	type testPair struct {
		input    string
		expected bool
	}

	testPairs := []testPair{
		{"/hello", false},
		{"/hello/world", false},
		{"/hello/:world", true},
		{"/hello/and/goodmorning", false},
		{"/hello/:and/good/:morning", true},
	}

	for _, test := range testPairs {
		if res := isTokenized(test.input); res != test.expected {
			t.Error("Expected ", test.expected, " got ", res)
		}
	}
}

func TestFindParams(t *testing.T) {

	type testPair struct {
		input    string
		expected map[string]string
	}

	testPairs := []testPair{
		{"/hello", make(map[string]string)},
		{"/hello/world", make(map[string]string)},
		{"/hello/:world", map[string]string{"world": "world"}},
		{"/hello/and/goodmorning", make(map[string]string)},
		{"/hello/:and/good/:morning", map[string]string{"and": "and", "morning": "morning"}},
	}

	for _, test := range testPairs {
		res := FindParams(test.input)
		// Empyty maps are not deepEqual...
		if len(test.expected) == 0 {
			if len(res) != 0 {
				t.Error("Expected ", test.expected, " got ", res)
			}
		} else if reflect.DeepEqual(test.expected, res) {
			t.Error("Expected ", test.expected, " got ", res)
		}
	}
}

func TestFindParamNames(t *testing.T) {

	type testPair struct {
		input    string
		expected []string
	}

	testPairs := []testPair{
		{"/hello", make([]string, 0)},
		{"/hello/world", make([]string, 0)},
		{"/hello/:world", []string{"world"}},
		{"/hello/and/goodmorning", make([]string, 0)},
		{"/hello/:and/good/:morning", []string{"and", "morning"}},
	}

	for _, test := range testPairs {
		res := FindParams(test.input)
		// Empyty maps are not deepEqual...
		if reflect.DeepEqual(test.expected, res) {
			t.Error("Expected ", test.expected, " got ", res)
		}
	}

}

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
		path     string
		expected bool
	}

	handler := func(w http.ResponseWriter, req *http.Request) {}
	requestHandler := makeRequestHandler("/hello", handler)

	testPairs := []testPair{
		{"/hello", true},
		{"/hello/", false},
		{"/helloo", false},
		{"/helo", false},
		{"/hello/something", false},
	}

	for _, test := range testPairs {
		isMatch := requestHandler.Matches(test.path)
		if isMatch != test.expected {
			t.Error("Expected ", test.expected, " got ", isMatch, " for path ", test.path)
		}
	}

	// Second...
	requestHandler = makeRequestHandler("/hello/world", handler)

	testPairs = []testPair{
		{"/hello", false},
		{"/hello/", false},
		{"/hello/world", true},
		{"/helloo/world", false},
		{"/hello/world/", false},
		{"/hello/something", false},
	}

	for _, test := range testPairs {
		isMatch := requestHandler.Matches(test.path)
		if isMatch != test.expected {
			t.Error("Expected ", test.expected, " got ", isMatch, " for path ", test.path)
		}
	}

	// Third...
	requestHandler = makeRequestHandler("/hello/:world", handler)

	testPairs = []testPair{
		{"/hello", false},
		{"/hello/", false},
		{"/hello/world", true},
		{"/hello/:world", true},
		{"/hello/14", true},
		{"/hello/15/", false},
		{"/hello/15/something", false},
	}

	for _, test := range testPairs {
		isMatch := requestHandler.Matches(test.path)
		if isMatch != test.expected {
			t.Error("Expected ", test.expected, " got ", isMatch, " for path ", test.path)
		}
	}

	// Fourth...
	requestHandler = makeRequestHandler("/hello/:world/and/:goodmorning", handler)

	testPairs = []testPair{
		{"/hello", false},
		{"/hello/:world/and/:goodmorning", true},
		{"/hello/12/and/54", true},
		{"/hello/16/and/something-else", true},
		{"/hello/:world/and/:goodmorning/", false},
		{"/hello/12/and/54/", false},
		{"/hello/16/and/something-else/", false},
		{"/hello/:world/and/:goodmorning/456", false},
	}

	for _, test := range testPairs {
		isMatch := requestHandler.Matches(test.path)
		if isMatch != test.expected {
			t.Error("Expected ", test.expected, " got ", isMatch, " for path ", test.path)
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
		w.Write([]byte("index"))
	}

	listHandler := func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("list"))
	}

	router.Get("/", indexHandler)
	router.Get("/list", listHandler)

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
		t.Error("Expected index as response but got ", string(body))
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
