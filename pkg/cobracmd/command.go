package cobracmd

import (
	canonical "github.com/baldaworks/codex-acp/pkg/cobracmd"
	"github.com/spf13/cobra"
)

// New returns the canonical command through the legacy import path.
func New() *cobra.Command {
	return canonical.New()
}

// Command retains the original public constructor.
func Command() *cobra.Command {
	return New()
}
