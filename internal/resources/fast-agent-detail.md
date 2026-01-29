## Decorators

`@fast.agent` basic agent decorator
`@fast.agent(model="sonnet")` agent with model
`@fast.agent(servers=["filesystem"])` agent with MCP servers
`@fast.workflow` workflow orchestrator decorator
`@fast.chain` sequential chain decorator
`@fast.parallel` parallel execution decorator
`@fast.evaluator_optimizer` optimization loop decorator
`@fast.router` conditional routing decorator

## Patterns

`async with fast.run() as agent:` run agent context
`await agent()` execute default agent
`await agent("prompt")` execute with prompt
`await agent.send("message")` send message to agent
`chain = agent1 | agent2` chain agents together

## MCP Servers

`servers=["filesystem", "fetch"]` built-in servers
`servers=["npx:server-name"]` npm-based servers
`servers=["uvx:server-name"]` uv-based servers
`servers=["docker:image"]` docker-based servers

## Config

`fastagent.config.yaml` main config file
`fastagent.secrets.yaml` secrets (API keys)
`ANTHROPIC_API_KEY` Claude API key env var
`OPENAI_API_KEY` OpenAI API key env var

## Models

`sonnet` Claude 3.5 Sonnet
`haiku` Claude 3.5 Haiku
`opus` Claude 3 Opus
`gpt-4o` GPT-4o
`o1` OpenAI o1
`o3-mini` OpenAI o3-mini
