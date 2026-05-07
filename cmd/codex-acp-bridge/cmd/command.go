package command

import (
	"fmt"
	"os"
	"strings"

	codexacpbridge "github.com/normahq/codex-acp-bridge/internal/apps/codexacpbridge"
	"github.com/normahq/codex-acp-bridge/internal/logging"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	runProxy    = codexacpbridge.RunProxy
	initLogging = logging.Init
)

const bridgeDefaultAgentName = "norma-codex-acp-bridge"

func Command() *cobra.Command {
	opts := codexacpbridge.Options{}
	var debugLogs bool
	reasoningStreaming := true
	reasoningThoughts := "summary"

	cmd := &cobra.Command{
		Use:          "codex-acp-bridge [flags]",
		Short:        "Expose Codex bridge backend as ACP over stdio",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			runOpts := opts
			runOpts.SetReasoningStreaming(reasoningStreaming)
			runOpts.ReasoningThoughts = reasoningThoughts
			if strings.TrimSpace(runOpts.Name) == "" {
				runOpts.Name = bridgeDefaultAgentName
			}

			logLevel := logging.LevelInfo
			if debugLogs {
				logLevel = logging.LevelDebug
			}
			if err := initLogging(logging.WithLevel(logLevel)); err != nil {
				return fmt.Errorf("initialize logging: %w", err)
			}
			ctx := log.Logger.With().Str("component", "codex.acp.bridge").Logger().WithContext(cmd.Context())

			return runProxy(ctx, workingDir, runOpts, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "ACP agent name exposed via initialize (defaults to norma-codex-acp-bridge)")
	cmd.Flags().BoolVar(&opts.MessageStreaming, "message-streaming", false, "stream app-server agentMessage deltas as ACP agent_message_chunk updates")
	cmd.Flags().BoolVar(&reasoningStreaming, "reasoning-streaming", true, "stream app-server reasoning deltas as ACP agent_thought_chunk updates")
	cmd.Flags().StringVar(&reasoningThoughts, "reasoning-thoughts", reasoningThoughts, "reasoning thought lane to project: off, summary, content, or both")
	cmd.Flags().BoolVar(&debugLogs, "debug", false, "enable debug logging")
	cmd.Long = "Run the Codex bridge backend and expose it as an ACP agent over stdio. Configure per-session Codex behavior using ACP session/new _meta.codex."
	//nolint:dupword
	cmd.Example = `  codex-acp-bridge
  codex-acp-bridge --name team-codex
  codex-acp-bridge --message-streaming
  codex-acp-bridge --reasoning-thoughts=both
  codex-acp-bridge --reasoning-streaming=false
  codex-acp-bridge --debug`
	return cmd
}
