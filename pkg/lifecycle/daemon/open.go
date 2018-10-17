package daemon

import (
	"fmt"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/skratchdot/open-golang/open"
)

type opener func(cli.Ui, string, bool) error

func tryOpenWebConsole(ui cli.Ui, url string, autoOpen bool) error {
	if !autoOpen {
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
			return nil
		}
	}

	ui.Info("\n       Opening console at " + url + " ...")
	return open.Start(url)
}
