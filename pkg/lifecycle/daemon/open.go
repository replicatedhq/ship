package daemon

import (
	"strings"

	"fmt"

	"github.com/mitchellh/cli"
	"github.com/skratchdot/open-golang/open"
)

type opener func(cli.Ui, string) error

func tryOpenWebConsole(ui cli.Ui, url string) error {
	openBrowser, err := ui.Ask("Open browser to continue? (Y/n)")
	if err != nil {
		return err
	}

	openBrowser = strings.ToLower(strings.Trim(openBrowser, " \r\n"))
	if strings.Compare(openBrowser, "n") == 0 {
		ui.Info(fmt.Sprintf(
			"\nPlease visit the following URL in your browser to continue the installation\n\n        %s\n\n ",
			url,
		))
	} else {
		ui.Info("\n\n       Opening console at " + url + " ...")
		err := open.Start(url)
		// debug.Log("event", "console.open.fail.ignore", "err", err)
		return err
	}
	return nil
}
