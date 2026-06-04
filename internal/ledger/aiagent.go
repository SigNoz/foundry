// Copyright (c) 2026 SigNoz, Inc.
// Copyright 2026 Pulumi Corporation
// SPDX-License-Identifier: Apache-2.0

package ledger

import "strings"

// aiAgentDetector pairs an agent name with the marker environment variables
// that identify it.
type aiAgentDetector struct {
	name string
	envs []string
}

// These are sourced from https://github.com/unjs/std-env/blob/main/src/agents.ts and
// https://github.com/vercel/vercel/blob/main/packages/detect-agent/src/index.ts, as a reference for common
// environment variables set by AI agents and tools.
//
// Order matters: specific forms should be identified before broad IDE/tool markers.
var aiAgentDetectors = []aiAgentDetector{
	{name: "cursor", envs: []string{"CURSOR_TRACE_ID"}},
	{name: "cursor-cli", envs: []string{"CURSOR_AGENT"}},
	{name: "gemini", envs: []string{"GEMINI_CLI"}},
	{name: "codex", envs: []string{"CODEX_SANDBOX", "CODEX_CI", "CODEX_THREAD_ID"}},
	{name: "antigravity", envs: []string{"ANTIGRAVITY_AGENT"}},
	{name: "augment-cli", envs: []string{"AUGMENT_AGENT"}},
	{name: "opencode", envs: []string{"OPENCODE", "OPENCODE_CALLER", "OPENCODE_CLIENT"}},
	{name: "cowork", envs: []string{"CLAUDE_CODE_IS_COWORK"}},
	{name: "claude", envs: []string{"CLAUDECODE", "CLAUDE_CODE"}},
	{name: "replit", envs: []string{"REPL_ID"}},
	{name: "github-copilot", envs: []string{"COPILOT_MODEL", "COPILOT_ALLOW_ALL", "COPILOT_GITHUB_TOKEN"}},
	{name: "goose", envs: []string{"GOOSE_PROVIDER"}},
}

// aiAgentAliases collapses self-declared names to the detector names so a
// tool reports one name regardless of detection path.
var aiAgentAliases = map[string]string{
	"claude-code":        "claude",
	"github-copilot-cli": "github-copilot",
}

// AIAgent returns a normalized name for the AI coding agent driving the
// CLI (e.g. "claude", "cursor", "codex"), or "" if none is detected. "" means
// unknown, not human: some agents are undetectable by design.
//
// Precedence: AI_AGENT (self-declared) > known per-agent environment markers.
func AIAgent(getEnv func(string) string) string {
	if agent := normalizeAIAgent(getEnv("AI_AGENT")); agent != "" {
		return agent
	}

	for _, d := range aiAgentDetectors {
		for _, envVar := range d.envs {
			if getEnv(envVar) != "" {
				return d.name
			}
		}
	}

	return ""
}

// normalizeAIAgent lowercases the agent name and keeps the leading token,
// since self-declared values carry suffixes after the first underscore, e.g.
// "claude-code_2-1-161_agent" -> "claude-code".
func normalizeAIAgent(agent string) string {
	agent = strings.TrimSpace(strings.ToLower(agent))
	agent, _, _ = strings.Cut(agent, "_")
	if alias, ok := aiAgentAliases[agent]; ok {
		return alias
	}

	return agent
}
