package daemon

import (
	"github.com/mitchellh/cli"
	"github.com/skratchdot/open-golang/open"
)

type opener func(cli.Ui, string) error

func tryOpenWebConsole(ui cli.Ui, url string) error {
	//ui.Info("Opening console...")
	// ignore error
	err := open.Start(url)
	//ui.Info(fmt.Sprintf(
	//	"Please visit the following URL in your browser to continue the installation\n\n        %s\n\n ",
	//	url,
	//))
	return err
}
