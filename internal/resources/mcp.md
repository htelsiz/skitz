# MCP

`npx @modelcontextprotocol/inspector` launch MCP inspector ^run
`npx @modelcontextprotocol/create-server {{name}}` scaffold new server ^run:name
`claude mcp list` list configured servers ^run
`claude mcp serve` start MCP server ^run
`curl -s http://localhost:8001/mcp/` check server health ^run
`docker run -p 8001:8001 {{image}}` run MCP server container ^run:image
