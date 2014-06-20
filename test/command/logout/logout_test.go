package logouttest

import (
	"io/ioutil"

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
	)

	BeforeEach(func() {
		ui = &cli.MockUi{}
		cmd = &LogoutCommand{UI: ui, TokenFilePath: "/tmp/.nitrotoken"}
	})

	It("implements cli.Command", func() {
		var _ cli.Command = cmd
	})

	Context("when token file exists", func() {
		BeforeEach(func() {
			ioutil.WriteFile("/tmp/.nitrotoken", []byte("accesstoken"), 0644)
		})

		It("removes the file and exits successfully", func() {
			exitCode := cmd.Run(args)
			_, err := ioutil.ReadFile("/tmp/.nitrotoken")
			Expect(err).To(MatchError("open /tmp/.nitrotoken: no such file or directory"))
			Expect(exitCode).To(Equal(0))
		})
	})

	Context("when token file does not exist", func() {
		It("prints a message and exits with an error code", func() {
			exitCode := cmd.Run(args)
			Expect(exitCode).To(Equal(1))
		})
	})
})
