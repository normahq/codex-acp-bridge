package codexacp

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

const bridgeAppServerSubcommand = "app-server"

type appServerSessionSpy struct {
	initializeResponse appServerInitializeResponse
	closeCalls         int
	waitCalls          int
}

func (s *appServerSessionSpy) InitializeResponse() appServerInitializeResponse {
	return s.initializeResponse
}

func (s *appServerSessionSpy) Events() <-chan appServerEvent {
	return nil
}

func (s *appServerSessionSpy) ThreadStart(context.Context, map[string]any) (appServerThreadStartResponse, error) {
	return appServerThreadStartResponse{}, nil
}

func (s *appServerSessionSpy) TurnStart(context.Context, map[string]any) (appServerTurnStartResponse, error) {
	return appServerTurnStartResponse{}, nil
}

func (s *appServerSessionSpy) ModelList(context.Context, map[string]any) (appServerModelListResponse, error) {
	return appServerModelListResponse{}, nil
}

func (s *appServerSessionSpy) TurnInterrupt(context.Context, string, string) error {
	return nil
}

func (s *appServerSessionSpy) RespondRequest(context.Context, *appServerRequest, any) error {
	return nil
}

func (s *appServerSessionSpy) RespondRequestError(context.Context, *appServerRequest, int, string, any) error {
	return nil
}

func (s *appServerSessionSpy) Close() error {
	s.closeCalls++
	return nil
}

func (s *appServerSessionSpy) Wait() error {
	s.waitCalls++
	return nil
}

func TestRunProxyRequiresIOStreams(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		stdin   io.Reader
		stdout  io.Writer
		stderr  io.Writer
		wantErr string
	}{
		{
			name:    "nil stdin",
			stdin:   nil,
			stdout:  io.Discard,
			stderr:  io.Discard,
			wantErr: "stdin is required",
		},
		{
			name:    "nil stdout",
			stdin:   strings.NewReader(""),
			stdout:  nil,
			stderr:  io.Discard,
			wantErr: "stdout is required",
		},
		{
			name:    "nil stderr",
			stdin:   strings.NewReader(""),
			stdout:  io.Discard,
			stderr:  nil,
			wantErr: "stderr is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := RunProxy(context.Background(), "/tmp", Options{}, tc.stdin, tc.stdout, tc.stderr)
			if err == nil {
				t.Fatalf("RunProxy() error = nil, want %q", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("RunProxy() error = %q, want %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestBuildCodexAppCommand(t *testing.T) {
	got := buildCodexAppCommand(Options{Name: "ignored"})
	if len(got) != 2 {
		t.Fatalf("buildCodexAppCommand() len = %d, want 2", len(got))
	}
	if got[0] != "codex" || got[1] != bridgeAppServerSubcommand {
		t.Fatalf("buildCodexAppCommand() = %#v, want [\"codex\", %q]", got, bridgeAppServerSubcommand)
	}
}

func TestValidateAppServerFactoryReturnsIdentityAndFinalizesBackend(t *testing.T) {
	t.Parallel()

	backend := &appServerSessionSpy{
		initializeResponse: appServerInitializeResponse{UserAgent: " codex_cli/1.2.3 "},
	}

	var receivedCWD string
	identity, err := validateAppServerFactory(context.Background(), func(_ context.Context, cwd string) (appServerSession, error) {
		receivedCWD = cwd
		return backend, nil
	}, "test-cwd")
	if err != nil {
		t.Fatalf("validateAppServerFactory() error = %v", err)
	}

	if receivedCWD != "test-cwd" {
		t.Fatalf("factory cwd = %q, want %q", receivedCWD, "test-cwd")
	}
	if got, want := identity.userAgent, "codex_cli/1.2.3"; got != want {
		t.Fatalf("identity.userAgent = %q, want %q", got, want)
	}
	if got, want := backend.closeCalls, 1; got != want {
		t.Fatalf("backend close calls = %d, want %d", got, want)
	}
	if got, want := backend.waitCalls, 1; got != want {
		t.Fatalf("backend wait calls = %d, want %d", got, want)
	}
}

func TestValidateAppServerFactoryPropagatesError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("factory failed")
	_, err := validateAppServerFactory(context.Background(), func(context.Context, string) (appServerSession, error) {
		return nil, wantErr
	}, "cwd")
	if !errors.Is(err, wantErr) {
		t.Fatalf("validateAppServerFactory() error = %v, want %v", err, wantErr)
	}
}

func TestSplitCommandForLog(t *testing.T) {
	cmd, args := splitCommandForLog(nil)
	if cmd != "" || args != nil {
		t.Fatalf("splitCommandForLog(nil) = (%q, %#v), want (\"\", nil)", cmd, args)
	}

	command := []string{"codex", "app-server", "--stdio"}
	cmd, args = splitCommandForLog(command)
	if cmd != "codex" {
		t.Fatalf("command name = %q, want %q", cmd, "codex")
	}
	if len(args) != 2 || args[0] != bridgeAppServerSubcommand || args[1] != "--stdio" {
		t.Fatalf("command args = %#v, want [%q, \"--stdio\"]", args, bridgeAppServerSubcommand)
	}

	command[1] = "mutated-bridge-command"
	if args[0] != bridgeAppServerSubcommand {
		t.Fatalf("args alias original slice: args[0] = %q, want %q", args[0], bridgeAppServerSubcommand)
	}
}
