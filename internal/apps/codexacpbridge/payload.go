package codexacp

import (
	"fmt"
	"strings"

	acp "github.com/coder/acp-go-sdk"
)

const (
	statusInProgress = "inProgress"
	statusCompleted  = "completed"
	statusFailed     = "failed"
)

func buildThreadStartParams(cwd string, cfg codexAppConfig, sessionModel string, sessionMCPServers map[string]acp.McpServer) map[string]any {
	params := map[string]any{
		"experimentalRawEvents":  false,
		"persistExtendedHistory": false,
	}
	if trimmedCWD := strings.TrimSpace(cwd); trimmedCWD != "" {
		params["cwd"] = trimmedCWD
	}

	effectiveCfg := cfg.withModel(sessionModel)
	if model := strings.TrimSpace(effectiveCfg.Model); model != "" {
		params["model"] = model
	}
	if modelProvider := strings.TrimSpace(effectiveCfg.ModelProvider); modelProvider != "" {
		params["modelProvider"] = modelProvider
	}
	if approval := strings.TrimSpace(effectiveCfg.ApprovalPolicy); approval != "" {
		params["approvalPolicy"] = approval
	}
	if approvalsReviewer := strings.TrimSpace(effectiveCfg.ApprovalsReviewer); approvalsReviewer != "" {
		params["approvalsReviewer"] = approvalsReviewer
	}
	if sandbox := strings.TrimSpace(effectiveCfg.Sandbox); sandbox != "" {
		params["sandbox"] = sandbox
	}
	if personality := strings.TrimSpace(effectiveCfg.Personality); personality != "" {
		params["personality"] = personality
	}
	if serviceTier := strings.TrimSpace(effectiveCfg.ServiceTier); serviceTier != "" {
		params["serviceTier"] = serviceTier
	}
	if effectiveCfg.Ephemeral != nil {
		params["ephemeral"] = *effectiveCfg.Ephemeral
	}
	if baseInstructions := strings.TrimSpace(effectiveCfg.BaseInstructions); baseInstructions != "" {
		params["baseInstructions"] = baseInstructions
	}
	if developerInstructions := strings.TrimSpace(effectiveCfg.DeveloperInstructions); developerInstructions != "" {
		params["developerInstructions"] = developerInstructions
	}

	config := cloneMap(effectiveCfg.Config)
	if config == nil {
		config = map[string]any{}
	}
	if profile := strings.TrimSpace(effectiveCfg.Profile); profile != "" {
		config["profile"] = profile
	}
	if compactPrompt := strings.TrimSpace(effectiveCfg.CompactPrompt); compactPrompt != "" {
		config["compact_prompt"] = compactPrompt
	}
	if mcpServersCfg := codexMCPServersConfig(sessionMCPServers); len(mcpServersCfg) > 0 {
		config["mcp_servers"] = mcpServersCfg
	}
	if len(config) > 0 {
		params["config"] = config
	}

	return params
}

func buildTurnStartParams(threadID string, prompt []acp.ContentBlock, model string) (map[string]any, error) {
	inputItems, err := buildTurnInputItems(prompt)
	if err != nil {
		return nil, err
	}
	if len(inputItems) == 0 {
		return nil, fmt.Errorf("prompt must include at least one text or image content block")
	}

	params := map[string]any{
		"threadId": strings.TrimSpace(threadID),
		"input":    inputItems,
	}
	if trimmedModel := strings.TrimSpace(model); trimmedModel != "" {
		params["model"] = trimmedModel
	}
	return params, nil
}

func buildTurnInputItems(prompt []acp.ContentBlock) ([]any, error) {
	items := make([]any, 0, len(prompt))
	for _, block := range prompt {
		switch {
		case block.Text != nil:
			trimmed := strings.TrimSpace(block.Text.Text)
			if trimmed == "" {
				continue
			}
			items = append(items, map[string]any{
				"type":          "text",
				"text":          trimmed,
				"text_elements": []any{},
			})
		case block.Image != nil:
			url, err := imageBlockURL(block.Image)
			if err != nil {
				return nil, err
			}
			items = append(items, map[string]any{
				"type": "image",
				"url":  url,
			})
		case block.Audio != nil:
			return nil, fmt.Errorf("unsupported prompt content block type: audio")
		case block.ResourceLink != nil:
			return nil, fmt.Errorf("unsupported prompt content block type: resource_link")
		case block.Resource != nil:
			return nil, fmt.Errorf("unsupported prompt content block type: resource")
		default:
			return nil, fmt.Errorf("unsupported prompt content block type: unknown")
		}
	}
	return items, nil
}

func imageBlockURL(block *acp.ContentBlockImage) (string, error) {
	if block == nil {
		return "", fmt.Errorf("image block is required")
	}
	if block.Uri != nil {
		if uri := strings.TrimSpace(*block.Uri); uri != "" {
			return uri, nil
		}
	}
	mimeType := strings.TrimSpace(block.MimeType)
	data := strings.TrimSpace(block.Data)
	if mimeType == "" || data == "" {
		return "", fmt.Errorf("image content block must include uri or mimeType+data")
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, data), nil
}

func codexMCPServersConfig(sessionMCPServers map[string]acp.McpServer) map[string]any {
	if len(sessionMCPServers) == 0 {
		return nil
	}
	result := make(map[string]any, len(sessionMCPServers))
	for name, server := range sessionMCPServers {
		serverCfg := map[string]any{}
		switch {
		case server.Stdio != nil:
			serverCfg["command"] = server.Stdio.Command
			if len(server.Stdio.Args) > 0 {
				serverCfg["args"] = server.Stdio.Args
			}
			if env := flattenEnvVars(server.Stdio.Env); len(env) > 0 {
				serverCfg["env"] = env
			}
		case server.Http != nil:
			serverCfg["url"] = server.Http.Url
			if headers := flattenHTTPHeaders(server.Http.Headers); len(headers) > 0 {
				serverCfg["http_headers"] = headers
			}
		default:
			continue
		}
		if len(serverCfg) > 0 {
			result[name] = serverCfg
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func toAppServerToolKind(itemType string) acp.ToolKind {
	switch itemType {
	case "commandExecution":
		return acp.ToolKindExecute
	case "fileChange":
		return acp.ToolKindEdit
	case "webSearch":
		return acp.ToolKindFetch
	case "mcpToolCall", "dynamicToolCall":
		return acp.ToolKindExecute
	case "imageView":
		return acp.ToolKindRead
	default:
		return acp.ToolKindOther
	}
}

func toACPToolCallStatus(status string) acp.ToolCallStatus {
	switch strings.TrimSpace(status) {
	case statusInProgress:
		return acp.ToolCallStatusInProgress
	case statusCompleted:
		return acp.ToolCallStatusCompleted
	case statusFailed, "declined":
		return acp.ToolCallStatusFailed
	default:
		return acp.ToolCallStatusInProgress
	}
}

func toACPPlanStatus(status string) acp.PlanEntryStatus {
	switch strings.TrimSpace(status) {
	case "pending":
		return acp.PlanEntryStatusPending
	case statusInProgress:
		return acp.PlanEntryStatusInProgress
	case statusCompleted:
		return acp.PlanEntryStatusCompleted
	default:
		return acp.PlanEntryStatusPending
	}
}
