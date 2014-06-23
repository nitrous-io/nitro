package environmenttest

import (
	"os"
	"path"

	. "github.com/nitrous-io/nitro/environment"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("environment", func() {
	var (
		env Environment
	)

	Context("BaseEnvironment", func() {
		BeforeEach(func() {
			env = BaseEnvironment()
		})

		It("uses ~/.nitro base dir", func() {
			Expect(env.BaseDir).To(Equal(path.Join(os.Getenv("HOME"), ".nitro")))
		})

		It("uses ~/.nitro/config config file", func() {
			Expect(env.ConfigFile).To(Equal(path.Join(os.Getenv("HOME"), ".nitro", "config")))
		})
	})
})
