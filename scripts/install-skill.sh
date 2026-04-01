#!/bin/bash
# 安装 personal-wiki Skill 到 OpenClaw 或 Claude Code

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKILL_SRC="$SCRIPT_DIR/../skills/personal-wiki/SKILL.md"
SKILL_NAME="personal-wiki"

TARGETS=(
  "$HOME/.openclaw/skills/$SKILL_NAME"
  "$HOME/.claude/skills/$SKILL_NAME"
)

for dir in "${TARGETS[@]}"; do
  mkdir -p "$dir"
  cp "$SKILL_SRC" "$dir/SKILL.md"
  echo "Installed: $dir/SKILL.md"
done
