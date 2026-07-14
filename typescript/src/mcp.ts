/**
 * L4 MCP self-registration — a service upserts its own registry row on deploy.
 * Credentials passed by reference (auth.secret_ref), never inline (C3).
 */

import { getConfig } from "./config";
import { authHeaders, postJson } from "./transport";

export interface McpRegistration {
  serverId: string;
  owner: string;
  displayName?: string;
  url: string;
  transport?: string;
  healthUrl?: string;
  authSecretRef?: string;
  authType?: string;
  scopes?: string[];
  toolSummary?: { name: string; description?: string }[];
  version?: string;
  enabled?: boolean;
}

export async function registerMcpServer(reg: McpRegistration): Promise<boolean> {
  const cfg = getConfig();
  if (!cfg.mcpRegistryUrl) {
    console.info(`[streamoid:mcp] registry URL unset; skipping ${reg.serverId}`);
    return false;
  }
  const resp = await postJson(
    `${cfg.mcpRegistryUrl}/api/v1/mcp/servers`,
    {
      server_id: reg.serverId,
      owner: reg.owner,
      display_name: reg.displayName,
      url: reg.url,
      transport: reg.transport ?? "streamable-http",
      health_url: reg.healthUrl,
      auth: { type: reg.authType ?? "bearer", secret_ref: reg.authSecretRef },
      scopes: reg.scopes ?? ["workspace"],
      tool_summary: reg.toolSummary ?? [],
      version: reg.version,
      enabled: reg.enabled ?? true,
    },
    { headers: authHeaders(cfg.platformToken), timeoutMs: cfg.timeoutMs, label: "mcp-registry" },
  );
  return resp !== null;
}
