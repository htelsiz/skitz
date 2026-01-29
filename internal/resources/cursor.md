# Cursor

`agent` start interactive agent session ^run
`agent "{{prompt}}"` run agent with prompt ^run:prompt
`agent --mode=plan` start in plan mode (design before coding) ^run
`agent --mode=ask` start in read-only ask mode ^run
`agent -p "{{prompt}}"` non-interactive mode, print response ^run:prompt
`agent -p "{{prompt}}" --output-format json` scripted JSON output ^run:prompt
`agent ls` list previous conversations ^run
`agent resume` resume most recent session ^run
`agent --resume "{{id}}"` resume specific conversation ^run:id
`& {{message}}` send to cloud agent (pick up on web/mobile) ^run:message

## Docs Settings

Add these URLs in Cursor Settings > Features > Docs:

**Go**
`https://pkg.go.dev/std` Go stdlib
`https://pkg.go.dev/github.com/charmbracelet/bubbletea` BubbleTea TUI
`https://pkg.go.dev/github.com/charmbracelet/lipgloss` Lipgloss styling
`https://pkg.go.dev/github.com/charmbracelet/glamour` Glamour markdown

**Web/JS**
`https://react.dev/reference/react` React
`https://nextjs.org/docs` Next.js
`https://tailwindcss.com/docs` Tailwind CSS
`https://docs.astro.build` Astro

**Python**
`https://docs.python.org/3/library/` Python stdlib
`https://fastapi.tiangolo.com/reference/` FastAPI
`https://docs.pydantic.dev/latest/` Pydantic

**AI/LLM**
`https://docs.anthropic.com/en/docs` Claude API
`https://platform.openai.com/docs` OpenAI API
`https://modelcontextprotocol.io/docs` MCP

**Cloud**
`https://docs.docker.com/reference/` Docker
`https://kubernetes.io/docs/reference/` Kubernetes
`https://learn.microsoft.com/en-us/cli/azure/` Azure CLI
`https://cloud.google.com/sdk/gcloud/reference` gcloud CLI

**Databases**
`https://www.postgresql.org/docs/current/` PostgreSQL
`https://redis.io/docs/latest/commands/` Redis
