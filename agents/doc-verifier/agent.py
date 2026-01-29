import asyncio
import os
import sys

from fast_agent import FastAgent

agents = FastAgent(name="Resource Verification System")


@agents.agent(
    name="CommandRunner",
    instruction="""You execute shell commands to verify CLI tools exist.

## Tools
- run_command(command): Execute shell commands
- read_file(path): Read file contents  
- setup_environment(command): Install packages (use for npm/pip installs)

## Environment
Pre-installed: uv, uvx, python, pip, node, npm, npx, git, curl, cursor, opencode

## Installing Tools (if needed)
- npm tools: `npm install -g @e2b/cli` or `npx -y <pkg>`
- Python tools: `uvx <tool>` or `uv pip install <pkg>`

## Verification
Only run `<tool> --help` to verify a tool exists. Do NOT run actual commands.

## File Paths
- Resources: /skitz/internal/resources/{name}.md""",
    servers=["shell"],
    human_input=False,
)
@agents.agent(
    name="DocumentationResearcher",
    instruction="""Verify commands exist in official documentation.

Use the fetch tool to check if a command is documented at the official docs URL.

Common docs:
- e2b: https://e2b.dev/docs/cli
- docker: https://docs.docker.com/reference/cli/docker/
- gcloud: https://cloud.google.com/sdk/gcloud/reference
- az: https://learn.microsoft.com/en-us/cli/azure/reference-index
- git: https://git-scm.com/docs

Report: FOUND or NOT FOUND for each command checked.""",
    servers=["fetch"],
    human_input=False,
)
@agents.orchestrator(
    name="ResourceVerificationOrchestrator",
    instruction="""Verify CLI commands documented in a Skitz resource file.

## Agents
- **CommandRunner**: Reads files, runs `--help` commands
- **DocumentationResearcher**: Checks official docs when needed

## Workflow

1. Ask CommandRunner to read `/skitz/internal/resources/{resource}.md`
2. Ask CommandRunner to run `<tool> --help` to verify the CLI exists (install first if needed)
3. Compare documented commands against --help output
4. For any command NOT in --help, ask DocumentationResearcher to verify against official docs

## Report Format
```
Resource: {name}
Tool: {tool}

Verified (in --help): [list]
Not in --help but documented online: [list]  
Undocumented: [list]
```

Keep it simple - only verify existence, don't test commands.""",
    agents=["CommandRunner", "DocumentationResearcher"],
)
async def main() -> None:
    resource = os.environ.get("AGENT_RESOURCE", "")
    print(f"[DEBUG] AGENT_RESOURCE={resource}")

    if not resource:
        print("Error: AGENT_RESOURCE not set. Specify which resource to verify.")
        sys.exit(1)

    async with agents.run() as agent:
        print("[DEBUG] Starting verification...")
        result = await agent.ResourceVerificationOrchestrator.send(
            f"Verify the CLI commands documented in the resource file: {resource}"
        )
        print(result)


if __name__ == "__main__":
    asyncio.run(main())
