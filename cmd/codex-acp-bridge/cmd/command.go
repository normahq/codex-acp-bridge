package command

import (
	"github.com/normahq/codex-acp-bridge/pkg/cobracmd"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return cobracmd.New()
}
