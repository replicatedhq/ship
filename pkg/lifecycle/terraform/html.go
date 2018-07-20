package terraform

import (
	"fmt"

	"github.com/buildkite/terminal"
)

func ansiToHTML(ansi string) string {
	html := terminal.Render([]byte(ansi))
	return fmt.Sprintf(`<div class="term-container">%s</div>`, html)
}
