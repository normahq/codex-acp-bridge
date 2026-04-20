package codexacp

import (
	"fmt"
	"strings"
)

var (
	validCodexApprovalPolicies = map[string]struct{}{
		"untrusted":  {},
		"on-failure": {},
		"on-request": {},
		"never":      {},
	}
	validCodexApprovalsReviewers = map[string]struct{}{
		"user":              {},
		"guardian_subagent": {},
	}
	validCodexPersonalities = map[string]struct{}{
		"none":      {},
		"friendly":  {},
		"pragmatic": {},
	}
	validCodexServiceTiers = map[string]struct{}{
		"fast": {},
		"flex": {},
	}
	validCodexSandboxModes = map[string]struct{}{
		"read-only":          {},
		"workspace-write":    {},
		"danger-full-access": {},
	}
)

// Options configures Codex bridge backend -> ACP proxy behavior.
type Options struct {
	Name string
}

type codexAppConfig struct {
	ApprovalPolicy        string
	ApprovalsReviewer     string
	BaseInstructions      string
	CompactPrompt         string
	Config                map[string]any
	DeveloperInstructions string
	Ephemeral             *bool
	Model                 string
	ModelProvider         string
	Personality           string
	Profile               string
	Sandbox               string
	ServiceTier           string
}

func (c codexAppConfig) withModel(model string) codexAppConfig {
	next := c
	nextModel := strings.TrimSpace(model)
	if nextModel != "" {
		next.Model = nextModel
	}
	return next
}

func (c codexAppConfig) clone() codexAppConfig {
	next := c
	next.Config = cloneMap(c.Config)
	if c.Ephemeral != nil {
		ephemeral := *c.Ephemeral
		next.Ephemeral = &ephemeral
	}
	return next
}

func (o Options) appConfig() codexAppConfig {
	return codexAppConfig{}
}

func (o Options) validate() error {
	return nil
}

func validateEnumValue(label string, value string, allowed map[string]struct{}) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if _, ok := allowed[trimmed]; ok {
		return nil
	}
	return fmt.Errorf("invalid %s %q", label, trimmed)
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
