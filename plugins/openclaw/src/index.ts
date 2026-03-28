/**
 * Stake — OpenClaw Plugin
 * Thin wrapper around the `stake` CLI binary.
 */

import { runCLI } from "./cli-bridge.js";
import { scheduleTaskCrons, cancelTaskCrons } from "./cron.js";

export function register(api: any) {
  // ─── Task tools ───

  api.registerTool({
    name: "stake_task_add",
    description: `Create an accountability task. BEFORE calling: 1) Ask the user why this matters 2) List secrets (stake_secrets_list) and suggest one 3) Confirm they're ready. The 'why' field drives all motivational reminders.`,
    parameters: {
      type: "object",
      properties: {
        desc: { type: "string", description: "Task description" },
        duration: { type: "number", description: "Minutes until deadline" },
        why: { type: "string", description: "Why this matters" },
        punishment: { type: "string", description: "Action name (e.g., post_to_beeper_whatsapp)" },
        targets: { type: "string", description: "Comma-separated recipient IDs" },
        secret_id: { type: "string", description: "Secret ID from bank" },
        custom_punishment_message: { type: "string" },
      },
      required: ["desc", "duration"],
    },
    handler: async (p: any) => {
      const args = ["task", "add", "--desc", p.desc, "--duration", String(p.duration)];
      if (p.why) args.push("--why", p.why);
      if (p.punishment) args.push("--punishment", p.punishment);
      if (p.targets) args.push("--targets", p.targets);
      if (p.secret_id) args.push("--secret-id", p.secret_id);
      if (p.custom_punishment_message) args.push("--custom-punishment-message", p.custom_punishment_message);
      const result = await runCLI(args);
      if (result.ok && result.data) {
        try {
          const crons = await scheduleTaskCrons(result.data as any);
          return { ...result, crons_scheduled: crons };
        } catch (e) { return { ...result, cron_warning: String(e) }; }
      }
      return result;
    },
  });

  api.registerTool({
    name: "stake_task_complete",
    description: `Mark task done. BEFORE calling: ask for proof, verify critically (see verification skill). After: celebrate, reference their 'why'.`,
    parameters: { type: "object", properties: { task_id: { type: "string" } }, required: ["task_id"] },
    handler: async (p: any) => {
      const removed = await cancelTaskCrons(p.task_id);
      const result = await runCLI(["task", "complete", p.task_id]);
      return { ...result, crons_cancelled: removed };
    },
  });

  api.registerTool({
    name: "stake_task_fail",
    description: `Mark task failed, execute punishment. After: growth mindset message, reference 'why', ask what they'll do differently.`,
    parameters: { type: "object", properties: { task_id: { type: "string" } }, required: ["task_id"] },
    handler: async (p: any) => {
      const removed = await cancelTaskCrons(p.task_id);
      const result = await runCLI(["task", "fail", p.task_id]);
      return { ...result, crons_cancelled: removed };
    },
  });

  api.registerTool({
    name: "stake_task_cancel",
    description: `Cancel task without punishment. Push back if user is avoiding — only allow for legitimate reasons (see strictness skill).`,
    parameters: { type: "object", properties: { task_id: { type: "string" } }, required: ["task_id"] },
    handler: async (p: any) => {
      const removed = await cancelTaskCrons(p.task_id);
      return { ...(await runCLI(["task", "cancel", p.task_id])), crons_cancelled: removed };
    },
  });

  api.registerTool({
    name: "stake_task_status",
    description: `Show active tasks with time remaining. Include 'why' in your response to reinforce motivation.`,
    parameters: { type: "object", properties: { task_id: { type: "string" } } },
    handler: async (p: any) => runCLI(p.task_id ? ["task", "status", p.task_id] : ["task", "status"]),
  });

  api.registerTool({
    name: "stake_task_list",
    description: `List tasks. Use --all to include history.`,
    parameters: { type: "object", properties: { all: { type: "boolean" } } },
    handler: async (p: any) => runCLI(p.all ? ["task", "list", "--all"] : ["task", "list"]),
  });

  api.registerTool({
    name: "stake_task_history",
    description: `Show task history. Use to reference track record for motivation.`,
    parameters: { type: "object", properties: { limit: { type: "number" } } },
    handler: async (p: any) => runCLI(p.limit ? ["task", "history", "--limit", String(p.limit)] : ["task", "history"]),
  });

  // ─── Secrets tools ───

  api.registerTool({
    name: "stake_secrets_add",
    description: `Add an embarrassing secret. Periodically prompt users to grow their bank.`,
    parameters: {
      type: "object",
      properties: { secret: { type: "string" }, severity: { type: "string", enum: ["mild", "medium", "spicy"] } },
      required: ["secret"],
    },
    handler: async (p: any) => {
      const args = ["secrets", "add", "--secret", p.secret];
      if (p.severity) args.push("--severity", p.severity);
      return runCLI(args);
    },
  });

  api.registerTool({
    name: "stake_secrets_list",
    description: `List all secrets. Use before task creation to suggest which one to stake.`,
    parameters: { type: "object", properties: {} },
    handler: async () => runCLI(["secrets", "list"]),
  });

  api.registerTool({
    name: "stake_secrets_pick",
    description: `Pick a secret (prefers least-used).`,
    parameters: { type: "object", properties: { severity: { type: "string", enum: ["mild", "medium", "spicy"] } } },
    handler: async (p: any) => runCLI(p.severity ? ["secrets", "pick", "--severity", p.severity] : ["secrets", "pick"]),
  });

  // ─── Motivation tools ───

  api.registerTool({
    name: "stake_motivation_list",
    description: `List motivational quotes, optionally by phase.`,
    parameters: { type: "object", properties: { phase: { type: "string" } } },
    handler: async (p: any) => runCLI(p.phase ? ["motivation", "list", "--phase", p.phase] : ["motivation", "list"]),
  });

  api.registerTool({
    name: "stake_motivation_add",
    description: `Add a custom quote the user finds personally motivating.`,
    parameters: {
      type: "object",
      properties: { quote: { type: "string" }, attribution: { type: "string" }, phase: { type: "string" } },
      required: ["quote"],
    },
    handler: async (p: any) => {
      const args = ["motivation", "add", "--quote", p.quote];
      if (p.attribution) args.push("--attribution", p.attribution);
      if (p.phase) args.push("--phase", p.phase);
      return runCLI(args);
    },
  });

  // ─── Punishment tools ───

  api.registerTool({
    name: "stake_punishment_list",
    description: `List available punishment actions and config status.`,
    parameters: { type: "object", properties: {} },
    handler: async () => runCLI(["punishment", "list"]),
  });

  api.registerTool({
    name: "stake_punishment_setup",
    description: `Configure a punishment action. After setup, run stake_punishment_health to verify.`,
    parameters: {
      type: "object",
      properties: {
        action_name: { type: "string" }, token: { type: "string" },
        beeper_url: { type: "string" }, default_group: { type: "string" },
        add_contact: { type: "string" },
      },
      required: ["action_name"],
    },
    handler: async (p: any) => {
      const args = ["punishment", "setup", p.action_name];
      if (p.token) args.push("--token", p.token);
      if (p.beeper_url) args.push("--beeper-url", p.beeper_url);
      if (p.default_group) args.push("--default-group", p.default_group);
      if (p.add_contact) args.push("--add-contact", p.add_contact);
      return runCLI(args);
    },
  });

  api.registerTool({
    name: "stake_punishment_health",
    description: `Test connectivity for a punishment action.`,
    parameters: { type: "object", properties: { action_name: { type: "string" } }, required: ["action_name"] },
    handler: async (p: any) => runCLI(["punishment", "health", p.action_name]),
  });

  // ─── Cleanup ───

  api.registerTool({
    name: "stake_cleanup",
    description: `Cancel all active tasks. Nuclear option — confirm with user first.`,
    parameters: { type: "object", properties: {} },
    handler: async () => runCLI(["cleanup", "--yes"]),
  });
}
