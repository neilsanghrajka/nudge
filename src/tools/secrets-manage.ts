import { runCLI } from "../cli-bridge.js";

export const secretsAddTool = {
  name: "accountability_secrets_add",
  description: `Add an embarrassing secret to the secrets bank. These get revealed as punishment when the user fails a task.

WORKFLOW:
- Periodically prompt the user: "What's something embarrassing you'd never want your friends to know?"
- The more personal and cringeworthy, the more motivating the threat
- Classify severity: mild (slightly embarrassing), medium (would rather not share), spicy (absolutely mortifying)`,
  parameters: {
    type: "object" as const,
    properties: {
      secret: { type: "string", description: "The embarrassing secret" },
      severity: { type: "string", enum: ["mild", "medium", "spicy"], description: "How embarrassing is it?" },
    },
    required: ["secret"],
  },
  handler: async (params: { secret: string; severity?: string }) => {
    const args = ["--secret", params.secret];
    if (params.severity) args.push("--severity", params.severity);
    return runCLI("secrets", "add", args);
  },
};

export const secretsListTool = {
  name: "accountability_secrets_list",
  description: `List all secrets in the bank. Use this before creating a task to suggest which secret to put on the line.`,
  parameters: { type: "object" as const, properties: {} },
  handler: async () => runCLI("secrets", "list"),
};

export const secretsPickTool = {
  name: "accountability_secrets_pick",
  description: `Pick a secret from the bank for use as punishment. Prefers least-used secrets to keep things fresh. Optionally filter by severity.`,
  parameters: {
    type: "object" as const,
    properties: {
      severity: { type: "string", enum: ["mild", "medium", "spicy"], description: "Filter by severity" },
    },
  },
  handler: async (params: { severity?: string }) => {
    const args = params.severity ? ["--severity", params.severity] : [];
    return runCLI("secrets", "pick", args);
  },
};
