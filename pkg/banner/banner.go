// Package banner prints a startup banner similar to Spring Boot.
package banner

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/kirklin/boot-backend-go-clean/pkg/version"
)

const logo = `
    ____              __     ____             __                  __
   / __ )____  ____  / /_   / __ )____ ______/ /_____  ____  ____/ /
  / __  / __ \/ __ \/ __/  / __  / __ '/ ___/ //_/ _ \/ __ \/ __  / 
 / /_/ / /_/ / /_/ / /_   / /_/ / /_/ / /__/ ,< /  __/ / / / /_/ /  
/_____/\____/\____/\__/  /_____/\__,_/\___/_/|_|\___/_/ /_/\__,_/   
`

// Print writes the startup banner to the given writer.
// In production, ANSI colors are stripped to keep log aggregators clean.
func Print(w io.Writer, environment string) {
	isProd := environment == "production"
	var b strings.Builder

	b.WriteString(colorize("\033[36m", logo, isProd))

	fmt.Fprintf(&b, "  %s    %s\n",
		colorize("\033[90m", ":: Boot Backend Go Clean ::", isProd),
		colorize("\033[32m", fmt.Sprintf("(v%s)", version.Version), isProd))
	b.WriteString("\n")

	rows := []struct{ label, value string }{
		{"Author", fmt.Sprintf("%s %s", applyHighlight("Kirk Lin", isProd), colorize("\033[90m", "— https://github.com/kirklin", isProd))},
		{"Environment", applyHighlight(environment, isProd)},
		{"Go Version", applyHighlight(runtime.Version(), isProd)},
		{"OS/Arch", applyHighlight(runtime.GOOS+"/"+runtime.GOARCH, isProd)},
		{"PID", applyHighlight(fmt.Sprintf("%d", os.Getpid()), isProd)},
		{"Git Commit", applyHighlight(version.GitCommit, isProd)},
		{"Build Time", applyHighlight(version.BuildTime, isProd)},
	}

	for _, r := range rows {
		fmt.Fprintf(&b, "  %-14s %s\n", r.label+":", r.value)
	}
	b.WriteString("\n")

	_, _ = fmt.Fprint(w, b.String())
}

// colorize wraps text with an ANSI color code. In production mode, returns plain text.
func colorize(code, text string, plain bool) string {
	if plain {
		return text
	}
	return code + text + "\033[0m"
}

// applyHighlight applies yellow highlight; plain text in production.
func applyHighlight(s string, plain bool) string {
	return colorize("\033[33m", s, plain)
}
