export const VERIFICATION_CONTROL_KIND = "verification-control";

export type ChatMessageEpochInput = {
  sessionID: string;
  messageID?: string;
};

export type ChatMessageEpochOutput = {
  message?: { id?: string; sessionID?: string; role?: string; [key: string]: unknown };
  parts?: unknown[];
};

function stringField(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function controlRoot(parts: unknown[] | undefined): string {
  if (!Array.isArray(parts)) return "";
  for (const candidate of parts) {
    if (!candidate || typeof candidate !== "object") continue;
    const part = candidate as Record<string, unknown>;
    if (part.synthetic !== true) continue;
    const metadata = part.metadata;
    if (!metadata || typeof metadata !== "object") continue;
    const values = metadata as Record<string, unknown>;
    if (stringField(values["codea.kind"]) !== VERIFICATION_CONTROL_KIND) continue;
    const root = stringField(values["codea.rootTurn"]);
    if (root) return root;
  }
  return "";
}

export class RootTurnEpochs {
  private readonly currentBySession = new Map<string, string>();

  observe(input: ChatMessageEpochInput, output: ChatMessageEpochOutput): string {
    const sessionID = stringField(input.sessionID || output.message?.sessionID);
    const messageID = stringField(input.messageID || output.message?.id);
    if (!sessionID || !messageID) return "";

    const continuationRoot = controlRoot(output.parts);
    if (continuationRoot) {
      this.currentBySession.set(sessionID, continuationRoot);
      return continuationRoot;
    }

    this.currentBySession.set(sessionID, messageID);
    return messageID;
  }

  current(sessionID: string): string {
    return this.currentBySession.get(sessionID.trim()) ?? "";
  }

  clear(sessionID: string): void {
    this.currentBySession.delete(sessionID.trim());
  }
}
