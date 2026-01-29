import asyncio
import subprocess
import os
import sys
from fast_agent import FastAgent

# Check if running in a container or E2B
# In E2B, we might not have access to docker daemon, so we run directly
# In local Docker, we are already in a container
IS_SANDBOX = os.environ.get("SANDBOX_RUNTIME", "false") == "true" or os.path.exists("/.dockerenv")

fast = FastAgent("Documentation Verifier")

@fast.tool
def run_shell_command(command: str) -> str:
    """Runs a shell command and returns the output.
    
    Args:
        command: The shell command to execute.
        
    Returns:
        The command output (stdout + stderr) or error message.
    """
    try:
        print(f"\n[Exec] Running: {command}")
        
        # In a sandbox environment (Docker or E2B), we run directly
        # The environment itself provides the isolation
        result = subprocess.run(
            ["sh", "-c", command],
            capture_output=True,
            text=True,
            timeout=30
        )
        
        output = result.stdout
        if result.stderr:
            output += "\nSTDERR:\n" + result.stderr
            
        return f"Exit Code: {result.returncode}\nOutput:\n{output}"
    except Exception as e:
        return f"Error executing command: {str(e)}"

@fast.agent(
    "verifier",
    """You are a Documentation Verifier. Your goal is to research a topic, extract commands, and verify them.
    
    1. Search for documentation on the user's topic.
    2. Extract key commands mentioned in the documentation.
    3. Verify these commands by running them.
    4. Report your findings:
       - Which commands worked?
       - Which commands failed and why?
       - Are the documentation examples accurate?
       
    You are running in a secure, disposable sandbox (Debian/Alpine based).
    You can install packages if needed (apt-get, apk, pip, etc).
    """,
    servers=["fetch"],
    tools=[run_shell_command],
)
async def main():
    # If a message is provided via env var (docker) or args, use it
    # Otherwise interactive mode
    
    initial_message = os.environ.get("AGENT_PROMPT")
    if len(sys.argv) > 1:
        # If args are passed, assume it's for the agent
        # But fast-agent CLI might handle args too, so check
        pass

    async with fast.run() as agent:
        if initial_message:
            # Single run mode
            result = await agent.verifier.run(initial_message)
            print(result)
        else:
            # Interactive mode
            await agent.verifier.interactive()

if __name__ == "__main__":
    asyncio.run(main())
