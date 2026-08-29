import type { z } from "zod";

// Local mirrors of the OpenCode v1.18.11 plugin SDK contracts. These are type-only
// (erased at build time), so the self-contained bundle never imports
// @opencode-ai/plugin at runtime. The shapes match the authoritative SDK at
// packages/plugin/src/index.ts and tool.ts.

export type AskInput = {
  permission: string;
  patterns: string[];
  always: string[];
  metadata: { [key: string]: any };
};

export type ToolContext = {
  sessionID: string;
  messageID: string;
  agent: string;
  directory: string;
  worktree: string;
  abort: AbortSignal;
  metadata(input: { title?: string; metadata?: { [key: string]: any } }): void;
  ask(input: AskInput): Promise<void>;
};

export type ToolAttachment = {
  type: "file";
  mime: string;
  url: string;
  filename?: string;
};

export type ToolResult =
  | string
  | {
      title?: string;
      output: string;
      metadata?: { [key: string]: any };
      attachments?: ToolAttachment[];
    };

export type ToolDefinition = {
  description: string;
  args: z.ZodRawShape;
  execute(args: any, context: ToolContext): Promise<ToolResult>;
};

export type PluginInput = {
  client: unknown;
  project: unknown;
  directory: string;
  worktree: string;
  experimental_workspace: unknown;
  serverUrl: URL;
  $: unknown;
};

export type ChatUserMessage = {
  id?: string;
  sessionID?: string;
  role?: string;
  [key: string]: unknown;
};

export type ChatPart = {
  type?: string;
  synthetic?: boolean;
  metadata?: { [key: string]: unknown };
  [key: string]: unknown;
};

export type PluginOptions = Record<string, unknown>;

export type Hooks = {
  tool?: { [key: string]: ToolDefinition };
  "chat.message"?: (
    input: {
      sessionID: string;
      agent?: string;
      model?: { providerID: string; modelID: string };
      messageID?: string;
      variant?: string;
    },
    output: { message: ChatUserMessage; parts: ChatPart[] },
  ) => Promise<void>;
  "tool.execute.before"?: (
    input: { tool: string; sessionID: string; callID: string },
    output: { args: any },
  ) => Promise<void>;
  "tool.execute.after"?: (
    input: { tool: string; sessionID: string; callID: string; args: any },
    output: { title: string; output: string; metadata: any },
  ) => Promise<void>;
  dispose?: () => Promise<void>;
};

export type Plugin = (input: PluginInput, options?: PluginOptions) => Promise<Hooks>;

export type PluginModule = {
  id: string;
  server: Plugin;
};
