package logouttest

import (
	"io/ioutil"
	"os"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/mitchellh/cli"
	. "github.com/nitrous-io/nitro/command/logout"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogoutCommand", func() {
	var (
		cmd  *LogoutCommand
		ui   *cli.MockUi
		args []string

		currentNetrcPath string
		testNetrcPath    string
	)

	BeforeEach(func() {
		ui = &cli.MockUi{}
		cmd = &LogoutCommand{UI: ui, Server: "www.example.com"}

		testNetrcPath = "/tmp/netrc"
		currentNetrcPath = os.Getenv("NETRC_PATH")
		os.Setenv("NETRC_PATH", testNetrcPath)
	})

	AfterEach(func() {
		os.Setenv("NETRC_PATH", currentNetrcPath)
	})

	It("implements cli.Command", func() {
		var _ cli.Command = cmd
	})

	Context("when token exist in Netrc file", func() {
		BeforeEach(func() {
			nrc := &netrc.Netrc{}
			nrc.NewMachine("www.example.com", "user", "token", "")

			body, err := nrc.MarshalText()
			if err != nil {
				panic(err)
			}

			ioutil.WriteFile(testNetrcPath, body, 0600)
		})

		AfterEach(func() {
			err := os.Remove(testNetrcPath)
			if err != nil {
				panic(err)
			}
		})

		It("removes the token", func() {
			exitCode := cmd.Run(args)
			Expect(exitCode).To(Equal(0))

			machine, _ := netrc.FindMachine(testNetrcPath, "www.example.com")
			Expect(machine).To(BeNil())
		})
	})
})
