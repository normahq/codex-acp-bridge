package cobracmd_test

import (
	"io"
	"testing"

	canonical "github.com/baldaworks/codex-acp/pkg/cobracmd"
	legacy "github.com/normahq/codex-acp-bridge/pkg/cobracmd"
	"github.com/spf13/cobra"
)

func TestLegacyPublicAPI(t *testing.T) {
	for _, factory := range []func() *cobra.Command{legacy.New, legacy.Command} {
		cmd := factory()
		want := canonical.New()
		if cmd.Short != want.Short {
			t.Fatalf("legacy description = %q, want %q", cmd.Short, want.Short)
		}
		for _, flag := range []string{"name", "defer-backend", "sandbox", "codex-args", "mcp-approval-policy", "reasoning-streaming", "message-streaming"} {
			if cmd.Flags().Lookup(flag) == nil {
				t.Fatalf("missing legacy flag %q", flag)
			}
		}
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"--help"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("legacy help: %v", err)
		}
	}
}
