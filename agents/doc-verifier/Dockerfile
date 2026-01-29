FROM astral/uv:python3.13-bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    nodejs npm git curl && rm -rf /var/lib/apt/lists/*

RUN curl https://cursor.com/install -fsSL | bash || true
RUN curl -fsSL https://opencode.ai/install | bash || true

WORKDIR /app

RUN uv init -q .
RUN uv add -q fast-agent-mcp mcp

COPY agent.py shell_server.py fastagent.config.yaml ./

ENV SANDBOX_RUNTIME=true

ENTRYPOINT ["/bin/sh", "-c", "uv run agent.py"]
