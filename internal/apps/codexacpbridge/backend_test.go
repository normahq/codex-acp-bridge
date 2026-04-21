package codexacp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

const (
	errRequestIDRequired = "request id is required"
	errBackendStopped    = "bridge backend stopped"
)

type captureWriteCloser struct {
	mu       sync.Mutex
	writes   [][]byte
	writeErr error
	closeErr error
}

func (c *captureWriteCloser) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	cp := append([]byte(nil), p...)
	c.writes = append(c.writes, cp)
	return len(p), nil
}

func (c *captureWriteCloser) Close() error {
	return c.closeErr
}

func (c *captureWriteCloser) writesSnapshot() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.writes))
	for i := range c.writes {
		out[i] = append([]byte(nil), c.writes[i]...)
	}
	return out
}

func newTestBackend(writer io.WriteCloser) *appServerBackend {
	logger := testNopLogger()
	return &appServerBackend{
		stdin:    writer,
		logger:   logger,
		pending:  make(map[string]chan appServerRPCResponse),
		events:   make(chan appServerEvent, 16),
		done:     make(chan struct{}),
		readDone: make(chan struct{}),
	}
}

func testNopLogger() *zerolog.Logger {
	nop := zerolog.Nop()
	return &nop
}

func parseFirstWriteAsJSONMap(t *testing.T, writer *captureWriteCloser) map[string]any {
	t.Helper()
	writes := writer.writesSnapshot()
	if len(writes) == 0 {
		t.Fatal("writer captured no payload")
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(writes[0]), &payload); err != nil {
		t.Fatalf("unmarshal payload error = %v", err)
	}
	return payload
}

func waitForPendingRequest(t *testing.T, b *appServerBackend, key string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		b.pendingMu.Lock()
		_, ok := b.pending[key]
		b.pendingMu.Unlock()
		if ok {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("pending request %q not found before timeout", key)
}

func TestAppServerRPCErrorError(t *testing.T) {
	var nilErr *appServerRPCError
	if got := nilErr.Error(); got != "" {
		t.Fatalf("(*appServerRPCError)(nil).Error() = %q, want empty", got)
	}

	errValue := &appServerRPCError{Code: -32000, Message: "boom"}
	if got, want := errValue.Error(), "bridge backend rpc error (-32000): boom"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestCanonicalRequestID(t *testing.T) {
	if got, want := canonicalRequestID(json.RawMessage("  123 ")), "123"; got != want {
		t.Fatalf("canonicalRequestID() = %q, want %q", got, want)
	}
}

func TestSendJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		ctx       context.Context
		closeDone bool
		payload   any
		writeErr  error
		wantErr   string
	}{
		{
			name:    "marshal error",
			ctx:     context.Background(),
			payload: map[string]any{"bad": make(chan int)},
			wantErr: "marshal bridge backend payload",
		},
		{
			name:    "context canceled",
			ctx:     canceledContext(),
			payload: map[string]any{"ok": true},
			wantErr: context.Canceled.Error(),
		},
		{
			name:      "backend stopped",
			ctx:       context.Background(),
			closeDone: true,
			payload:   map[string]any{"ok": true},
			wantErr:   errBackendStopped,
		},
		{
			name:     "write error",
			ctx:      context.Background(),
			payload:  map[string]any{"ok": true},
			writeErr: errors.New("sink failed"),
			wantErr:  "write bridge backend payload",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &captureWriteCloser{writeErr: tc.writeErr}
			backend := newTestBackend(writer)
			if tc.closeDone {
				close(backend.done)
			}

			err := backend.sendJSON(tc.ctx, tc.payload)
			if err == nil {
				t.Fatalf("sendJSON() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("sendJSON() error = %q, want contains %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestSendRequestResponseErrorAndNotification(t *testing.T) {
	writer := &captureWriteCloser{}
	backend := newTestBackend(writer)

	if err := backend.sendRequest(context.Background(), json.RawMessage("1"), "model/list", map[string]any{"a": "b"}); err != nil {
		t.Fatalf("sendRequest() error = %v", err)
	}
	payload := parseFirstWriteAsJSONMap(t, writer)
	if got, want := payload["method"], "model/list"; got != want {
		t.Fatalf("sendRequest method = %#v, want %q", got, want)
	}
	if _, ok := payload["params"]; !ok {
		t.Fatal("sendRequest params missing")
	}

	writer = &captureWriteCloser{}
	backend = newTestBackend(writer)
	if err := backend.sendResponse(context.Background(), json.RawMessage("2"), map[string]any{"ok": true}); err != nil {
		t.Fatalf("sendResponse() error = %v", err)
	}
	payload = parseFirstWriteAsJSONMap(t, writer)
	if _, ok := payload["result"]; !ok {
		t.Fatal("sendResponse result missing")
	}

	writer = &captureWriteCloser{}
	backend = newTestBackend(writer)
	if err := backend.sendError(context.Background(), json.RawMessage("3"), -32601, "bad", map[string]any{"m": "x"}); err != nil {
		t.Fatalf("sendError() error = %v", err)
	}
	payload = parseFirstWriteAsJSONMap(t, writer)
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("sendError payload error type = %T, want map[string]any", payload["error"])
	}
	if got, want := errObj["message"], "bad"; got != want {
		t.Fatalf("sendError message = %#v, want %q", got, want)
	}

	writer = &captureWriteCloser{}
	backend = newTestBackend(writer)
	if err := backend.sendNotification(context.Background(), "initialized", nil); err != nil {
		t.Fatalf("sendNotification() error = %v", err)
	}
	payload = parseFirstWriteAsJSONMap(t, writer)
	if got, want := payload["method"], "initialized"; got != want {
		t.Fatalf("sendNotification method = %#v, want %q", got, want)
	}
}

func TestCallSuccessAndDecodePaths(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)

		type response struct {
			Value string `json:"value"`
		}
		var out response
		errCh := make(chan error, 1)
		go func() {
			errCh <- backend.call(context.Background(), "x/method", map[string]any{"p": "v"}, &out)
		}()

		waitForPendingRequest(t, backend, "1")
		if err := backend.handleIncomingLine([]byte(`{"id":1,"result":{"value":"ok"}}` + "\n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("call() error = %v", err)
		}
		if got, want := out.Value, "ok"; got != want {
			t.Fatalf("decoded value = %q, want %q", got, want)
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)
		errCh := make(chan error, 1)
		go func() {
			errCh <- backend.call(context.Background(), "x/method", nil, nil)
		}()

		waitForPendingRequest(t, backend, "1")
		if err := backend.handleIncomingLine([]byte(`{"id":1,"error":{"code":-32000,"message":"nope"}}` + "\n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		err := <-errCh
		if err == nil {
			t.Fatal("call() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "bridge backend rpc error") {
			t.Fatalf("call() error = %q, want rpc error", err.Error())
		}
	})

	t.Run("decode error", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)
		var out struct {
			Value string `json:"value"`
		}
		errCh := make(chan error, 1)
		go func() {
			errCh <- backend.call(context.Background(), "x/method", nil, &out)
		}()

		waitForPendingRequest(t, backend, "1")
		if err := backend.handleIncomingLine([]byte(`{"id":1,"result":["bad"]}` + "\n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		err := <-errCh
		if err == nil {
			t.Fatal("call() error = nil, want decode error")
		}
		if !strings.Contains(err.Error(), "decode x/method response") {
			t.Fatalf("call() error = %q, want decode context", err.Error())
		}
	})
}

func TestCallContextAndDonePaths(t *testing.T) {
	t.Run("context canceled after send", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)
		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			errCh <- backend.call(ctx, "x/method", map[string]any{"p": "v"}, nil)
		}()
		waitForPendingRequest(t, backend, "1")
		cancel()
		err := <-errCh
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("call() error = %v, want context.Canceled", err)
		}
	})

	t.Run("backend stopped", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)
		errCh := make(chan error, 1)
		go func() {
			errCh <- backend.call(context.Background(), "x/method", nil, nil)
		}()
		waitForPendingRequest(t, backend, "1")
		close(backend.done)
		err := <-errCh
		if err == nil || err.Error() != errBackendStopped {
			t.Fatalf("call() error = %v, want %q", err, errBackendStopped)
		}
	})
}

func TestHandleIncomingLine(t *testing.T) {
	t.Run("blank line", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		if err := backend.handleIncomingLine([]byte(" \n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		if err := backend.handleIncomingLine([]byte("{oops\n")); err == nil {
			t.Fatal("handleIncomingLine() error = nil, want non-nil")
		}
	})

	t.Run("request event", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		err := backend.handleIncomingLine([]byte(`{"id":1,"method":"ask","params":{"k":"v"}}` + "\n"))
		if err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		ev := <-backend.events
		if ev.Request == nil || ev.Request.Method != "ask" {
			t.Fatalf("request event = %#v, want method ask", ev.Request)
		}
	})

	t.Run("notification event", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		err := backend.handleIncomingLine([]byte(`{"method":"note","params":{"x":1}}` + "\n"))
		if err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		ev := <-backend.events
		if ev.Notification == nil || ev.Notification.Method != "note" {
			t.Fatalf("notification event = %#v, want method note", ev.Notification)
		}
	})

	t.Run("response without pending request", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		if err := backend.handleIncomingLine([]byte(`{"id":1,"result":{"ok":true}}` + "\n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
	})

	t.Run("drops request event after backend stop", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		close(backend.done)
		if err := backend.handleIncomingLine([]byte(`{"id":1,"method":"ask","params":{"k":"v"}}` + "\n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		select {
		case ev := <-backend.events:
			t.Fatalf("unexpected event after backend stop: %#v", ev)
		default:
		}
	})
}

func TestTurnInterrupt(t *testing.T) {
	t.Run("empty ids is no-op", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)
		if err := backend.TurnInterrupt(context.Background(), " ", "\t"); err != nil {
			t.Fatalf("TurnInterrupt() error = %v", err)
		}
		if got := len(writer.writesSnapshot()); got != 0 {
			t.Fatalf("writes = %d, want 0", got)
		}
	})

	t.Run("sends request when ids present", func(t *testing.T) {
		writer := &captureWriteCloser{}
		backend := newTestBackend(writer)
		errCh := make(chan error, 1)
		go func() {
			errCh <- backend.TurnInterrupt(context.Background(), "thr-1", "turn-1")
		}()
		waitForPendingRequest(t, backend, "1")
		if err := backend.handleIncomingLine([]byte(`{"id":1,"result":null}` + "\n")); err != nil {
			t.Fatalf("handleIncomingLine() error = %v", err)
		}
		if err := <-errCh; err != nil {
			t.Fatalf("TurnInterrupt() error = %v", err)
		}
		payload := parseFirstWriteAsJSONMap(t, writer)
		if got, want := payload["method"], "turn/interrupt"; got != want {
			t.Fatalf("method = %#v, want %q", got, want)
		}
	})
}

func TestRespondRequestAndErrorValidation(t *testing.T) {
	backend := newTestBackend(&captureWriteCloser{})
	if err := backend.RespondRequest(context.Background(), nil, nil); err == nil || err.Error() != errRequestIDRequired {
		t.Fatalf("RespondRequest(nil) error = %v, want %q", err, errRequestIDRequired)
	}
	if err := backend.RespondRequestError(context.Background(), nil, -1, "bad", nil); err == nil || err.Error() != errRequestIDRequired {
		t.Fatalf("RespondRequestError(nil) error = %v, want %q", err, errRequestIDRequired)
	}
}

func TestFailPending(t *testing.T) {
	t.Run("generic stop error", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		ch := make(chan appServerRPCResponse, 1)
		backend.pending["1"] = ch
		backend.failPending(nil)
		resp := <-ch
		if resp.err == nil || resp.err.Error() != errBackendStopped {
			t.Fatalf("failPending(nil) err = %v, want %q", resp.err, errBackendStopped)
		}
	})

	t.Run("exit error wraps wait error", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		ch := make(chan appServerRPCResponse, 1)
		backend.pending["1"] = ch
		backend.failPending(errors.New("boom"))
		resp := <-ch
		if resp.err == nil || !strings.Contains(resp.err.Error(), "bridge backend exited: boom") {
			t.Fatalf("failPending(err) = %v, want wrapped error", resp.err)
		}
	})

	t.Run("process done uses generic stop error", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		ch := make(chan appServerRPCResponse, 1)
		backend.pending["1"] = ch
		backend.failPending(os.ErrProcessDone)
		resp := <-ch
		if resp.err == nil || resp.err.Error() != errBackendStopped {
			t.Fatalf("failPending(os.ErrProcessDone) err = %v, want %q", resp.err, errBackendStopped)
		}
	})
}

func TestWait(t *testing.T) {
	t.Run("nil wait err", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		close(backend.done)
		if err := backend.Wait(); err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
	})

	t.Run("process done wait err", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		backend.waitErr = os.ErrProcessDone
		close(backend.done)
		if err := backend.Wait(); err != nil {
			t.Fatalf("Wait() error = %v", err)
		}
	})

	t.Run("returns wait err", func(t *testing.T) {
		backend := newTestBackend(&captureWriteCloser{})
		wantErr := errors.New("wait failed")
		backend.waitErr = wantErr
		close(backend.done)
		if err := backend.Wait(); !errors.Is(err, wantErr) {
			t.Fatalf("Wait() error = %v, want %v", err, wantErr)
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("close error returned", func(t *testing.T) {
		writer := &captureWriteCloser{closeErr: errors.New("close failed")}
		backend := newTestBackend(writer)
		backend.cmd = &exec.Cmd{}
		if err := backend.Close(); err == nil || err.Error() != "close failed" {
			t.Fatalf("Close() error = %v, want %q", err, "close failed")
		}
	})

	t.Run("os.ErrClosed ignored", func(t *testing.T) {
		writer := &captureWriteCloser{closeErr: os.ErrClosed}
		backend := newTestBackend(writer)
		backend.cmd = &exec.Cmd{}
		if err := backend.Close(); err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
	})
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
