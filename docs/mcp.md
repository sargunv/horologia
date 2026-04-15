# MCP

The server exposes an MCP endpoint at `/mcp`. MCP clients that support OAuth discovery will
authenticate automatically:

```json
{
  "mcpServers": {
    "horologia": {
      "type": "streamable-http",
      "url": "https://horologia.example.com/mcp"
    }
  }
}
```

For clients without OAuth support, pass an API token directly:

```json
{
  "mcpServers": {
    "horologia": {
      "type": "streamable-http",
      "url": "https://horologia.example.com/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_TOKEN"
      }
    }
  }
}
```
