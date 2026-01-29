FROM astral/uv:python3.13-bookworm-slim

WORKDIR /app

# Initialize uv project and add fast-agent
RUN uv init -q . && uv add -q fast-agent-mcp

# Use fast-agent CLI directly - AGENT_PROMPT and AGENT_MODEL are passed at runtime
# Usage: docker run -e OPENAI_API_KEY=... -e AGENT_PROMPT="..." -e AGENT_MODEL=gpt-4o skitz-fastagent
ENTRYPOINT ["/bin/sh", "-c", "uv run fast-agent -q --model $AGENT_MODEL --message \"$AGENT_PROMPT\""]
