package cobracmd

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	codexacpbridge "github.com/normahq/codex-acp-bridge/internal/apps/codexacpbridge"
	"github.com/normahq/codex-acp-bridge/internal/logging"
	appversion "github.com/normahq/codex-acp-bridge/internal/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestCommandUsesBridgeComponentLogger(t *testing.T) {
	origRunProxy := runProxy
	origInitLogging := initLogging
	origLogger := log.Logger
	t.Cleanup(func() {
		runProxy = origRunProxy
		initLogging = origInitLogging
		log.Logger = origLogger
	})

	initLogging = func(...logging.OptOptionsSetter) error {
		return nil
	}
	runProxy = func(ctx context.Context, _ string, _ codexacpbridge.Options, _ io.Reader, _, _ io.Writer) error {
		logging.Ctx(ctx).Info().Msg("probe")
		return nil
	}

	var logs bytes.Buffer
	log.Logger = zerolog.New(&logs).Level(zerolog.DebugLevel)

	cmd := New()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := logs.String()
	if !strings.Contains(got, `"component":"codex.acp.bridge"`) {
		t.Fatalf("logs missing bridge component: %q", got)
	}
	if strings.Contains(got, `"component":"codex.acp.proxy"`) {
		t.Fatalf("logs contain old proxy component: %q", got)
	}
}

func TestCommandExposesOnlyBridgeFlags(t *testing.T) {
	cmd := New()
	for _, removedFlag := range []string{
		"codex-sandbox",
		"codex-approval-policy",
		"codex-profile",
		"codex-base-instructions",
		"codex-developer-instructions",
		"codex-compact-prompt",
		"codex-config",
	} {
		if got := cmd.Flags().Lookup(removedFlag); got != nil {
			t.Fatalf("flag %q unexpectedly present", removedFlag)
		}
	}
	if got := cmd.Flags().Lookup("sandbox"); got != nil {
		t.Fatal("flag \"sandbox\" unexpectedly present")
	}
	for _, expectedFlag := range []string{"name", "message-streaming", "reasoning-streaming", "reasoning-thoughts", "reasoning-summary", "codex-args", "debug"} {
		if got := cmd.Flags().Lookup(expectedFlag); got == nil {
			t.Fatalf("flag %q missing", expectedFlag)
		}
	}
	if got := cmd.Commands(); len(got) == 0 {
		t.Fatal("commands = 0, want version subcommand")
	}
}

func TestCommandPassesCodexArgsToRunProxy(t *testing.T) {
	origRunProxy := runProxy
	origInitLogging := initLogging
	t.Cleanup(func() {
		runProxy = origRunProxy
		initLogging = origInitLogging
	})

	initLogging = func(...logging.OptOptionsSetter) error {
		return nil
	}

	var gotOpts codexacpbridge.Options
	runProxy = func(_ context.Context, _ string, opts codexacpbridge.Options, _ io.Reader, _, _ io.Writer) error {
		gotOpts = opts
		return nil
	}

	cmd := New()
	cmd.SetArgs([]string{"--codex-args=--sandbox=workspace-write", "--codex-args=--search"})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(gotOpts.CodexArgs) != 2 {
		t.Fatalf("CodexArgs len = %d, want 2", len(gotOpts.CodexArgs))
	}
	if gotOpts.CodexArgs[0] != "--sandbox=workspace-write" {
		t.Fatalf("CodexArgs[0] = %q, want %q", gotOpts.CodexArgs[0], "--sandbox=workspace-write")
	}
	if gotOpts.CodexArgs[1] != "--search" {
		t.Fatalf("CodexArgs[1] = %q, want %q", gotOpts.CodexArgs[1], "--search")
	}
}

func TestVersionCommandPrintsBuildVersion(t *testing.T) {
	cmd := New()
	var stdout bytes.Buffer
	cmd.SetArgs([]string{"version"})
	cmd.SetOut(&stdout)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := stdout.String(); got != appversion.String()+"\n" {
		t.Fatalf("stdout = %q, want %q", got, appversion.String()+"\n")
	}
}

func TestCommandPassesExecuteContextToRunProxy(t *testing.T) {
	origRunProxy := runProxy
	origInitLogging := initLogging
	t.Cleanup(func() {
		runProxy = origRunProxy
		initLogging = origInitLogging
	})

	type ctxKey string
	const key ctxKey = "signal-test"

	initLogging = func(...logging.OptOptionsSetter) error {
		return nil
	}

	var gotValue string
	runProxy = func(ctx context.Context, _ string, _ codexacpbridge.Options, _ io.Reader, _, _ io.Writer) error {
		gotValue, _ = ctx.Value(key).(string)
		return nil
	}

	cmd := New()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	ctx := context.WithValue(context.Background(), key, "propagated")
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if gotValue != "propagated" {
		t.Fatalf("runProxy context value = %q, want %q", gotValue, "propagated")
	}
}

func TestCommandPassesReasoningFlagsToRunProxy(t *testing.T) {
	origRunProxy := runProxy
	origInitLogging := initLogging
	t.Cleanup(func() {
		runProxy = origRunProxy
		initLogging = origInitLogging
	})

	initLogging = func(...logging.OptOptionsSetter) error {
		return nil
	}

	var gotOpts codexacpbridge.Options
	runProxy = func(_ context.Context, _ string, opts codexacpbridge.Options, _ io.Reader, _, _ io.Writer) error {
		gotOpts = opts
		return nil
	}

	cmd := New()
	cmd.SetArgs([]string{"--reasoning-thoughts=both", "--reasoning-summary=detailed", "--reasoning-streaming=false"})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotOpts.ReasoningThoughts != "both" {
		t.Fatalf("ReasoningThoughts = %q, want both", gotOpts.ReasoningThoughts)
	}
	if gotOpts.ReasoningSummary != "detailed" {
		t.Fatalf("ReasoningSummary = %q, want detailed", gotOpts.ReasoningSummary)
	}
	if gotOpts.ReasoningStreaming {
		t.Fatal("ReasoningStreaming = true, want false")
	}
}
