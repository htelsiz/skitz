FROM astral/uv:python3.13-bookworm-slim

WORKDIR /app

# Initialize uv project
RUN uv init -q .

# Add dependencies
RUN uv add -q fast-agent-mcp

# Copy agent code
COPY agent.py .
COPY fastagent.config.yaml .

# Set env var to indicate sandbox environment
ENV SANDBOX_RUNTIME=true

# Entry point
ENTRYPOINT ["/bin/sh", "-c", "uv run agent.py"]
