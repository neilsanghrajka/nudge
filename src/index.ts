/**
 * Accountability Coach — OpenClaw Plugin
 *
 * Set deadlines with real social consequences. Features:
 * - Configurable punishment actions (WhatsApp via Beeper, desktop notifications, extensible)
 * - Secrets bank for embarrassing reveals
 * - Psychology-backed motivation engine with per-task "why"
 * - Timed warnings with motivational content via OpenClaw cron
 */

import { taskAddTool } from "./tools/task-add.js";
import { taskCompleteTool } from "./tools/task-complete.js";
import { taskFailTool } from "./tools/task-fail.js";
import { taskCancelTool } from "./tools/task-cancel.js";
import { taskStatusTool } from "./tools/task-status.js";
import { taskListTool } from "./tools/task-list.js";
import { taskHistoryTool } from "./tools/task-history.js";
import { secretsAddTool, secretsListTool, secretsPickTool } from "./tools/secrets-manage.js";
import { motivationListTool, motivationAddTool } from "./tools/motivation-manage.js";
import { punishmentListTool, punishmentSetupTool, punishmentHealthTool } from "./tools/punishment-setup.js";
import { cleanupTool } from "./tools/cleanup.js";

/**
 * Plugin entry point. Registers all accountability tools with OpenClaw.
 */
export function register(api: any) {
  const tools = [
    // Task lifecycle
    taskAddTool,
    taskCompleteTool,
    taskFailTool,
    taskCancelTool,
    taskStatusTool,
    taskListTool,
    taskHistoryTool,

    // Secrets bank
    secretsAddTool,
    secretsListTool,
    secretsPickTool,

    // Motivation
    motivationListTool,
    motivationAddTool,

    // Punishment configuration
    punishmentListTool,
    punishmentSetupTool,
    punishmentHealthTool,

    // Cleanup
    cleanupTool,
  ];

  for (const tool of tools) {
    api.registerTool(tool);
  }
}
