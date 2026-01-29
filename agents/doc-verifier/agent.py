import asyncio
import subprocess
import os
import sys
from fast_agent import FastAgent

fast = FastAgent("Documentation Verifier")

def list_resources() -> str:
    """Lists all available skitz resource files.

    Returns:
        List of resource markdown files in /skitz/internal/resources/
    """
    try:
        import os
        resources_dir = "/skitz/internal/resources"
        if not os.path.exists(resources_dir):
            return "Resources directory not found. Ensure /skitz is mounted."

        files = [f for f in os.listdir(resources_dir) if f.endswith('.md')]
        return "\n".join(sorted(files))
    except Exception as e:
        return f"Error listing resources: {str(e)}"

def read_resource(filename: str) -> str:
    """Reads a skitz resource file.

    Args:
        filename: The resource filename (e.g., 'kubectl.md')

    Returns:
        Contents of the resource file
    """
    try:
        filepath = f"/skitz/internal/resources/{filename}"
        with open(filepath, 'r') as f:
            return f.read()
    except Exception as e:
        return f"Error reading {filename}: {str(e)}"

def run_shell_command(command: str) -> str:
    """Runs a shell command and returns the output.
    
    Args:
        command: The shell command to execute.
        
    Returns:
        The command output (stdout + stderr) or error message.
    """
    try:
        print(f"\n[Exec] Running: {command}")
        
        result = subprocess.run(
            ["sh", "-c", command],
            capture_output=True,
            text=True,
            timeout=30
        )
        
        parts = [
            f"Exit Code: {result.returncode}",
            "Output:",
            result.stdout
        ]
        
        if result.stderr:
            parts.extend([
                "STDERR:",
                result.stderr
            ])
            
        return "\n".join(parts)
    except Exception as e:
        return f"Error executing command: {str(e)}"

@fast.agent(
    "verifier",
    """You are a Skitz Documentation Verifier.

    ABOUT SKITZ:
    Skitz is a terminal UI command center that provides quick access to curated documentation
    and command references. Resources are markdown files containing organized collections of
    commands for various tools (kubectl, docker, git, etc.).

    YOUR MISSION:
    Verify that commands in Skitz resource files work as documented - WITHOUT causing any harm.

    CONTEXT:
    - Skitz repo mounted at /skitz (read-only)
    - Resources in /skitz/internal/resources/ (e.g., kubectl.md)
    - AGENT_RESOURCE env var = which resource to verify
    - AGENT_PROMPT = optional additional instructions

    WORKFLOW:
    1. Read AGENT_RESOURCE env var to get resource name
    2. Use read_resource(name + ".md") to read the file
    3. Extract shell commands from bash/sh code blocks
    4. Test SAFE commands only (see safety rules below)
    5. Report findings with clear summary

    SAFETY RULES - DO NOT RUN:
    - Destructive commands: rm, delete, drop, truncate, destroy
    - Write operations that modify files: >, >>, tee (unless to /tmp)
    - System modifications: apt, yum, brew install, systemctl
    - Commands requiring credentials or network access you don't have
    - For commands that support --dry-run, USE IT

    ONLY RUN:
    - Help/version commands: --help, --version, -h
    - Read-only queries: get, list, describe, show, cat (on safe files)
    - Safe demonstrations that don't modify state

    WHEN YOU CAN'T TEST:
    Mark command as "SKIPPED (unsafe/requires setup)" and note why.

    START NOW - read AGENT_RESOURCE and begin verification.
    """,
    servers=["fetch"],
    tools=[list_resources, read_resource, run_shell_command],
)
async def main():
    # Get resource and prompt from environment
    resource = os.environ.get("AGENT_RESOURCE", "")
    prompt = os.environ.get("AGENT_PROMPT", "")

    if not resource:
        print("Error: AGENT_RESOURCE not set. Specify which resource to verify.")
        sys.exit(1)

    # Build initial message
    initial_message = f"Verify the {resource} resource."
    if prompt:
        initial_message += f" {prompt}"

    async with fast.run() as agent:
        result = await agent.verifier.send(initial_message)
        print(result)

if __name__ == "__main__":
    asyncio.run(main())
