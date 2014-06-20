package logouttest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLogout(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logout")
}
