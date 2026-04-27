package optcli

import (
	"test-cli/internal/optimizely"
	"test-cli/internal/tui"
)

type Commander struct {
	opt    optimizely.Service
	styles tui.Styles
}
