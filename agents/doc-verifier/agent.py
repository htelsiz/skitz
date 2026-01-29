import asyncio
import subprocess
import os
import sys
from fast_agent import FastAgent

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
    """You are a Documentation Verifier. Your goal is to research a topic, extract commands, and verify them.
    
    1. Search for documentation on the user's topic.
    2. Extract key commands mentioned in the documentation.
    3. Verify these commands by running them.
    4. Report your findings:
       - Which commands worked?
       - Which commands failed and why?
       - Are the documentation examples accurate?
    """,
    servers=["fetch"],
    tools=[run_shell_command],
)
async def main():
    # Priority: AGENT_PROMPT env var -> command line args -> error
    
    initial_message = os.environ.get("AGENT_PROMPT")
    if not initial_message and len(sys.argv) > 1:
        initial_message = " ".join(sys.argv[1:])
        
    if not initial_message:
        print("Error: No prompt provided. Set AGENT_PROMPT env var or pass as argument.")
        sys.exit(1)

    async with fast.run() as agent:
        result = await agent.verifier.run(initial_message)
        print(result)

if __name__ == "__main__":
    asyncio.run(main())
