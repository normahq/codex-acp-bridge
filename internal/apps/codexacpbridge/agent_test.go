package codexacp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/rs/zerolog"
)

const testModelGPT54 = "gpt-5.4"

const (
	testApprovalOnRequest      = "on-request"
	testApprovalsReviewerGuard = "guardian_subagent"
	testMCPTransportStdio      = "stdio"
	testPersonalityPragmatic   = "pragmatic"
	testReasoningXHigh         = "xhigh"
	testServiceTierFlex        = "flex"
)

func TestBuildThreadStartParamsIncludesConfigAndMCPServers(t *testing.T) {
	ephemeral := true
	params := buildThreadStartParams(
		"/tmp/work",
		codexAppConfig{
			ApprovalPolicy:    testApprovalOnRequest,
			ApprovalsReviewer: testApprovalsReviewerGuard,
			BaseInstructions:  "base",
			CompactPrompt:     "compact",
			Config: map[string]any{
				"foo": "bar",
			},
			DeveloperInstructions: "dev",
			Ephemeral:             &ephemeral,
			Model:                 testModelGPT54,
			ModelProvider:         "openai",
			Personality:           testPersonalityPragmatic,
			Profile:               "dev-profile",
			Sandbox:               "workspace-write",
			ServiceTier:           testServiceTierFlex,
		},
		"",
		map[string]acp.McpServer{
			"docs": {
				Stdio: &acp.McpServerStdio{
					Name:    "docs",
					Command: "docs-server",
					Args:    []string{"--listen"},
				},
			},
		},
	)

	if got := stringValue(params, "cwd"); got != "/tmp/work" {
		t.Fatalf("cwd = %q, want %q", got, "/tmp/work")
	}
	if got := stringValue(params, "model"); got != testModelGPT54 {
		t.Fatalf("model = %q, want %q", got, testModelGPT54)
	}
	if got := stringValue(params, "approvalPolicy"); got != testApprovalOnRequest {
		t.Fatalf("approvalPolicy = %q, want %q", got, testApprovalOnRequest)
	}
	if got := stringValue(params, "approvalsReviewer"); got != testApprovalsReviewerGuard {
		t.Fatalf("approvalsReviewer = %q, want %q", got, testApprovalsReviewerGuard)
	}
	if got := stringValue(params, "modelProvider"); got != "openai" {
		t.Fatalf("modelProvider = %q, want %q", got, "openai")
	}
	if got := stringValue(params, "personality"); got != testPersonalityPragmatic {
		t.Fatalf("personality = %q, want %q", got, testPersonalityPragmatic)
	}
	if got := stringValue(params, "serviceTier"); got != testServiceTierFlex {
		t.Fatalf("serviceTier = %q, want %q", got, testServiceTierFlex)
	}
	if got, ok := boolValue(params, "ephemeral"); !ok || !got {
		t.Fatalf("ephemeral = %t (ok=%t), want true", got, ok)
	}
	config := mapValue(params, "config")
	if got := stringValue(config, "profile"); got != "dev-profile" {
		t.Fatalf("config.profile = %q, want %q", got, "dev-profile")
	}
	if got := stringValue(config, "compact_prompt"); got != "compact" {
		t.Fatalf("config.compact_prompt = %q, want %q", got, "compact")
	}
	if _, ok := config["mcp_servers"]; !ok {
		t.Fatalf("config.mcp_servers missing")
	}
}

func TestNewSessionAppliesCodexMetaOverridesToThreadStart(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	l := zerolog.Nop()
	defaultEphemeral := false
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{
		Sandbox:               "read-only",
		ApprovalPolicy:        "never",
		ApprovalsReviewer:     "user",
		BaseInstructions:      "default-base",
		DeveloperInstructions: "default-dev",
		Profile:               "default-profile",
		CompactPrompt:         "default-compact",
		ModelProvider:         "default-provider",
		Personality:           "friendly",
		ServiceTier:           "fast",
		Ephemeral:             &defaultEphemeral,
		Config: map[string]any{
			"shared":         "default",
			"profile":        "default-config-profile",
			"compact_prompt": "default-config-compact",
			"mcp_servers":    "default-config-mcp",
		},
	}, &l)

	resp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: "/tmp/work",
		Meta: map[string]any{
			"sessionId": "session-meta-1",
			"codex": map[string]any{
				"sandbox":               "workspace-write",
				"approvalPolicy":        testApprovalOnRequest,
				"approvalsReviewer":     testApprovalsReviewerGuard,
				"baseInstructions":      "meta-base",
				"developerInstructions": "meta-dev",
				"profile":               "meta-profile",
				"compactPrompt":         "meta-compact",
				"modelProvider":         "meta-provider",
				"personality":           testPersonalityPragmatic,
				"serviceTier":           testServiceTierFlex,
				"ephemeral":             true,
				"config": map[string]any{
					"shared":                 "meta",
					"x":                      "y",
					"profile":                "from-meta-config-profile",
					"compact_prompt":         "from-meta-config-compact",
					"mcp_servers":            "from-meta-config-mcp",
					"model_reasoning_effort": testReasoningXHigh,
				},
			},
		},
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "docs",
					Command: "docs-server",
					Args:    []string{"--listen"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if got, want := string(resp.SessionId), "session-meta-1"; got != want {
		t.Fatalf("NewSession().SessionId = %q, want %q", got, want)
	}
	meta := resp.Meta
	codexMeta := mapValue(meta, "codex")
	mcpMeta := mapValue(codexMeta, "mcp")
	if got := stringValue(mcpMeta, "contract"); got != mcpContractMerge {
		t.Fatalf("NewSession()._meta.codex.mcp.contract = %q, want %q", got, mcpContractMerge)
	}
	requested := listValue(mcpMeta, "requested")
	if len(requested) != 1 {
		t.Fatalf("NewSession()._meta.codex.mcp.requested len = %d, want 1", len(requested))
	}
	requestedServer, ok := requested[0].(map[string]any)
	if !ok {
		t.Fatalf("NewSession()._meta.codex.mcp.requested[0] type = %T, want map[string]any", requested[0])
	}
	if got := stringValue(requestedServer, "name"); got != "docs" {
		t.Fatalf("NewSession()._meta.codex.mcp.requested[0].name = %q, want %q", got, "docs")
	}
	if got := stringValue(requestedServer, "transport"); got != testMCPTransportStdio {
		t.Fatalf("NewSession()._meta.codex.mcp.requested[0].transport = %q, want %q", got, testMCPTransportStdio)
	}

	threadStartParams := session.threadStartParamsSnapshot()
	if len(threadStartParams) != 1 {
		t.Fatalf("thread/start calls = %d, want 1", len(threadStartParams))
	}
	params := threadStartParams[0]
	if got := stringValue(params, "sandbox"); got != "workspace-write" {
		t.Fatalf("sandbox = %q, want %q", got, "workspace-write")
	}
	if got := stringValue(params, "approvalPolicy"); got != testApprovalOnRequest {
		t.Fatalf("approvalPolicy = %q, want %q", got, testApprovalOnRequest)
	}
	if got := stringValue(params, "approvalsReviewer"); got != testApprovalsReviewerGuard {
		t.Fatalf("approvalsReviewer = %q, want %q", got, testApprovalsReviewerGuard)
	}
	if got := stringValue(params, "baseInstructions"); got != "meta-base" {
		t.Fatalf("baseInstructions = %q, want %q", got, "meta-base")
	}
	if got := stringValue(params, "developerInstructions"); got != "meta-dev" {
		t.Fatalf("developerInstructions = %q, want %q", got, "meta-dev")
	}
	if got := stringValue(params, "modelProvider"); got != "meta-provider" {
		t.Fatalf("modelProvider = %q, want %q", got, "meta-provider")
	}
	if got := stringValue(params, "personality"); got != testPersonalityPragmatic {
		t.Fatalf("personality = %q, want %q", got, testPersonalityPragmatic)
	}
	if got := stringValue(params, "serviceTier"); got != testServiceTierFlex {
		t.Fatalf("serviceTier = %q, want %q", got, testServiceTierFlex)
	}
	if got, ok := boolValue(params, "ephemeral"); !ok || !got {
		t.Fatalf("ephemeral = %t (ok=%t), want true", got, ok)
	}

	config := mapValue(params, "config")
	if got := stringValue(config, "profile"); got != "meta-profile" {
		t.Fatalf("config.profile = %q, want %q", got, "meta-profile")
	}
	if got := stringValue(config, "compact_prompt"); got != "meta-compact" {
		t.Fatalf("config.compact_prompt = %q, want %q", got, "meta-compact")
	}
	if got := stringValue(config, "shared"); got != "meta" {
		t.Fatalf("config.shared = %q, want %q", got, "meta")
	}
	if got := stringValue(config, "x"); got != "y" {
		t.Fatalf("config.x = %q, want %q", got, "y")
	}
	if got := stringValue(config, "model_reasoning_effort"); got != testReasoningXHigh {
		t.Fatalf("config.model_reasoning_effort = %q, want xhigh", got)
	}

	mcpServersCfg := mapValue(config, "mcp_servers")
	docsCfg := mapValue(mcpServersCfg, "docs")
	if got := stringValue(docsCfg, "command"); got != "docs-server" {
		t.Fatalf("config.mcp_servers.docs.command = %q, want %q", got, "docs-server")
	}
}

func TestNewSessionRejectsUnsupportedCodexMetaKey(t *testing.T) {
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1"), nil
	}, "agent", codexAppConfig{}, &l)

	_, err := agent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: "/tmp/work",
		Meta: map[string]any{
			"codex": map[string]any{
				"unsupported": true,
			},
		},
	})
	if err == nil {
		t.Fatal("NewSession() error = nil, want non-nil")
	}
	reqErr := &acp.RequestError{}
	if !errors.As(err, &reqErr) {
		t.Fatalf("NewSession() error type = %T, want *acp.RequestError", err)
	}
	if reqErr.Code != -32602 {
		t.Fatalf("NewSession() request error code = %d, want -32602", reqErr.Code)
	}
}

func TestInitializeAdvertisesImagePromptCapability(t *testing.T) {
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return nil, errors.New("not used")
	}, "agent", codexAppConfig{}, &l)

	resp, err := agent.Initialize(context.Background(), acp.InitializeRequest{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if !resp.AgentCapabilities.PromptCapabilities.Image {
		t.Fatal("prompt image capability = false, want true")
	}
	if resp.AgentCapabilities.PromptCapabilities.Audio {
		t.Fatal("prompt audio capability = true, want false")
	}
}

func TestSessionModeIsStoredButNotForwardedToBackendPayloads(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{Model: testModelGPT54}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	if _, err := agent.SetSessionMode(context.Background(), acp.SetSessionModeRequest{
		SessionId: newResp.SessionId,
		ModeId:    acp.SessionModeId("workspace-write"),
	}); err != nil {
		t.Fatalf("SetSessionMode() error = %v", err)
	}

	if _, err := agent.UnstableSetSessionModel(context.Background(), acp.UnstableSetSessionModelRequest{
		SessionId: newResp.SessionId,
		ModelId:   acp.UnstableModelId("gpt-5.5"),
	}); err != nil {
		t.Fatalf("UnstableSetSessionModel() error = %v", err)
	}

	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	threadStartParams := session.threadStartParamsSnapshot()
	if len(threadStartParams) != 1 {
		t.Fatalf("thread/start calls = %d, want 1", len(threadStartParams))
	}
	if got := stringValue(threadStartParams[0], "cwd"); got != "/tmp/work" {
		t.Fatalf("thread/start cwd = %q, want %q", got, "/tmp/work")
	}
	if _, ok := threadStartParams[0]["mode"]; ok {
		t.Fatalf("thread/start params unexpectedly include mode: %#v", threadStartParams[0])
	}

	turnStartParams := session.turnStartParamsSnapshot()
	if len(turnStartParams) != 1 {
		t.Fatalf("turn/start calls = %d, want 1", len(turnStartParams))
	}
	if got := stringValue(turnStartParams[0], "model"); got != "gpt-5.5" {
		t.Fatalf("turn/start model = %q, want %q", got, "gpt-5.5")
	}
	if _, ok := turnStartParams[0]["mode"]; ok {
		t.Fatalf("turn/start params unexpectedly include mode: %#v", turnStartParams[0])
	}
}

func TestPromptForwardsTextAndImageBlocksToTurnStart(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	imageURI := "https://example.com/image.png"
	promptResp, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("hello"),
			{Image: &acp.ContentBlockImage{Uri: &imageURI}},
		},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("StopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	turnStartParams := session.turnStartParamsSnapshot()
	if len(turnStartParams) != 1 {
		t.Fatalf("turn/start calls = %d, want 1", len(turnStartParams))
	}
	inputItems := listValue(turnStartParams[0], "input")
	if len(inputItems) != 2 {
		t.Fatalf("turn/start input items = %d, want 2", len(inputItems))
	}
	first, ok := inputItems[0].(map[string]any)
	if !ok {
		t.Fatalf("turn/start input[0] type = %T, want map[string]any", inputItems[0])
	}
	if got := stringValue(first, "type"); got != "text" {
		t.Fatalf("turn/start input[0].type = %q, want text", got)
	}
	if got := stringValue(first, "text"); got != "hello" {
		t.Fatalf("turn/start input[0].text = %q, want hello", got)
	}
	second, ok := inputItems[1].(map[string]any)
	if !ok {
		t.Fatalf("turn/start input[1] type = %T, want map[string]any", inputItems[1])
	}
	if got := stringValue(second, "type"); got != "image" {
		t.Fatalf("turn/start input[1].type = %q, want image", got)
	}
	if got := stringValue(second, "url"); got != imageURI {
		t.Fatalf("turn/start input[1].url = %q, want %q", got, imageURI)
	}
}

func TestPromptRejectsAudioContentBlock(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	_, err = agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt: []acp.ContentBlock{
			{
				Audio: &acp.ContentBlockAudio{
					MimeType: "audio/wav",
					Data:     "AAAA",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("Prompt() error = nil, want invalid params error")
	}
	if !strings.Contains(err.Error(), "unsupported prompt content block type: audio") {
		t.Fatalf("Prompt() error = %v, want unsupported audio message", err)
	}
	turnStartParams := session.turnStartParamsSnapshot()
	if len(turnStartParams) != 0 {
		t.Fatalf("turn/start calls = %d, want 0", len(turnStartParams))
	}
}

func TestNewSessionIncludesModelsFromModelList(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	session.modelListResponses = []appServerModelListResponse{
		{
			Data: []appServerModel{
				{ID: "gpt-5.4", DisplayName: "GPT-5.4", IsDefault: true},
				{ID: "gpt-5.4-mini", DisplayName: "GPT-5.4 mini"},
			},
		},
	}

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	resp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if resp.Models == nil {
		t.Fatal("NewSession().Models = nil, want non-nil")
	}
	if got := resp.Models.CurrentModelId; got != acp.ModelId("gpt-5.4") {
		t.Fatalf("current model = %q, want %q", got, "gpt-5.4")
	}
	if got := len(resp.Models.AvailableModels); got != 2 {
		t.Fatalf("available models len = %d, want 2", got)
	}
	if got := string(resp.Models.AvailableModels[0].ModelId); got != "gpt-5.4" {
		t.Fatalf("available models[0].modelId = %q, want %q", got, "gpt-5.4")
	}
}

func TestNewSessionIncludesReasoningEffortConfigOptionsFromModelList(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	session.modelListResponses = []appServerModelListResponse{
		{
			Data: []appServerModel{
				appServerModelWithReasoning("gpt-5.4", true, testReasoningXHigh, "minimal", "low", "medium", "high", testReasoningXHigh),
			},
		},
	}

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	resp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	option := requireReasoningEffortOption(t, resp.ConfigOptions)
	if got := option.CurrentValue; got != acp.SessionConfigValueId(testReasoningXHigh) {
		t.Fatalf("reasoning current value = %q, want xhigh", got)
	}
	if option.Category == nil || option.Category.Other == nil || *option.Category.Other != sessionConfigCategoryThoughtLevel {
		t.Fatalf("reasoning category = %#v, want %q", option.Category, sessionConfigCategoryThoughtLevel)
	}
	if !reasoningEffortOptionsInclude(option, testReasoningXHigh) {
		t.Fatalf("reasoning options missing xhigh: %#v", option.Options)
	}
}

func TestSetSessionConfigOptionReasoningEffortAppliesToNextTurn(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	session.modelListResponses = []appServerModelListResponse{
		{
			Data: []appServerModel{
				appServerModelWithReasoning("gpt-5.4", true, "medium", "low", "medium", "high", testReasoningXHigh),
			},
		},
		{
			Data: []appServerModel{
				appServerModelWithReasoning("gpt-5.4", true, "medium", "low", "medium", "high", testReasoningXHigh),
			},
		},
	}
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	setResp, err := agent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: newResp.SessionId,
			ConfigId:  acp.SessionConfigId(sessionConfigIDReasoningEffort),
			Value:     acp.SessionConfigValueId(testReasoningXHigh),
		},
	})
	if err != nil {
		t.Fatalf("SetSessionConfigOption() error = %v", err)
	}
	option := requireReasoningEffortOption(t, setResp.ConfigOptions)
	if got := option.CurrentValue; got != acp.SessionConfigValueId(testReasoningXHigh) {
		t.Fatalf("reasoning current value = %q, want xhigh", got)
	}

	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	turnStartParams := session.turnStartParamsSnapshot()
	if len(turnStartParams) != 1 {
		t.Fatalf("turn/start calls = %d, want 1", len(turnStartParams))
	}
	if got := stringValue(turnStartParams[0], "effort"); got != testReasoningXHigh {
		t.Fatalf("turn/start effort = %q, want xhigh", got)
	}
}

func TestSetSessionConfigOptionRejectsUnsupportedReasoningEffort(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	session.modelListResponses = []appServerModelListResponse{
		{
			Data: []appServerModel{
				appServerModelWithReasoning("gpt-5.4", true, "medium", "low", "medium", "high"),
			},
		},
		{
			Data: []appServerModel{
				appServerModelWithReasoning("gpt-5.4", true, "medium", "low", "medium", "high"),
			},
		},
	}

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	_, err = agent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: newResp.SessionId,
			ConfigId:  acp.SessionConfigId(sessionConfigIDReasoningEffort),
			Value:     acp.SessionConfigValueId(testReasoningXHigh),
		},
	})
	if err == nil {
		t.Fatal("SetSessionConfigOption() error = nil, want unsupported value error")
	}
	if !strings.Contains(err.Error(), sessionConfigOptionValueUnsupported) {
		t.Fatalf("SetSessionConfigOption() error = %v, want unsupported value", err)
	}
}

func TestSetSessionConfigOptionRejectsUnknownConfigID(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	_, err = agent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: newResp.SessionId,
			ConfigId:  acp.SessionConfigId("unknown"),
			Value:     acp.SessionConfigValueId(testReasoningXHigh),
		},
	})
	if err == nil {
		t.Fatal("SetSessionConfigOption() error = nil, want unsupported config error")
	}
	if !strings.Contains(err.Error(), sessionConfigOptionIDUnsupported) {
		t.Fatalf("SetSessionConfigOption() error = %v, want unsupported config", err)
	}
}

func TestNewSessionModelListPaginationAndThreadModelPrecedence(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	session.threadStartResp.Model = "gpt-5.4-mini"
	cursor := "page-2"
	session.modelListResponses = []appServerModelListResponse{
		{
			Data: []appServerModel{
				{ID: "gpt-5.4", DisplayName: "GPT-5.4", IsDefault: true},
			},
			NextCursor: &cursor,
		},
		{
			Data: []appServerModel{
				{ID: "gpt-5.4-mini", DisplayName: "GPT-5.4 mini"},
			},
		},
	}

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	resp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if resp.Models == nil {
		t.Fatal("NewSession().Models = nil, want non-nil")
	}
	if got := resp.Models.CurrentModelId; got != acp.ModelId("gpt-5.4-mini") {
		t.Fatalf("current model = %q, want %q", got, "gpt-5.4-mini")
	}
	if got := len(resp.Models.AvailableModels); got != 2 {
		t.Fatalf("available models len = %d, want 2", got)
	}
	if got := len(session.modelListParamsSnapshot()); got != 2 {
		t.Fatalf("model/list calls = %d, want 2", got)
	}
}

func TestNewSessionContinuesWhenModelListFails(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	session.modelListErr = errors.New("boom")

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	resp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if got := strings.TrimSpace(string(resp.SessionId)); got == "" {
		t.Fatal("session id = empty, want non-empty")
	}
	if resp.Models != nil {
		t.Fatalf("NewSession().Models = %#v, want nil on model/list error", resp.Models)
	}
}

func TestHandleNotificationCommandExecOutputDeltaDoesNotEmitThought(t *testing.T) {
	sessionID := acp.SessionId("s1")
	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return nil, errors.New("not used")
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)
	agent.sessions[sessionID] = &codexProxySessionState{
		threadID: "thr-1",
		turnID:   "turn-1",
	}

	raw, err := json.Marshal(map[string]any{
		"threadId":    "thr-1",
		"turnId":      "turn-1",
		"processId":   "proc-1",
		"stream":      "stdout",
		"deltaBase64": "QUJDRA==",
		"capReached":  false,
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	done, stopReason, usage, err := agent.handleNotification(context.Background(), sessionID, "thr-1", "turn-1", true, &appServerNotification{
		Method: "command/exec/outputDelta",
		Params: raw,
	})
	if err != nil {
		t.Fatalf("handleNotification() error = %v", err)
	}
	if done {
		t.Fatalf("handleNotification() done = %v, want false", done)
	}
	if stopReason != "" {
		t.Fatalf("handleNotification() stopReason = %q, want empty", stopReason)
	}
	if usage != nil {
		t.Fatalf("handleNotification() usage = %#v, want nil", usage)
	}

	updates := conn.sessionUpdates(sessionID)
	if containsThoughtSubstring(updates, "QUJDRA==") {
		t.Fatalf("unexpected raw delta payload in thought update: %#v", updates)
	}
	if countThoughtChunks(updates) != 0 {
		t.Fatalf("unexpected thought updates for command/exec output delta: %#v", updates)
	}
}

func TestUsageFromTokenNotificationUsesLastFieldsOnly(t *testing.T) {
	usage := usageFromTokenNotification(map[string]any{
		"tokenUsage": map[string]any{
			"last": map[string]any{
				"inputTokens":       10,
				"outputTokens":      2,
				"totalTokens":       12,
				"cachedInputTokens": 4,
			},
			"total": map[string]any{
				"inputTokens": 999,
			},
			"modelContextWindow": 200000,
		},
	})
	if usage == nil {
		t.Fatal("usage = nil, want non-nil")
	}
	if got := usage["inputTokens"]; got != 10 {
		t.Fatalf("usage.inputTokens = %#v, want %d", got, 10)
	}
	if got := usage["outputTokens"]; got != 2 {
		t.Fatalf("usage.outputTokens = %#v, want %d", got, 2)
	}
	if got := usage["totalTokens"]; got != 12 {
		t.Fatalf("usage.totalTokens = %#v, want %d", got, 12)
	}
	if got := usage["cachedReadTokens"]; got != 4 {
		t.Fatalf("usage.cachedReadTokens = %#v, want %d", got, 4)
	}
	if _, ok := usage["modelContextWindow"]; ok {
		t.Fatalf("usage unexpectedly includes modelContextWindow: %#v", usage)
	}
}

func TestResolveAgentIdentityFromUserAgent(t *testing.T) {
	name, version := resolveAgentIdentity("", parseAppServerIdentity("codex_vscode/0.1.0 (darwin)"))
	if name != "codex_vscode" {
		t.Fatalf("name = %q, want %q", name, "codex_vscode")
	}
	if version != "0.1.0" {
		t.Fatalf("version = %q, want %q", version, "0.1.0")
	}
}

func TestPromptStreamsAppServerNotificationsToACPUpdates(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "item/started", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":    "commandExecution",
			"id":      "item-cmd-1",
			"status":  "inProgress",
			"command": "go test ./...",
		},
	})
	queueNotification(session, "item/commandExecution/outputDelta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-cmd-1",
		"delta":    "  ok   ./...",
	})
	queueNotification(session, "item/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":   "commandExecution",
			"id":     "item-cmd-1",
			"status": "completed",
		},
	})
	queueNotification(session, "turn/plan/updated", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"plan": []any{
			map[string]any{"step": "Run tests", "status": "completed"},
		},
	})
	queueNotification(session, "item/agentMessage/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-msg-1",
		"delta":    "Hi",
	})
	queueNotification(session, "item/agentMessage/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-msg-1",
		"delta":    " ",
	})
	queueNotification(session, "item/agentMessage/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-msg-1",
		"delta":    "done",
	})
	queueNotification(session, "item/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":  "agentMessage",
			"id":    "item-msg-1",
			"text":  "Hi done",
			"phase": "final_answer",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	promptResp, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("StopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if len(updates) == 0 {
		t.Fatal("expected ACP session updates")
	}
	if !containsAgentMessageText(updates, "Hi done") {
		t.Fatalf("missing completed agent message in ACP updates: %#v", updates)
	}
	if !containsToolCallText(updates, "  ok   ./...") {
		t.Fatalf("missing tool call output delta with leading spaces in ACP updates: %#v", updates)
	}
	if !containsPlanEntry(updates, "Run tests") {
		t.Fatalf("missing plan update in ACP updates: %#v", updates)
	}
	if !containsToolCall(updates, "codex-item-item-cmd-1") {
		t.Fatalf("missing tool call start/update in ACP updates: %#v", updates)
	}
}

func TestPromptDoesNotProjectNonToolItemLifecycleAsToolCalls(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	for _, itemType := range []string{"reasoning", "plan", "agentMessage"} {
		itemID := "item-" + itemType + "-1"
		queueNotification(session, "item/started", map[string]any{
			"threadId": "thr-1",
			"turnId":   "turn-1",
			"item": map[string]any{
				"type":   itemType,
				"id":     itemID,
				"status": "inProgress",
			},
		})
		queueNotification(session, "item/completed", map[string]any{
			"threadId": "thr-1",
			"turnId":   "turn-1",
			"item": map[string]any{
				"type":   itemType,
				"id":     itemID,
				"status": "completed",
			},
		})
	}
	queueNotification(session, "item/reasoning/textDelta", map[string]any{
		"threadId":     "thr-1",
		"turnId":       "turn-1",
		"itemId":       "item-reasoning-1",
		"contentIndex": 0,
		"delta":        "thinking",
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	promptResp, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("StopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if countToolCallEvents(updates) != 0 {
		t.Fatalf("unexpected tool call lifecycle updates for non-tool items: %#v", updates)
	}
	if countThoughtText(updates, "thinking") != 1 {
		t.Fatalf("missing reasoning text delta thought update: %#v", updates)
	}
}

func TestPromptSuppressesCommentaryAgentMessage(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "item/agentMessage/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-msg-1",
		"delta":    "working",
	})
	queueNotification(session, "item/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":  "agentMessage",
			"id":    "item-msg-1",
			"text":  "working",
			"phase": "commentary",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if countAgentMessageChunks(updates) != 0 {
		t.Fatalf("unexpected agent message chunks for commentary: %#v", updates)
	}
	if countThoughtChunks(updates) != 0 {
		t.Fatalf("unexpected thought chunks for commentary: %#v", updates)
	}
}

func TestPromptForwardsCompletedAgentMessageForVisiblePhases(t *testing.T) {
	tests := []struct {
		name     string
		hasPhase bool
		phase    any
	}{
		{name: "final answer", hasPhase: true, phase: "final_answer"},
		{name: "null phase", hasPhase: true, phase: nil},
		{name: "empty phase", hasPhase: true, phase: ""},
		{name: "missing phase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
			item := map[string]any{
				"type": "agentMessage",
				"id":   "item-msg-1",
				"text": "visible answer",
			}
			if tt.hasPhase {
				item["phase"] = tt.phase
			}
			queueNotification(session, "item/completed", map[string]any{
				"threadId": "thr-1",
				"turnId":   "turn-1",
				"item":     item,
			})
			queueNotification(session, "turn/completed", map[string]any{
				"threadId": "thr-1",
				"turn": map[string]any{
					"id":     "turn-1",
					"status": "completed",
				},
			})

			conn := &fakeACPAppConnection{}
			l := zerolog.Nop()
			agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
				return session, nil
			}, "agent", codexAppConfig{}, &l)
			agent.setConnection(conn)

			newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
			if err != nil {
				t.Fatalf("NewSession() error = %v", err)
			}
			if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
				SessionId: newResp.SessionId,
				Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
			}); err != nil {
				t.Fatalf("Prompt() error = %v", err)
			}

			updates := conn.sessionUpdates(newResp.SessionId)
			if !containsAgentMessageText(updates, "visible answer") {
				t.Fatalf("missing visible agent message: %#v", updates)
			}
			if countThoughtChunks(updates) != 0 {
				t.Fatalf("unexpected thought chunks for visible agent message: %#v", updates)
			}
		})
	}
}

func TestPromptPrefersCompletedAgentMessageTextOverBufferedDeltas(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "item/agentMessage/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-msg-1",
		"delta":    "draft",
	})
	queueNotification(session, "item/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":  "agentMessage",
			"id":    "item-msg-1",
			"text":  "final",
			"phase": "final_answer",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if !containsAgentMessageText(updates, "final") {
		t.Fatalf("missing completed agent message text: %#v", updates)
	}
	if containsAgentMessageText(updates, "draft") {
		t.Fatalf("unexpected buffered draft agent message text: %#v", updates)
	}
}

func TestCompletedFinalAnswerDoesNotCompletePrompt(t *testing.T) {
	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return nil, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)
	sessionID := acp.SessionId("session-phase")
	agent.mu.Lock()
	agent.sessions[sessionID] = &codexProxySessionState{}
	agent.mu.Unlock()

	raw, err := json.Marshal(map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":  "agentMessage",
			"id":    "item-msg-1",
			"text":  "done",
			"phase": "final_answer",
		},
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	done, stopReason, usage, err := agent.handleNotification(context.Background(), sessionID, "thr-1", "turn-1", true, &appServerNotification{
		Method: "item/completed",
		Params: raw,
	})
	if err != nil {
		t.Fatalf("handleNotification() error = %v", err)
	}
	if done {
		t.Fatalf("handleNotification() done = %v, want false", done)
	}
	if stopReason != "" {
		t.Fatalf("handleNotification() stopReason = %q, want empty", stopReason)
	}
	if usage != nil {
		t.Fatalf("handleNotification() usage = %#v, want nil", usage)
	}
	if !containsAgentMessageText(conn.sessionUpdates(sessionID), "done") {
		t.Fatalf("missing final answer agent message update: %#v", conn.sessionUpdates(sessionID))
	}
}

func TestPromptBridgesCommandApprovalRequest(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "item/commandExecution/requestApproval", json.RawMessage("1"), map[string]any{
		"threadId":           "thr-1",
		"turnId":             "turn-1",
		"itemId":             "item-cmd-1",
		"command":            "curl example.com",
		"availableDecisions": []any{decisionDecline, decisionAcceptForSession},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-2"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	if got := len(conn.permissionRequests); got != 1 {
		t.Fatalf("permission requests = %d, want 1", got)
	}
	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	decision := stringValue(responses[0], "decision")
	if decision != decisionAcceptForSession {
		t.Fatalf("approval decision = %q, want %q", decision, decisionAcceptForSession)
	}
}

func TestPromptFallbackRespondsUnsupportedServerRequest(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "custom/unknown", json.RawMessage("2"), map[string]any{"foo": "bar"})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	errResponses := session.errorResponsesSnapshot()
	if len(errResponses) != 1 {
		t.Fatalf("error responses = %d, want 1", len(errResponses))
	}
	if got := errResponses[0]["message"]; got != "unsupported server request" {
		t.Fatalf("fallback error message = %v, want %q", got, "unsupported server request")
	}
}

func TestPromptMapsExtendedNotifications(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "thread/started", map[string]any{
		"thread": map[string]any{
			"id": "thr-1",
		},
	})
	queueNotification(session, "turn/started", map[string]any{
		"threadId": "thr-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "inProgress",
		},
	})
	queueNotification(session, "thread/status/changed", map[string]any{
		"threadId": "thr-1",
		"status": map[string]any{
			"type":        "active",
			"activeFlags": []any{"waitingOnApproval"},
		},
	})
	// Duplicate status should be deduplicated.
	queueNotification(session, "thread/status/changed", map[string]any{
		"threadId": "thr-1",
		"status": map[string]any{
			"type":        "active",
			"activeFlags": []any{"waitingOnApproval"},
		},
	})
	queueNotification(session, "item/plan/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-plan-1",
		"delta":    "Run ",
	})
	queueNotification(session, "item/plan/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-plan-1",
		"delta":    "tests",
	})
	queueNotification(session, "item/reasoning/summaryPartAdded", map[string]any{
		"threadId":     "thr-1",
		"turnId":       "turn-1",
		"itemId":       "item-reason-1",
		"summaryIndex": 2,
	})
	queueNotification(session, "item/started", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"type":    "commandExecution",
			"id":      "item-cmd-1",
			"status":  "inProgress",
			"command": "echo hi",
		},
	})
	queueNotification(session, "item/commandExecution/terminalInteraction", map[string]any{
		"threadId":  "thr-1",
		"turnId":    "turn-1",
		"itemId":    "item-cmd-1",
		"processId": "p-1",
		"stdin":     "y\n",
	})
	queueNotification(session, "item/fileChange/patchUpdated", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-file-1",
		"changes": []any{
			map[string]any{
				"path": "README.md",
				"kind": map[string]any{
					"type": "update",
				},
				"diff": "@@ -1 +1 @@\n-old\n+new",
			},
		},
	})
	queueNotification(session, "item/autoApprovalReview/started", map[string]any{
		"threadId":     "thr-1",
		"turnId":       "turn-1",
		"targetItemId": "item-cmd-1",
		"review": map[string]any{
			"status":    "inProgress",
			"riskLevel": "highRiskCyberActivity",
			"rationale": "command touches network",
		},
	})
	queueNotification(session, "item/autoApprovalReview/completed", map[string]any{
		"threadId":     "thr-1",
		"turnId":       "turn-1",
		"targetItemId": "item-cmd-1",
		"review": map[string]any{
			"status":    "approved",
			"riskLevel": "highRiskCyberActivity",
		},
	})
	queueNotification(session, "hook/started", map[string]any{
		"threadId": "thr-1",
		"run": map[string]any{
			"id":        "hk-1",
			"eventName": "preCommand",
			"status":    "running",
		},
	})
	queueNotification(session, "hook/completed", map[string]any{
		"threadId": "thr-1",
		"run": map[string]any{
			"id":            "hk-1",
			"eventName":     "preCommand",
			"status":        "completed",
			"statusMessage": "ok",
		},
	})
	queueNotification(session, "mcpServer/startupStatus/updated", map[string]any{
		"name":   "docs",
		"status": statusFailed,
		"error":  "spawn failed",
	})
	queueNotification(session, "mcpServer/oauthLogin/completed", map[string]any{
		"name":    "docs",
		"success": false,
		"error":   "device code expired",
	})
	queueNotification(session, "model/rerouted", map[string]any{
		"threadId":  "thr-1",
		"turnId":    "turn-1",
		"fromModel": "gpt-5.4",
		"toModel":   "gpt-5.4-mini",
		"reason":    "highRiskCyberActivity",
	})
	queueNotification(session, "model/verification", map[string]any{
		"threadId":      "thr-1",
		"turnId":        "turn-1",
		"verifications": []any{"trustedAccessForCyber"},
	})
	queueNotification(session, "configWarning", map[string]any{
		"summary": "Invalid config value",
		"path":    "/tmp/config.toml",
		"details": "using default",
	})
	queueNotification(session, "deprecationNotice", map[string]any{
		"summary": "old flag is deprecated",
		"details": "use --new-flag",
	})
	queueNotification(session, "account/login/completed", map[string]any{
		"accountId": "acc-1",
	})
	queueNotification(session, "account/updated", map[string]any{})
	queueNotification(session, "app/list/updated", map[string]any{})
	queueNotification(session, "skills/changed", map[string]any{})
	queueNotification(session, "externalAgentConfig/import/completed", map[string]any{})
	queueNotification(session, "guardianWarning", map[string]any{
		"threadId": "thr-1",
		"message":  "guardian warning",
	})
	queueNotification(session, "warning", map[string]any{
		"threadId": "thr-1",
		"message":  "warning",
	})
	queueNotification(session, "thread/compacted", map[string]any{
		"threadId": "thr-1",
	})
	queueNotification(session, "thread/archived", map[string]any{
		"threadId": "thr-1",
	})
	queueNotification(session, "thread/unarchived", map[string]any{
		"threadId": "thr-1",
	})
	queueNotification(session, "thread/closed", map[string]any{
		"threadId": "thr-1",
	})
	queueNotification(session, "thread/name/updated", map[string]any{
		"threadId":   "thr-1",
		"threadName": "ACP Mapping Thread",
	})
	queueNotification(session, "windows/worldWritableWarning", map[string]any{
		"extraCount": 2,
		"failedScan": false,
		"samplePaths": []any{
			"C:/tmp/a",
			"C:/tmp/b",
		},
	})
	queueNotification(session, "windowsSandbox/setupCompleted", map[string]any{
		"mode":    "sandbox",
		"success": true,
		"error":   nil,
	})
	queueNotification(session, "thread/realtime/started", map[string]any{
		"threadId":  "thr-1",
		"version":   "v1",
		"sessionId": "rt-1",
	})
	queueNotification(session, "thread/realtime/itemAdded", map[string]any{
		"threadId": "thr-1",
		"item": map[string]any{
			"type": "message",
			"id":   "rt-item-1",
		},
	})
	queueNotification(session, "thread/realtime/outputAudio/delta", map[string]any{
		"threadId": "thr-1",
		"audio": map[string]any{
			"itemId":      "rt-item-1",
			"sampleRate":  24000,
			"numChannels": 1,
			"data":        "AQID",
		},
	})
	queueNotification(session, "thread/realtime/transcriptUpdated", map[string]any{
		"threadId": "thr-1",
		"role":     "assistant",
		"text":     "hello realtime",
	})
	queueNotification(session, "thread/realtime/transcript/delta", map[string]any{
		"threadId": "thr-1",
		"role":     "assistant",
		"delta":    "hello",
	})
	queueNotification(session, "thread/realtime/transcript/done", map[string]any{
		"threadId": "thr-1",
		"role":     "assistant",
		"text":     "hello",
	})
	queueNotification(session, "thread/realtime/sdp", map[string]any{
		"threadId": "thr-1",
		"sdp":      "v=0",
	})
	queueNotification(session, "thread/realtime/error", map[string]any{
		"threadId": "thr-1",
		"message":  "transport issue",
	})
	queueNotification(session, "thread/realtime/closed", map[string]any{
		"threadId": "thr-1",
		"reason":   "done",
	})
	queueNotification(session, "fs/changed", map[string]any{
		"watchId": "watch-1",
		"changedPaths": []any{
			"/tmp/a.go",
			"/tmp/b.go",
		},
	})
	queueNotification(session, "fuzzyFileSearch/sessionUpdated", map[string]any{
		"sessionId": "fuzzy-1",
		"query":     "agent",
		"files": []any{
			map[string]any{"path": "internal/apps/codexacpbridge/agent.go", "score": 0.9},
		},
	})
	queueNotification(session, "fuzzyFileSearch/sessionCompleted", map[string]any{
		"sessionId": "fuzzy-1",
	})
	queueNotification(session, "command/exec/outputDelta", map[string]any{
		"processId":   "proc-1",
		"stream":      "stdout",
		"deltaBase64": "AQID",
		"capReached":  false,
	})
	queueNotification(session, "turn/diff/updated", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"diff":     "@@ -1 +1 @@\n-foo\n+bar\n  tail",
	})
	queueNotification(session, "account/rateLimits/updated", map[string]any{
		"rateLimits": map[string]any{
			"planType": "plus",
			"primary": map[string]any{
				"usedPercent": 12,
			},
		},
	})
	queueNotification(session, "thread/tokenUsage/updated", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"tokenUsage": map[string]any{
			"last": map[string]any{
				"inputTokens":       10,
				"outputTokens":      2,
				"totalTokens":       12,
				"cachedInputTokens": 4,
			},
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	promptResp, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("StopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if !containsPlanEntry(updates, "Run tests") {
		t.Fatalf("missing aggregated plan delta update: %#v", updates)
	}
	if !containsToolCallText(updates, "y\n") {
		t.Fatalf("missing terminal interaction content: %#v", updates)
	}
	if !containsToolCallText(updates, "@@ -1 +1 @@\n-old\n+new") {
		t.Fatalf("missing patchUpdated tool call content: %#v", updates)
	}
	if !containsToolCall(updates, "codex-guardian-item-cmd-1") {
		t.Fatalf("missing guardian synthetic tool call updates: %#v", updates)
	}
	if !containsToolCall(updates, "codex-hook-hk-1") {
		t.Fatalf("missing hook synthetic tool call updates: %#v", updates)
	}
	if countThoughtText(updates, "Reasoning summary part added (#2).") != 0 {
		t.Fatalf("unexpected reasoning summary part thought update: %#v", updates)
	}
	if countThoughtChunks(updates) != 0 {
		t.Fatalf("unexpected non-reasoning thought updates: %#v", updates)
	}
	meta := promptResp.Meta
	if _, ok := meta["usage"]; !ok {
		t.Fatalf("PromptResponse.Meta.usage missing: %#v", promptResp.Meta)
	}
	if _, ok := meta["rateLimits"]; !ok {
		t.Fatalf("PromptResponse.Meta.rateLimits missing: %#v", promptResp.Meta)
	}
}

func TestPromptStopsOnErrorNotificationWithoutRetry(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "error", map[string]any{
		"threadId":  "thr-1",
		"turnId":    "turn-1",
		"willRetry": false,
		"error": map[string]any{
			"message":           "fatal boom",
			"additionalDetails": "stacktrace",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	promptResp, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if promptResp.StopReason != acp.StopReasonRefusal {
		t.Fatalf("StopReason = %q, want %q", promptResp.StopReason, acp.StopReasonRefusal)
	}
	updates := conn.sessionUpdates(newResp.SessionId)
	if countThoughtChunks(updates) != 0 {
		t.Fatalf("unexpected thought updates after error notification: %#v", updates)
	}
}

func TestPromptMetaIncludesRequestedMCPStartupStatusOnly(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "mcpServer/startupStatus/updated", map[string]any{
		"name":   "docs",
		"status": statusFailed,
		"error":  "spawn failed",
	})
	queueNotification(session, "mcpServer/startupStatus/updated", map[string]any{
		"name":   "global-only",
		"status": "completed",
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: "/tmp/work",
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "docs",
					Command: "docs-server",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	promptResp, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	meta := promptResp.Meta
	codexMeta := mapValue(meta, "codex")
	mcpMeta := mapValue(codexMeta, "mcp")
	if got := stringValue(mcpMeta, "contract"); got != mcpContractMerge {
		t.Fatalf("PromptResponse.Meta.codex.mcp.contract = %q, want %q", got, mcpContractMerge)
	}
	requested := listValue(mcpMeta, "requested")
	if len(requested) != 1 {
		t.Fatalf("PromptResponse.Meta.codex.mcp.requested len = %d, want 1", len(requested))
	}
	requestedServer, ok := requested[0].(map[string]any)
	if !ok {
		t.Fatalf("PromptResponse.Meta.codex.mcp.requested[0] type = %T, want map[string]any", requested[0])
	}
	if got := stringValue(requestedServer, "name"); got != "docs" {
		t.Fatalf("PromptResponse.Meta.codex.mcp.requested[0].name = %q, want %q", got, "docs")
	}
	if got := stringValue(requestedServer, "transport"); got != testMCPTransportStdio {
		t.Fatalf("PromptResponse.Meta.codex.mcp.requested[0].transport = %q, want %q", got, testMCPTransportStdio)
	}
	startupStatus := mapValue(mcpMeta, "startupStatus")
	docsStatus := mapValue(startupStatus, "docs")
	if got := stringValue(docsStatus, "status"); got != statusFailed {
		t.Fatalf("PromptResponse.Meta.codex.mcp.startupStatus.docs.status = %q, want %q", got, statusFailed)
	}
	if got := stringValue(docsStatus, "error"); got != "spawn failed" {
		t.Fatalf("PromptResponse.Meta.codex.mcp.startupStatus.docs.error = %q, want %q", got, "spawn failed")
	}
	if _, ok := startupStatus["global-only"]; ok {
		t.Fatalf("PromptResponse.Meta.codex.mcp.startupStatus unexpectedly contains non-requested server: %#v", startupStatus)
	}
}

func TestPromptForwardsSessionScopedUpdatesAfterCompletion(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	queueNotification(session, "account/updated", map[string]any{})
	time.Sleep(50 * time.Millisecond)
	updates := conn.sessionUpdates(newResp.SessionId)
	if countThoughtChunks(updates) != 0 {
		t.Fatalf("unexpected thought updates after completion for session-scoped event: %#v", updates)
	}
}

func TestPromptRebindsTurnIDFromTurnStartedNotification(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	type promptResult struct {
		resp acp.PromptResponse
		err  error
	}
	resultCh := make(chan promptResult, 1)
	go func() {
		resp, promptErr := agent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: newResp.SessionId,
			Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
		})
		resultCh <- promptResult{resp: resp, err: promptErr}
	}()

	waitForCondition(t, time.Second, func() bool {
		return len(session.turnStartParamsSnapshot()) == 1
	}, "turn/start not observed before queuing turn events")
	queueNotification(session, "turn/started", map[string]any{
		"threadId": "thr-1",
		"turn": map[string]any{
			"id":     "turn-2",
			"status": "inProgress",
		},
	})
	queueNotification(session, "item/agentMessage/delta", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-2",
		"itemId":   "item-msg-1",
		"delta":    "rebound",
	})
	queueNotification(session, "item/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-2",
		"item": map[string]any{
			"type":  "agentMessage",
			"id":    "item-msg-1",
			"text":  "rebound",
			"phase": "final_answer",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-2",
		"turn": map[string]any{
			"id":     "turn-2",
			"status": "completed",
		},
	})

	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("Prompt() error = %v", res.err)
		}
		if res.resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("Prompt().StopReason = %q, want %q", res.resp.StopReason, acp.StopReasonEndTurn)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt completion after turn-id rebind")
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if !containsAgentMessageText(updates, "rebound") {
		t.Fatalf("missing rebound delta after turn-id rebind: %#v", updates)
	}
}

func TestPromptRebindsThreadIDFromThreadStartedNotification(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	conn := &fakeACPAppConnection{}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	type promptResult struct {
		resp acp.PromptResponse
		err  error
	}
	resultCh := make(chan promptResult, 1)
	go func() {
		resp, promptErr := agent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: newResp.SessionId,
			Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
		})
		resultCh <- promptResult{resp: resp, err: promptErr}
	}()

	waitForCondition(t, time.Second, func() bool {
		return len(session.turnStartParamsSnapshot()) == 1
	}, "turn/start not observed before queuing thread-rebind events")
	queueNotification(session, "thread/started", map[string]any{
		"thread": map[string]any{
			"id": "thr-2",
		},
	})
	queueNotification(session, "thread/status/changed", map[string]any{
		"threadId": "thr-2",
		"status": map[string]any{
			"type": "active",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-2",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	select {
	case res := <-resultCh:
		if res.err != nil {
			t.Fatalf("Prompt() error = %v", res.err)
		}
		if res.resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("Prompt().StopReason = %q, want %q", res.resp.StopReason, acp.StopReasonEndTurn)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt completion after thread-id rebind")
	}

	updates := conn.sessionUpdates(newResp.SessionId)
	if countThoughtChunks(updates) != 0 {
		t.Fatalf("unexpected thought updates after thread-id rebind: %#v", updates)
	}
}

func TestPromptBridgesToolCallRequestAsPartialForward(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "item/tool/call", json.RawMessage("3"), map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"callId":   "call-1",
		"tool":     "relay.agents.start",
		"arguments": map[string]any{
			"agent_name": "planner",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-1"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	if got, ok := responses[0]["success"].(bool); !ok || got {
		t.Fatalf("response.success = %v, want false", responses[0]["success"])
	}
	contentItems, ok := responses[0]["contentItems"].([]any)
	if !ok || len(contentItems) == 0 {
		t.Fatalf("response.contentItems missing: %#v", responses[0])
	}
	first, ok := contentItems[0].(map[string]any)
	if !ok {
		t.Fatalf("response.contentItems[0] type = %T, want map[string]any", contentItems[0])
	}
	if got := stringValue(first, "type"); got != "inputText" {
		t.Fatalf("response.contentItems[0].type = %q, want %q", got, "inputText")
	}
	if got := stringValue(first, "text"); !strings.Contains(got, "not executed by ACP bridge") {
		t.Fatalf("response.contentItems[0].text = %q, want contains %q", got, "not executed by ACP bridge")
	}
}

func TestPromptBridgesToolRequestUserInputRequest(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "item/tool/requestUserInput", json.RawMessage("4"), map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"itemId":   "item-ui-1",
		"questions": []any{
			map[string]any{
				"header":   "Scope",
				"id":       "scope",
				"question": "Which scope?",
				"options": []any{
					map[string]any{"label": "Tier1", "description": "must-have only"},
					map[string]any{"label": "Tier1+2", "description": "must + should"},
				},
			},
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-1"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	answers, ok := responses[0]["answers"].(map[string]any)
	if !ok {
		t.Fatalf("response.answers type = %T, want map[string]any", responses[0]["answers"])
	}
	scopeAnswer, ok := answers["scope"].(map[string]any)
	if !ok {
		t.Fatalf("response.answers.scope type = %T, want map[string]any", answers["scope"])
	}
	answerList, ok := scopeAnswer["answers"].([]string)
	if !ok {
		rawList, okAny := scopeAnswer["answers"].([]any)
		if !okAny {
			t.Fatalf("response.answers.scope.answers type = %T, want []string/[]any", scopeAnswer["answers"])
		}
		answerList = make([]string, 0, len(rawList))
		for _, raw := range rawList {
			s, _ := raw.(string)
			answerList = append(answerList, s)
		}
	}
	if len(answerList) != 1 || answerList[0] != "Tier1" {
		t.Fatalf("response.answers.scope.answers = %#v, want [Tier1]", answerList)
	}
}

func TestPromptBridgesMcpElicitationRequest(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "mcpServer/elicitation/request", json.RawMessage("5"), map[string]any{
		"serverName": "docs",
		"threadId":   "thr-1",
		"mode":       "form",
		"message":    "Need user input",
		"requestedSchema": map[string]any{
			"type": "object",
		},
		"_meta": map[string]any{
			"trace": "abc",
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-1"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	if got := stringValue(responses[0], "action"); got != "accept" {
		t.Fatalf("response.action = %q, want %q", got, "accept")
	}
	if _, ok := responses[0]["content"]; !ok {
		t.Fatalf("response.content missing for accept action: %#v", responses[0])
	}
	if _, ok := responses[0]["_meta"]; !ok {
		t.Fatalf("response._meta missing passthrough: %#v", responses[0])
	}
}

func TestPromptBridgesApplyPatchApprovalRequest(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "applyPatchApproval", json.RawMessage("6"), map[string]any{
		"callId":         "call-apply-1",
		"conversationId": "thr-1",
		"fileChanges": map[string]any{
			"README.md": map[string]any{"kind": "modify"},
		},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-2"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	if got := stringValue(responses[0], "decision"); got != "approved_for_session" {
		t.Fatalf("response.decision = %q, want %q", got, "approved_for_session")
	}
}

func TestPromptBridgesExecCommandApprovalRequest(t *testing.T) {
	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "execCommandApproval", json.RawMessage("7"), map[string]any{
		"callId":         "call-exec-1",
		"conversationId": "thr-1",
		"command":        []any{"curl", "example.com"},
		"cwd":            "/tmp/work",
		"parsedCmd":      []any{},
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-3"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	if got := stringValue(responses[0], "decision"); got != "denied" {
		t.Fatalf("response.decision = %q, want %q", got, "denied")
	}
}

func TestPromptBridgesChatgptAuthTokensRefreshFromEnv(t *testing.T) {
	t.Setenv("CODEX_CHATGPT_ACCESS_TOKEN", "token-1")
	t.Setenv("CODEX_CHATGPT_ACCOUNT_ID", "acct-1")
	t.Setenv("CODEX_CHATGPT_PLAN_TYPE", "plus")

	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "account/chatgptAuthTokens/refresh", json.RawMessage("8"), map[string]any{
		"reason": "expired",
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-1"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	responses := session.responsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %d, want 1", len(responses))
	}
	if got := stringValue(responses[0], "accessToken"); got != "token-1" {
		t.Fatalf("response.accessToken = %q, want %q", got, "token-1")
	}
	if got := stringValue(responses[0], "chatgptAccountId"); got != "acct-1" {
		t.Fatalf("response.chatgptAccountId = %q, want %q", got, "acct-1")
	}
	if got := stringValue(responses[0], "chatgptPlanType"); got != "plus" {
		t.Fatalf("response.chatgptPlanType = %q, want %q", got, "plus")
	}
}

func TestPromptBridgesChatgptAuthTokensRefreshUnavailable(t *testing.T) {
	t.Setenv("CODEX_CHATGPT_ACCESS_TOKEN", "")
	t.Setenv("CODEX_CHATGPT_ACCOUNT_ID", "")

	session := newFakeAppServerSession("codex_test/1.0.0", "thr-1", "turn-1")
	queueRequest(session, "account/chatgptAuthTokens/refresh", json.RawMessage("9"), map[string]any{
		"reason": "expired",
	})
	queueNotification(session, "turn/completed", map[string]any{
		"threadId": "thr-1",
		"turnId":   "turn-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	conn := &fakeACPAppConnection{
		permissionResponse: acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected("opt-1"),
		},
	}
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return session, nil
	}, "agent", codexAppConfig{}, &l)
	agent.setConnection(conn)

	newResp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp/work"})
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if _, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	}); err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	errResponses := session.errorResponsesSnapshot()
	if len(errResponses) != 1 {
		t.Fatalf("error responses = %d, want 1", len(errResponses))
	}
	if got := stringValue(errResponses[0], "message"); !strings.Contains(got, "chatgpt token refresh unavailable") {
		t.Fatalf("error message = %v, want contains %q", got, "chatgpt token refresh unavailable")
	}
}

func TestServerRequestResolvedClearsPendingRequest(t *testing.T) {
	sessionID := acp.SessionId("s1")
	l := zerolog.Nop()
	agent := newCodexACPProxyAgent(func(context.Context, string) (appServerSession, error) {
		return nil, errors.New("not used")
	}, "agent", codexAppConfig{}, &l)
	agent.sessions[sessionID] = &codexProxySessionState{
		threadID:         "thr-1",
		turnID:           "turn-1",
		pendingRequests:  map[string]string{"1": "item/tool/call"},
		planDeltaByItem:  map[string]string{},
		latestRateLimits: map[string]any{},
	}

	raw, err := json.Marshal(map[string]any{
		"threadId":  "thr-1",
		"requestId": 1,
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	done, stopReason, usage, err := agent.handleNotification(context.Background(), sessionID, "thr-1", "turn-1", true, &appServerNotification{
		Method: "serverRequest/resolved",
		Params: raw,
	})
	if err != nil {
		t.Fatalf("handleNotification() error = %v", err)
	}
	if done {
		t.Fatalf("handleNotification() done = %v, want %v", done, false)
	}
	if stopReason != "" {
		t.Fatalf("handleNotification() stopReason = %q, want empty", stopReason)
	}
	if usage != nil {
		t.Fatalf("handleNotification() usage = %#v, want nil", usage)
	}
	if got := len(agent.sessions[sessionID].pendingRequests); got != 0 {
		t.Fatalf("pending requests = %d, want 0", got)
	}
}

type fakeACPAppConnection struct {
	mu                 sync.Mutex
	permissionResponse acp.RequestPermissionResponse
	permissionError    error
	permissionRequests []acp.RequestPermissionRequest
	updates            []acp.SessionNotification
}

func (f *fakeACPAppConnection) SessionUpdate(_ context.Context, params acp.SessionNotification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, params)
	return nil
}

func (f *fakeACPAppConnection) RequestPermission(_ context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.permissionRequests = append(f.permissionRequests, params)
	if f.permissionError != nil {
		return acp.RequestPermissionResponse{}, f.permissionError
	}
	if f.permissionResponse.Outcome.Cancelled == nil && f.permissionResponse.Outcome.Selected == nil {
		if len(params.Options) > 0 {
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeSelected(params.Options[0].OptionId),
			}, nil
		}
		return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
	}
	return f.permissionResponse, nil
}

func (f *fakeACPAppConnection) sessionUpdates(sessionID acp.SessionId) []acp.SessionNotification {
	f.mu.Lock()
	defer f.mu.Unlock()
	filtered := make([]acp.SessionNotification, 0, len(f.updates))
	for _, update := range f.updates {
		if update.SessionId == sessionID {
			filtered = append(filtered, update)
		}
	}
	return filtered
}

type fakeAppServerSession struct {
	mu sync.Mutex

	initializeResp appServerInitializeResponse
	events         chan appServerEvent

	threadStartResp appServerThreadStartResponse
	turnStartResp   appServerTurnStartResponse
	modelListErr    error

	threadStartParams  []map[string]any
	turnStartParams    []map[string]any
	modelListParams    []map[string]any
	modelListCalls     int
	modelListResponses []appServerModelListResponse
	responses          []map[string]any
	errorResponses     []map[string]any
}

func newFakeAppServerSession(userAgent string, threadID string, turnID string) *fakeAppServerSession {
	threadResp := appServerThreadStartResponse{}
	threadResp.Thread.ID = threadID
	turnResp := appServerTurnStartResponse{}
	turnResp.Turn.ID = turnID
	return &fakeAppServerSession{
		initializeResp:  appServerInitializeResponse{UserAgent: userAgent},
		events:          make(chan appServerEvent, 64),
		threadStartResp: threadResp,
		turnStartResp:   turnResp,
	}
}

func appServerModelWithReasoning(
	id string,
	isDefault bool,
	defaultReasoningEffort string,
	supportedReasoningEfforts ...string,
) appServerModel {
	model := appServerModel{
		ID:                     id,
		DisplayName:            id,
		IsDefault:              isDefault,
		DefaultReasoningEffort: defaultReasoningEffort,
	}
	for _, effort := range supportedReasoningEfforts {
		model.SupportedReasoningEfforts = append(model.SupportedReasoningEfforts, appServerReasoningEffortOption{
			Description:     effort + " effort",
			ReasoningEffort: effort,
		})
	}
	return model
}

func (f *fakeAppServerSession) InitializeResponse() appServerInitializeResponse {
	return f.initializeResp
}

func (f *fakeAppServerSession) Events() <-chan appServerEvent {
	return f.events
}

func (f *fakeAppServerSession) ThreadStart(_ context.Context, params map[string]any) (appServerThreadStartResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.threadStartParams = append(f.threadStartParams, params)
	return f.threadStartResp, nil
}

func (f *fakeAppServerSession) TurnStart(_ context.Context, params map[string]any) (appServerTurnStartResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.turnStartParams = append(f.turnStartParams, params)
	return f.turnStartResp, nil
}

func (f *fakeAppServerSession) ModelList(_ context.Context, params map[string]any) (appServerModelListResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.modelListParams = append(f.modelListParams, cloneAnyMap(params))
	if f.modelListErr != nil {
		return appServerModelListResponse{}, f.modelListErr
	}
	if f.modelListCalls >= len(f.modelListResponses) {
		f.modelListCalls++
		return appServerModelListResponse{}, nil
	}
	resp := f.modelListResponses[f.modelListCalls]
	f.modelListCalls++
	return resp, nil
}

func (f *fakeAppServerSession) TurnInterrupt(_ context.Context, _ string, _ string) error {
	return nil
}

func (f *fakeAppServerSession) RespondRequest(_ context.Context, _ *appServerRequest, result any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := result.(map[string]any)
	if !ok {
		return errors.New("result must be map")
	}
	f.responses = append(f.responses, m)
	return nil
}

func (f *fakeAppServerSession) RespondRequestError(_ context.Context, _ *appServerRequest, code int, message string, data any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errorResponses = append(f.errorResponses, map[string]any{
		"code":    code,
		"message": message,
		"data":    data,
	})
	return nil
}

func (f *fakeAppServerSession) Close() error { return nil }
func (f *fakeAppServerSession) Wait() error  { return nil }

func (f *fakeAppServerSession) responsesSnapshot() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, len(f.responses))
	copy(out, f.responses)
	return out
}

func (f *fakeAppServerSession) threadStartParamsSnapshot() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, len(f.threadStartParams))
	copy(out, f.threadStartParams)
	return out
}

func (f *fakeAppServerSession) turnStartParamsSnapshot() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, len(f.turnStartParams))
	copy(out, f.turnStartParams)
	return out
}

func (f *fakeAppServerSession) errorResponsesSnapshot() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, len(f.errorResponses))
	copy(out, f.errorResponses)
	return out
}

func (f *fakeAppServerSession) modelListParamsSnapshot() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, len(f.modelListParams))
	copy(out, f.modelListParams)
	return out
}

func queueNotification(session *fakeAppServerSession, method string, params map[string]any) {
	raw, _ := json.Marshal(params)
	session.events <- appServerEvent{
		Notification: &appServerNotification{
			Method: method,
			Params: raw,
		},
	}
}

func queueRequest(session *fakeAppServerSession, method string, id json.RawMessage, params map[string]any) {
	raw, _ := json.Marshal(params)
	session.events <- appServerEvent{
		Request: &appServerRequest{
			ID:     id,
			Method: method,
			Params: raw,
		},
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool, failureMessage string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(failureMessage)
}

func containsAgentMessageText(updates []acp.SessionNotification, text string) bool {
	for _, update := range updates {
		if chunk := update.Update.AgentMessageChunk; chunk != nil && chunk.Content.Text != nil {
			if chunk.Content.Text.Text == text {
				return true
			}
		}
	}
	return false
}

func requireReasoningEffortOption(t *testing.T, options []acp.SessionConfigOption) *acp.SessionConfigOptionSelect {
	t.Helper()
	for _, option := range options {
		if option.Select == nil {
			continue
		}
		if option.Select.Id == acp.SessionConfigId(sessionConfigIDReasoningEffort) {
			return option.Select
		}
	}
	t.Fatalf("reasoning effort option missing: %#v", options)
	return nil
}

func reasoningEffortOptionsInclude(option *acp.SessionConfigOptionSelect, effort string) bool {
	if option == nil || option.Options.Ungrouped == nil {
		return false
	}
	for _, candidate := range *option.Options.Ungrouped {
		if candidate.Value == acp.SessionConfigValueId(effort) {
			return true
		}
	}
	return false
}

func countAgentMessageChunks(updates []acp.SessionNotification) int {
	count := 0
	for _, update := range updates {
		if update.Update.AgentMessageChunk != nil {
			count++
		}
	}
	return count
}

func containsPlanEntry(updates []acp.SessionNotification, step string) bool {
	for _, update := range updates {
		if plan := update.Update.Plan; plan != nil {
			for _, entry := range plan.Entries {
				if entry.Content == step {
					return true
				}
			}
		}
	}
	return false
}

func containsToolCall(updates []acp.SessionNotification, toolCallID string) bool {
	for _, update := range updates {
		if call := update.Update.ToolCall; call != nil && string(call.ToolCallId) == toolCallID {
			return true
		}
		if callUpdate := update.Update.ToolCallUpdate; callUpdate != nil && string(callUpdate.ToolCallId) == toolCallID {
			return true
		}
	}
	return false
}

func countToolCallEvents(updates []acp.SessionNotification) int {
	count := 0
	for _, update := range updates {
		if update.Update.ToolCall != nil {
			count++
		}
		if update.Update.ToolCallUpdate != nil {
			count++
		}
	}
	return count
}

func containsToolCallText(updates []acp.SessionNotification, text string) bool {
	for _, update := range updates {
		callUpdate := update.Update.ToolCallUpdate
		if callUpdate == nil {
			continue
		}
		for _, content := range callUpdate.Content {
			if content.Content == nil || content.Content.Content.Text == nil {
				continue
			}
			if content.Content.Content.Text.Text == text {
				return true
			}
		}
	}
	return false
}

func containsThoughtSubstring(updates []acp.SessionNotification, substring string) bool {
	for _, update := range updates {
		if chunk := update.Update.AgentThoughtChunk; chunk != nil && chunk.Content.Text != nil {
			if strings.Contains(chunk.Content.Text.Text, substring) {
				return true
			}
		}
	}
	return false
}

func countThoughtText(updates []acp.SessionNotification, text string) int {
	count := 0
	for _, update := range updates {
		if chunk := update.Update.AgentThoughtChunk; chunk != nil && chunk.Content.Text != nil {
			if chunk.Content.Text.Text == text {
				count++
			}
		}
	}
	return count
}

func countThoughtChunks(updates []acp.SessionNotification) int {
	count := 0
	for _, update := range updates {
		if update.Update.AgentThoughtChunk != nil {
			count++
		}
	}
	return count
}

func TestStopReasonFromTurnStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   acp.StopReason
	}{
		{"completed", "completed", acp.StopReasonEndTurn},
		{"interrupted", "interrupted", acp.StopReasonCancelled},
		{"failed", "failed", acp.StopReasonRefusal},
		{"empty", "", acp.StopReasonEndTurn},
		{"unknown status", "unknown_status", acp.StopReasonEndTurn},
		{"inProgress", "inProgress", acp.StopReasonEndTurn},
		{"cancelled", "cancelled", acp.StopReasonEndTurn},
		{"max_iterations", "max_iterations", acp.StopReasonEndTurn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stopReasonFromTurnStatus(tt.status)
			if got != tt.want {
				t.Errorf("stopReasonFromTurnStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestToACPToolCallStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   acp.ToolCallStatus
	}{
		{"inProgress", "inProgress", acp.ToolCallStatusInProgress},
		{"completed", "completed", acp.ToolCallStatusCompleted},
		{"failed", "failed", acp.ToolCallStatusFailed},
		{"declined", "declined", acp.ToolCallStatusFailed},
		{"empty", "", acp.ToolCallStatusInProgress},
		{"unknown", "unknown", acp.ToolCallStatusInProgress},
		{"pending", "pending", acp.ToolCallStatusInProgress},
		{"in_progress", "in_progress", acp.ToolCallStatusInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toACPToolCallStatus(tt.status)
			if got != tt.want {
				t.Errorf("toACPToolCallStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestToACPPlanStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   acp.PlanEntryStatus
	}{
		{"pending", "pending", acp.PlanEntryStatusPending},
		{"inProgress", "inProgress", acp.PlanEntryStatusInProgress},
		{"completed", "completed", acp.PlanEntryStatusCompleted},
		{"empty", "", acp.PlanEntryStatusPending},
		{"unknown", "unknown", acp.PlanEntryStatusPending},
		{"done", "done", acp.PlanEntryStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toACPPlanStatus(tt.status)
			if got != tt.want {
				t.Errorf("toACPPlanStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestPermissionToolCallID(t *testing.T) {
	tests := []struct {
		name     string
		rawInput map[string]any
		want     acp.ToolCallId
	}{
		{
			name:     "with itemId",
			rawInput: map[string]any{"itemId": "item-123"},
			want:     "codex-item-item-123",
		},
		{
			name:     "with tool name",
			rawInput: map[string]any{"tool": "myTool"},
			want:     "codex-tool-myTool",
		},
		{
			name:     "with patchId",
			rawInput: map[string]any{"patchId": "patch-456"},
			want:     "codex-patch-patch-456",
		},
		{
			name:     "with command",
			rawInput: map[string]any{"command": "ls -la"},
			want:     "codex-cmd-ls -la",
		},
		{
			name:     "with serverName",
			rawInput: map[string]any{"serverName": "myMcp"},
			want:     "codex-mcp-myMcp",
		},
		{
			name:     "empty input",
			rawInput: map[string]any{},
			want:     "codex-permission-unknown",
		},
		{
			name:     "nil input",
			rawInput: nil,
			want:     "codex-permission-unknown",
		},
		{
			name:     "priority: itemId over tool",
			rawInput: map[string]any{"itemId": "item-1", "tool": "myTool"},
			want:     "codex-item-item-1",
		},
		{
			name:     "priority: tool over patchId",
			rawInput: map[string]any{"tool": "myTool", "patchId": "patch-1"},
			want:     "codex-tool-myTool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := permissionToolCallID(tt.rawInput)
			if got != tt.want {
				t.Errorf("permissionToolCallID() = %v, want %v", got, tt.want)
			}
		})
	}
}
