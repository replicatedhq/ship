package daemon

import (
	"github.com/mitchellh/cli"
	"github.com/skratchdot/open-golang/open"
)

type opener func(cli.Ui, string) error

func tryOpenWebConsole(ui cli.Ui, url string) error {
	ui.Info("\n\n       Opening console at " + url + " ...")
	err := open.Start(url)
	return err
}
