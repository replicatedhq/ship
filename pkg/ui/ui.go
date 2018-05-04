package ui

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/spf13/viper"
)

func FromViper(v *viper.Viper) cli.Ui {
	base := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if !isInteractive() && !v.GetBool("force-color") {
		return base
	}

	if v.GetBool("no-color") {
		return base
	}

	return &cli.ColoredUi{
		OutputColor: cli.UiColorNone,
		ErrorColor:  cli.UiColorRed,
		WarnColor:   cli.UiColorYellow,
		InfoColor:   cli.UiColorGreen,
		Ui:          base,
	}
}

// todo detect if this is an interactive session and/or if we have a tty
func isInteractive() bool {
	return true
}
