#!/usr/bin/env python3
"""
Shell MCP Server - Provides shell command execution and filesystem tools.
Runs inside the container and exposes tools via Model Context Protocol.
"""

import asyncio
import os
import subprocess
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent

server = Server("shell-server")


@server.list_tools()
async def list_tools():
    return [
        Tool(
            name="run_command",
            description="Execute a shell command and return the output. Use for running CLI tools like --help, list, get, etc.",
            inputSchema={
                "type": "object",
                "properties": {
                    "command": {
                        "type": "string",
                        "description": "The shell command to execute",
                    }
                },
                "required": ["command"],
            },
        ),
        Tool(
            name="list_files",
            description="List files and directories in a given path.",
            inputSchema={
                "type": "object",
                "properties": {
                    "directory": {
                        "type": "string",
                        "description": "The directory path to list",
                    }
                },
                "required": ["directory"],
            },
        ),
        Tool(
            name="read_file",
            description="Read the contents of a file.",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {
                        "type": "string",
                        "description": "The file path to read",
                    }
                },
                "required": ["path"],
            },
        ),
        Tool(
            name="setup_environment",
            description="Run environment setup commands like installing packages. Has a longer timeout (120s).",
            inputSchema={
                "type": "object",
                "properties": {
                    "command": {
                        "type": "string",
                        "description": "The setup command to execute (e.g., pip install, npm install)",
                    }
                },
                "required": ["command"],
            },
        ),
    ]


@server.call_tool()
async def call_tool(name: str, arguments: dict):
    if name == "run_command":
        command = arguments.get("command", "")
        try:
            result = subprocess.run(
                ["sh", "-c", command],
                capture_output=True,
                text=True,
                timeout=30,
            )
            output = f"Exit Code: {result.returncode}\nOutput:\n{result.stdout}"
            if result.stderr:
                output += f"\nSTDERR:\n{result.stderr}"
            return [TextContent(type="text", text=output)]
        except subprocess.TimeoutExpired:
            return [TextContent(type="text", text="Error: Command timed out after 30 seconds")]
        except Exception as e:
            return [TextContent(type="text", text=f"Error executing command: {str(e)}")]

    elif name == "list_files":
        directory = arguments.get("directory", ".")
        try:
            if not os.path.exists(directory):
                return [TextContent(type="text", text=f"Error: Directory not found: {directory}")]
            files = os.listdir(directory)
            result = "\n".join(sorted(files))
            return [TextContent(type="text", text=result if result else "(empty directory)")]
        except Exception as e:
            return [TextContent(type="text", text=f"Error listing directory: {str(e)}")]

    elif name == "read_file":
        path = arguments.get("path", "")
        try:
            if not os.path.exists(path):
                return [TextContent(type="text", text=f"Error: File not found: {path}")]
            with open(path, "r") as f:
                content = f.read()
            return [TextContent(type="text", text=content)]
        except Exception as e:
            return [TextContent(type="text", text=f"Error reading file: {str(e)}")]

    elif name == "setup_environment":
        command = arguments.get("command", "")
        try:
            result = subprocess.run(
                ["sh", "-c", command],
                capture_output=True,
                text=True,
                timeout=120,
            )
            output = f"Exit Code: {result.returncode}\nOutput:\n{result.stdout}"
            if result.stderr:
                output += f"\nSTDERR:\n{result.stderr}"
            return [TextContent(type="text", text=output)]
        except subprocess.TimeoutExpired:
            return [TextContent(type="text", text="Error: Setup command timed out after 120 seconds")]
        except Exception as e:
            return [TextContent(type="text", text=f"Error executing setup: {str(e)}")]

    else:
        return [TextContent(type="text", text=f"Unknown tool: {name}")]


async def main():
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())


if __name__ == "__main__":
    asyncio.run(main())
