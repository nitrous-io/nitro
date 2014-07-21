package command

import (
	"strings"

	"github.com/mitchellh/cli"
)

type UI interface {
	cli.Ui
	Confirm(query string) Confirmation
}

type BasicUI struct {
	cli.BasicUi
}

type Confirmation int

const (
	ConfirmYes Confirmation = iota + 1
	ConfirmNo
	ConfirmCancel
	ConfirmErr
)

func (u *BasicUI) Confirm(query string) Confirmation {
	answer, err := u.BasicUi.Ask(query)
	if err != nil {
		if err.Error() == "interrupted" {
			return ConfirmCancel
		}
		return ConfirmErr
	}

	if strings.ToLower(answer) == "y" {
		return ConfirmYes
	}

	return ConfirmNo
}
