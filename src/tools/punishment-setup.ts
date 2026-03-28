import { runCLI } from "../cli-bridge.js";

export const punishmentListTool = {
  name: "accountability_punishment_list",
  description: `List available punishment actions and their configuration status. Shows which actions are set up and which need configuration.`,
  parameters: { type: "object" as const, properties: {} },
  handler: async () => runCLI("punishment", "list"),
};

export const punishmentSetupTool = {
  name: "accountability_punishment_setup",
  description: `Configure a punishment action. Each action requires different settings.

For post_to_beeper_whatsapp:
  --token: Beeper API token
  --beeper-url: Beeper API URL (default: http://localhost:23373)
  --default-group: Default WhatsApp group chat ID
  --add-contact: Add contact as "name=chat_id"

WORKFLOW:
- Guide the user through setup for their chosen action
- After setup, run accountability_punishment_health to verify connectivity`,
  parameters: {
    type: "object" as const,
    properties: {
      action_name: { type: "string", description: "Action to configure (e.g., post_to_beeper_whatsapp)" },
      token: { type: "string", description: "API token" },
      beeper_url: { type: "string", description: "Beeper API URL" },
      default_group: { type: "string", description: "Default group/chat ID" },
      add_contact: { type: "string", description: "Add contact as name=chat_id" },
    },
    required: ["action_name"],
  },
  handler: async (params: {
    action_name: string;
    token?: string;
    beeper_url?: string;
    default_group?: string;
    add_contact?: string;
  }) => {
    const args = [params.action_name];
    if (params.token) args.push("--token", params.token);
    if (params.beeper_url) args.push("--beeper-url", params.beeper_url);
    if (params.default_group) args.push("--default-group", params.default_group);
    if (params.add_contact) args.push("--add-contact", params.add_contact);
    return runCLI("punishment", "setup", args);
  },
};

export const punishmentHealthTool = {
  name: "accountability_punishment_health",
  description: `Test connectivity for a punishment action. Run this after setup to verify everything works.`,
  parameters: {
    type: "object" as const,
    properties: {
      action_name: { type: "string", description: "Action to test" },
    },
    required: ["action_name"],
  },
  handler: async (params: { action_name: string }) => {
    return runCLI("punishment", "health", [params.action_name]);
  },
};
