package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/gomega"
)

// a CheckRequestFn has expectations on the request made
type CheckRequestFn func(r *http.Request)

// a MakeRequestFn makes a request to the MockServer
type MakeRequestFn func() (*http.Response, error)

type MockServer struct {
	*httptest.Server
	HandlerFn http.HandlerFunc
	RequestFn CheckRequestFn
	Called    bool
}

// QuickMockServer creates a server with a handler according to the arguments:
// path - handler expects the given path, all others return StatusNotImplemented
// code - handler will return the given code
// body - handler response will contain given body
func QuickMockServer(path string, code int, body string) *MockServer {
	ms := &MockServer{}
	ms.HandlerFn = func(w http.ResponseWriter, r *http.Request) {
		ms.Called = true
		if r.URL.String() != path {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		w.WriteHeader(code)
		_, err := w.Write([]byte(body))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		return
	}
	ms.Server = httptest.NewServer(ms.HandlerFn)
	return ms
}

func ErrorJSON(errorKey, errorDesc string) string {
	m := map[string]string{
		"error":             errorKey,
		"error_description": errorDesc,
	}
	j, err := json.Marshal(m)
	Expect(err).To(BeNil())
	return string(j)
}

// Check runs MakeRequestFn, tests that the server handler was called, and
// closes the server.  If the MakeRequestFn doesn't return a response directly, it can return (nil, nil)
func (ms *MockServer) Check(fn MakeRequestFn) {
	Expect(ms.Server).NotTo(BeNil())
	defer ms.Server.Close()
	resp, err := fn()
	Expect(err).To(BeNil())
	if resp != nil {
		defer resp.Body.Close()
	}
	Expect(ms.Called).To(BeTrue())
	return
}

func RequestChecker(crfn CheckRequestFn) *MockServer {
	ms := &MockServer{RequestFn: crfn}
	ms.HandlerFn = func(w http.ResponseWriter, r *http.Request) {
		ms.Called = true
		Expect(ms.RequestFn).NotTo(BeNil())
		ms.RequestFn(r)
	}
	ms.Server = httptest.NewServer(ms.HandlerFn)
	return ms
}

// an AuthMockServer expects an Authorization header with the given token if token is non-empty.  If token is empty, the Authorization header is expected to be absent
func AuthMockServer(token string) *MockServer {
	return RequestChecker(func(r *http.Request) {
		if token == "" {
			Expect(r.Header.Get("Authorization")).To(BeEmpty())
		} else {
			Expect(r.Header.Get("Authorization")).To(Equal("Bearer " + token))
		}
	})
}

// a MethodMockServer expects the request method to equal the method argument
func MethodMockServer(method string) *MockServer {
	return RequestChecker(func(r *http.Request) {
		Expect(r.Method).To(Equal(method))
	})
}
