package httpclient_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nitrous-io/nitro/pkg/httpclient"
	"github.com/nitrous-io/nitro/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "httpclient")
}

var _ = Describe("httpclient", func() {
	var (
		c      *httpclient.Client
		ms     *test.MockServer
		check  func()
		token  = "correct_token"
		params = &url.Values{
			"containers": {"arch", "ubuntu"},
			"say":        {"hello world!"},
		}
	)

	BeforeEach(func() {
		ms = nil
		check = nil
		c = httpclient.New(token)
	})

	ItHasAuthHeader := func() {
		It("has auth header", func() {
			ms = test.AuthMockServer(token)
			check()
		})
	}

	ItHasNoAuthHeader := func() {
		It("has no auth header", func() {
			ms = test.AuthMockServer("")
			check()
		})
	}

	ItUsesHTTPVerb := func(verb string) {
		It("is a "+verb+" request", func() {
			ms = test.MethodMockServer(verb)
			check()
		})
	}

	ItIncludesQueryString := func() {
		It("includes query string", func() {
			ms = test.RequestChecker(func(r *http.Request) {
				Expect(strings.HasSuffix(r.URL.String(), "?containers=arch&containers=ubuntu&say=hello+world%21")).To(BeTrue())
			})
			check()
		})
	}

	ItIncludesFormData := func() {
		It("includes form data", func() {
			ms = test.RequestChecker(func(r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				Expect(err).To(BeNil())
				Expect(body).To(Equal([]byte("containers=arch&containers=ubuntu&say=hello+world%21")))
			})
			check()
		})
	}

	Describe("Get()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.Get(ms.URL, params)
				})
			}
		})

		ItHasNoAuthHeader()
		ItUsesHTTPVerb("GET")
		ItIncludesQueryString()
	})

	Describe("Delete()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.Delete(ms.URL, params)
				})
			}
		})

		ItHasNoAuthHeader()
		ItUsesHTTPVerb("DELETE")
		ItIncludesQueryString()
	})

	Describe("PostForm()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.PostForm(ms.URL, params)
				})
			}
		})

		ItHasNoAuthHeader()
		ItUsesHTTPVerb("POST")
		ItIncludesFormData()
	})

	Describe("PutForm()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.PutForm(ms.URL, params)
				})
			}
		})

		ItHasNoAuthHeader()
		ItUsesHTTPVerb("PUT")
		ItIncludesFormData()
	})

	Describe("GetAuthed()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.GetAuthed(ms.URL, params)
				})
			}
		})

		ItHasAuthHeader()
		ItUsesHTTPVerb("GET")
		ItIncludesQueryString()
	})

	Describe("DeleteAuthed()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.DeleteAuthed(ms.URL, params)
				})
			}
		})

		ItHasAuthHeader()
		ItUsesHTTPVerb("DELETE")
		ItIncludesQueryString()
	})

	Describe("PostFormAuthed()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.PostFormAuthed(ms.URL, params)
				})
			}
		})

		ItHasAuthHeader()
		ItUsesHTTPVerb("POST")
		ItIncludesFormData()
	})

	Describe("PutFormAuthed()", func() {
		BeforeEach(func() {
			check = func() {
				ms.Check(func() (*http.Response, error) {
					return c.PutFormAuthed(ms.URL, params)
				})
			}
		})

		ItHasAuthHeader()
		ItUsesHTTPVerb("PUT")
		ItIncludesFormData()
	})
})
