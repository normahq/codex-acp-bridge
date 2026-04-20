package codexacp

import (
	"fmt"
	"strings"

	acp "github.com/coder/acp-go-sdk"
)

var sessionCodexMetaKeys = map[string]struct{}{
	"approvalPolicy":        {},
	"approvalsReviewer":     {},
	"baseInstructions":      {},
	"compactPrompt":         {},
	"config":                {},
	"developerInstructions": {},
	"ephemeral":             {},
	"modelProvider":         {},
	"personality":           {},
	"profile":               {},
	"sandbox":               {},
	"serviceTier":           {},
}

func sessionConfigFromNewSessionMeta(meta any, defaults codexAppConfig) (acp.SessionId, codexAppConfig, error) {
	cfg := defaults.clone()
	if meta == nil {
		return "", cfg, nil
	}
	metaMap, ok := meta.(map[string]any)
	if !ok {
		return "", codexAppConfig{}, fmt.Errorf("session/new _meta must be an object")
	}

	sessionID := acp.SessionId("")
	if rawSessionID, hasSessionID := metaMap["sessionId"]; hasSessionID {
		sessionIDText, ok := rawSessionID.(string)
		if !ok {
			return "", codexAppConfig{}, fmt.Errorf("session/new _meta.sessionId must be a string")
		}
		trimmed := strings.TrimSpace(sessionIDText)
		if trimmed == "" {
			return "", codexAppConfig{}, fmt.Errorf("session/new _meta.sessionId must not be empty")
		}
		sessionID = acp.SessionId(trimmed)
	}

	rawCodex, hasCodex := metaMap["codex"]
	if !hasCodex {
		return sessionID, cfg, nil
	}
	if rawCodex == nil {
		return sessionID, cfg, nil
	}

	codexMap, ok := rawCodex.(map[string]any)
	if !ok {
		return "", codexAppConfig{}, fmt.Errorf("session/new _meta.codex must be an object")
	}
	for key := range codexMap {
		if _, allowed := sessionCodexMetaKeys[key]; !allowed {
			return "", codexAppConfig{}, fmt.Errorf("session/new _meta.codex.%s is not supported", key)
		}
	}

	var err error
	if value, exists := codexMap["sandbox"]; exists {
		cfg.Sandbox, err = optionalMetaString("session/new _meta.codex.sandbox", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["approvalPolicy"]; exists {
		cfg.ApprovalPolicy, err = optionalMetaString("session/new _meta.codex.approvalPolicy", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["baseInstructions"]; exists {
		cfg.BaseInstructions, err = optionalMetaString("session/new _meta.codex.baseInstructions", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["developerInstructions"]; exists {
		cfg.DeveloperInstructions, err = optionalMetaString("session/new _meta.codex.developerInstructions", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["profile"]; exists {
		cfg.Profile, err = optionalMetaString("session/new _meta.codex.profile", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["compactPrompt"]; exists {
		cfg.CompactPrompt, err = optionalMetaString("session/new _meta.codex.compactPrompt", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["config"]; exists {
		cfg.Config, err = optionalMetaObject("session/new _meta.codex.config", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["approvalsReviewer"]; exists {
		cfg.ApprovalsReviewer, err = optionalMetaString("session/new _meta.codex.approvalsReviewer", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["personality"]; exists {
		cfg.Personality, err = optionalMetaString("session/new _meta.codex.personality", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["serviceTier"]; exists {
		cfg.ServiceTier, err = optionalMetaString("session/new _meta.codex.serviceTier", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["modelProvider"]; exists {
		cfg.ModelProvider, err = optionalMetaString("session/new _meta.codex.modelProvider", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}
	if value, exists := codexMap["ephemeral"]; exists {
		cfg.Ephemeral, err = optionalMetaBool("session/new _meta.codex.ephemeral", value)
		if err != nil {
			return "", codexAppConfig{}, err
		}
	}

	if err := validateEnumValue("codex approval policy", cfg.ApprovalPolicy, validCodexApprovalPolicies); err != nil {
		return "", codexAppConfig{}, err
	}
	if err := validateEnumValue("codex sandbox", cfg.Sandbox, validCodexSandboxModes); err != nil {
		return "", codexAppConfig{}, err
	}
	if err := validateEnumValue("codex approvals reviewer", cfg.ApprovalsReviewer, validCodexApprovalsReviewers); err != nil {
		return "", codexAppConfig{}, err
	}
	if err := validateEnumValue("codex personality", cfg.Personality, validCodexPersonalities); err != nil {
		return "", codexAppConfig{}, err
	}
	if err := validateEnumValue("codex service tier", cfg.ServiceTier, validCodexServiceTiers); err != nil {
		return "", codexAppConfig{}, err
	}

	return sessionID, cfg, nil
}

func optionalMetaString(label string, value any) (string, error) {
	if value == nil {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", label)
	}
	return strings.TrimSpace(text), nil
}

func optionalMetaObject(label string, value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", label)
	}
	return cloneMap(obj), nil
}

func optionalMetaBool(label string, value any) (*bool, error) {
	if value == nil {
		return nil, nil
	}
	boolValue, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf("%s must be a boolean", label)
	}
	out := boolValue
	return &out, nil
}
