#!/bin/bash
# PreToolUse hook: refuses tool calls that name a .env file (.env, .env.local,
# .env.production, ...), so Claude Code cannot read or change the project's
# secrets. Receives the hook JSON on stdin.
#
# Checks the fields that name a target: a shell command, file paths, and glob
# patterns. File contents being written are not checked, and neither are
# Grep search patterns, so searching code for the text ".env" still works.
#
# This is a guardrail against accidental access, not a security boundary: a
# deliberately obfuscated shell command could get past it. For OS-level
# enforcement, turn on the sandbox with sandbox.filesystem.denyRead.

input=$(cat)

targets=$(printf '%s' "$input" | jq -r '
  .tool_name as $tool
  | .tool_input
  | [
      (.command // empty),
      (.file_path // empty),
      (.notebook_path // empty),
      (.path // empty),
      (if $tool == "Glob" then (.pattern // empty) else empty end),
      (if $tool == "Grep" then (.glob // empty) else empty end)
    ]
  | join("\n")
' 2>/dev/null)

if printf '%s' "$targets" | grep -Eq '(^|[^A-Za-z0-9_])\.env([^A-Za-z0-9_]|$)'; then
  echo "Blocked: this project protects .env files from Claude Code. Ask the user to run the command themselves (for example with the ! prefix)." >&2
  exit 2
fi

exit 0
