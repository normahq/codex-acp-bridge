package codexacp

import (
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"
)

func TestBuildTurnStartParamsImageDataURLFallback(t *testing.T) {
	params, err := buildTurnStartParams("thr-1", []acp.ContentBlock{
		{
			Image: &acp.ContentBlockImage{
				MimeType: "image/png",
				Data:     "QUJDRA==",
			},
		},
	}, "gpt-5.4", testReasoningXHigh)
	if err != nil {
		t.Fatalf("buildTurnStartParams() error = %v", err)
	}

	if got := stringValue(params, "threadId"); got != "thr-1" {
		t.Fatalf("threadId = %q, want %q", got, "thr-1")
	}
	if got := stringValue(params, "model"); got != "gpt-5.4" {
		t.Fatalf("model = %q, want %q", got, "gpt-5.4")
	}
	if got := stringValue(params, "effort"); got != testReasoningXHigh {
		t.Fatalf("effort = %q, want %q", got, testReasoningXHigh)
	}
	input := listValue(params, "input")
	if len(input) != 1 {
		t.Fatalf("input items = %d, want 1", len(input))
	}
	item, ok := input[0].(map[string]any)
	if !ok {
		t.Fatalf("input[0] type = %T, want map[string]any", input[0])
	}
	if got := stringValue(item, "url"); got != "data:image/png;base64,QUJDRA==" {
		t.Fatalf("input[0].url = %q, want data URL", got)
	}
}

func TestBuildTurnStartParamsRejectsUnsupportedResourceLink(t *testing.T) {
	_, err := buildTurnStartParams("thr-1", []acp.ContentBlock{
		{
			ResourceLink: &acp.ContentBlockResourceLink{
				Name: "repo",
				Uri:  "file:///tmp/repo",
			},
		},
	}, "", "")
	if err == nil {
		t.Fatal("buildTurnStartParams() error = nil, want unsupported resource_link error")
	}
	if !strings.Contains(err.Error(), "unsupported prompt content block type: resource_link") {
		t.Fatalf("buildTurnStartParams() error = %v, want unsupported resource_link", err)
	}
}

func TestBuildThreadResumeParamsIncludesThreadAndOverrides(t *testing.T) {
	params := buildThreadResumeParams(
		"thr-9",
		"/tmp/work",
		codexAppConfig{
			ApprovalPolicy:    testApprovalOnRequest,
			ApprovalsReviewer: testApprovalsReviewerGuard,
			ModelProvider:     "openai",
			Personality:       testPersonalityPragmatic,
			Sandbox:           "workspace-write",
			ServiceTier:       testServiceTierFlex,
		},
		testModelGPT54,
		nil,
	)

	if got := stringValue(params, "threadId"); got != "thr-9" {
		t.Fatalf("threadId = %q, want %q", got, "thr-9")
	}
	if got := stringValue(params, "cwd"); got != "/tmp/work" {
		t.Fatalf("cwd = %q, want %q", got, "/tmp/work")
	}
	if got, ok := boolValue(params, "excludeTurns"); !ok || !got {
		t.Fatalf("excludeTurns = %t (ok=%t), want true", got, ok)
	}
	if got := stringValue(params, "model"); got != testModelGPT54 {
		t.Fatalf("model = %q, want %q", got, testModelGPT54)
	}
}

func TestBuildThreadSettingsUpdateParams(t *testing.T) {
	params := buildThreadSettingsUpdateParams("thr-9", testModelGPT54, testReasoningXHigh)
	if got := stringValue(params, "threadId"); got != "thr-9" {
		t.Fatalf("threadId = %q, want %q", got, "thr-9")
	}
	if got := stringValue(params, "model"); got != testModelGPT54 {
		t.Fatalf("model = %q, want %q", got, testModelGPT54)
	}
	if got := stringValue(params, "effort"); got != testReasoningXHigh {
		t.Fatalf("effort = %q, want %q", got, testReasoningXHigh)
	}
}
