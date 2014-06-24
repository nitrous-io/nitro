package environmenttest

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/nitrous-io/nitro/environment"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const fullConfig = `{
	"server": "https://www.example.com",
	"client_id": "client_id",
	"client_secret": "client_secret"
	}`
const configFile = "/tmp/nitro-test-config"

var _ = BeforeSuite(func() {
	ioutil.WriteFile(configFile, []byte(fullConfig), 0644)
})

var _ = AfterSuite(func() {
	os.Remove(configFile)
})

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

	Context("PopulateConfig", func() {
		BeforeEach(func() {
			env = BaseEnvironment()
		})

		It("adds valid config file settings to base", func() {
			env.ConfigFile = configFile
			env.PopulateConfig()
			Expect(env.Server).To(Equal("https://www.example.com"))
		})

		It("errs on file not existing", func() {
			env.ConfigFile = "/no/such/path"
			err := env.PopulateConfig()
			Expect(err).Should(MatchError("could not find config file at /no/such/path"))
		})
	})

	Context("ParseConfig", func() {
		It("populates config fields", func() {
			c, err := ParseConfig([]byte(fullConfig))
			Expect(err).Should(BeNil())
			Expect(c.Server).To(Equal("https://www.example.com"))
			Expect(c.ClientID).To(Equal("client_id"))
			Expect(c.ClientSecret).To(Equal("client_secret"))
		})

		It("handles empty data", func() {
			c, err := ParseConfig([]byte("{}"))
			Expect(err).Should(BeNil())
			Expect(c.Server).To(Equal(""))
		})

		It("errs on invalid data", func() {
			_, err := ParseConfig([]byte("badjson"))
			Expect(err).Should(MatchError("invalid config data: badjson"))
		})
	})
})
