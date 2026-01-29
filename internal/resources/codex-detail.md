# Codex CLI - AI Coding Agent

OpenAI's Codex CLI is a terminal-based AI coding agent that can understand your codebase, make edits, and execute commands.

## Core Concepts

- **Approval Modes**: Control how much autonomy the agent has
- **Project Context**: Automatically reads AGENTS.md, README, and project files
- **Sandboxing**: Optional isolation for command execution

## Approval Modes Explained

| Mode | File Edits | Commands | Best For |
|------|-----------|----------|----------|
| `suggest` | Manual approval | Manual approval | Learning, reviewing changes |
| `auto-edit` | Auto-applied | Manual approval | Trusted file changes |
| `full-auto` | Auto-applied | Auto-executed | CI/CD, automation |

## Model Selection

- **o4-mini** (default): Fast, cost-effective for most tasks
- **o3**: More capable reasoning for complex problems
- **gpt-4.1**: Alternative for specific use cases

## Project Documentation

Codex automatically looks for context in this order:
1. `AGENTS.md` - Agent-specific instructions
2. `CODEX.md` - Codex-specific instructions
3. `README.md` - General project documentation

Override with `--project-doc <file>` or skip with `--no-project-doc`.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `OPENAI_API_KEY` | API authentication (required) |
| `CODEX_HOME` | Custom config directory |
| `CODEX_SANDBOX_TYPE` | Default sandbox type |

## Safety & Sandboxing

Use `--writable-root` to restrict file modifications:
```bash
codex --full-auto --writable-root ./src "refactor utils"
```

The `--sandbox` flag runs all commands in an isolated environment.

## Tips & Best Practices

1. **Start with suggest mode** until you trust the output
2. **Use specific prompts** - "fix the null check in auth.go line 42" > "fix bugs"
3. **Leverage project docs** - keep AGENTS.md updated with conventions
4. **Review full-auto carefully** - check the plan before confirming

## Documentation

Visit [platform.openai.com/docs/codex](https://platform.openai.com/docs/codex) for full documentation.
