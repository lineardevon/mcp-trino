# Programmatic Authentication Guide

This guide explains how to authenticate with mcp-trino programmatically using tokens obtained from mcp-remote's OAuth flow. This is useful for building automated agents, scripts, or applications that need to interact with the MCP server without interactive authentication.

## Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  1. OAuth Flow  │ ──► │ 2. Extract Token│ ──► │ 3. Use in Code  │
│  (mcp-remote)   │     │ (~/.mcp-auth/)  │     │ (Agent SDK)     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Prerequisites

- [mcp-remote](https://github.com/geelen/mcp-remote) installed
- Access to an mcp-trino server with OAuth enabled
- Node.js 18+ (for Claude Agent SDK)
- `jq` command-line tool (optional, for token extraction)

## Step 1: Authenticate with mcp-remote

### Using Claude Desktop or Cursor

Add the MCP server to your configuration:

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "trino": {
      "command": "npx",
      "args": ["mcp-remote", "https://your-mcp-server.example.com/mcp"]
    }
  }
}
```

When you first use the MCP server, a browser window will open for OAuth authentication.

### Direct Authentication

```bash
npx mcp-remote https://your-mcp-server.example.com/mcp
```

This opens a browser for OAuth login and stores tokens locally.

## Step 2: Extract the Token

### Token Storage Location

mcp-remote stores credentials in:

```
~/.mcp-auth/
└── mcp-remote-{VERSION}/
    ├── {server_hash}_client_info.json
    └── {server_hash}_tokens.json      # <-- Contains access token
```

### Token File Format

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "id_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3599,
  "scope": "openid profile email"
}
```

### Extract via Command Line

```bash
# Find and extract the access token
export MCP_TOKEN=$(jq -r '.access_token' ~/.mcp-auth/mcp-remote-*/*_tokens.json | head -1)
```

### Helper Script

```bash
#!/bin/bash
# get-mcp-token.sh

MCP_AUTH_DIR=$(ls -td ~/.mcp-auth/mcp-remote-* 2>/dev/null | head -1)
TOKEN_FILE=$(ls -t "$MCP_AUTH_DIR"/*_tokens.json 2>/dev/null | head -1)

if [ -z "$TOKEN_FILE" ]; then
    echo "Error: No token found. Run mcp-remote first." >&2
    exit 1
fi

jq -r '.access_token' "$TOKEN_FILE"
```

## Step 3: Use the Token

### Verify with curl

```bash
curl -H "Authorization: Bearer $MCP_TOKEN" \
  -H "Content-Type: application/json" \
  -X POST https://your-mcp-server.example.com/mcp \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}'
```

### Claude Agent SDK (TypeScript)

```typescript
import { query } from "@anthropic-ai/claude-agent-sdk";

for await (const message of query({
  prompt: "List all available catalogs",
  options: {
    mcpServers: {
      trino: {
        type: "http",
        url: "https://your-mcp-server.example.com/mcp",
        headers: {
          Authorization: `Bearer ${process.env.MCP_TOKEN}`,
        },
      },
    },
  },
})) {
  if (message.type === "result" && message.subtype === "success") {
    console.log(message.result);
  }
}
```

Run with:
```bash
export MCP_TOKEN=$(./get-mcp-token.sh)
npx ts-node your-script.ts
```

### Reading Token from File (TypeScript)

```typescript
import fs from "fs";
import path from "path";
import os from "os";

function getToken(): string {
  if (process.env.MCP_TOKEN) return process.env.MCP_TOKEN;

  const authDir = fs.readdirSync(path.join(os.homedir(), ".mcp-auth"))
    .filter(d => d.startsWith("mcp-remote-"))
    .sort()
    .pop();

  const tokenFile = fs.readdirSync(path.join(os.homedir(), ".mcp-auth", authDir!))
    .find(f => f.endsWith("_tokens.json"));

  const tokens = JSON.parse(
    fs.readFileSync(path.join(os.homedir(), ".mcp-auth", authDir!, tokenFile!), "utf-8")
  );

  return tokens.access_token;
}
```

### Python Example

```python
import json
from pathlib import Path
import httpx

def get_token() -> str:
    mcp_auth = Path.home() / ".mcp-auth"
    version_dir = sorted(mcp_auth.glob("mcp-remote-*"))[-1]
    token_file = next(version_dir.glob("*_tokens.json"))
    return json.loads(token_file.read_text())["access_token"]

# Use with MCP server
client = httpx.Client(headers={"Authorization": f"Bearer {get_token()}"})
response = client.post(
    "https://your-mcp-server.example.com/mcp",
    json={"jsonrpc": "2.0", "method": "tools/list", "id": 1}
)
```

## Token Expiration

OAuth tokens typically expire in **1 hour** (`expires_in: 3599` seconds).

**Options for handling expiration:**

1. **Re-authenticate manually**: Run `npx mcp-remote <server-url>` again
2. **Client Credentials flow**: For fully automated systems without user interaction, use OAuth Client Credentials (see [OAuth Documentation](oauth.md))
3. **Check expiration in code**: Decode the JWT and check the `exp` claim

> **Note:** mcp-remote tokens may not include a `refresh_token` depending on the OAuth provider configuration. If no refresh token is available, re-authentication is required when the access token expires.

## Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| No auth directory found | Never authenticated | Run `npx mcp-remote <url>` |
| 401 Unauthorized | Token expired | Re-authenticate with mcp-remote |
| Invalid session ID | Missing session init | SDK handles this; for raw HTTP, call `initialize` first |

## Security Notes

- Tokens are stored in plaintext in `~/.mcp-auth/`
- Set appropriate file permissions: `chmod 700 ~/.mcp-auth`
- Never commit tokens to git
- Tokens expire in ~1 hour; implement refresh for long-running apps
