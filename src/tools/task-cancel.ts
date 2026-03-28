import { runCLI } from "../cli-bridge.js";
import { cancelTaskCrons } from "../cron.js";

export const taskCancelTool = {
  name: "accountability_task_cancel",
  description: `Cancel an accountability task without sending any punishment. Use when the user has a legitimate reason to cancel (e.g., task is no longer relevant). Do NOT use this as an escape hatch to avoid consequences — push back if the user is just trying to dodge their commitment.`,
  parameters: {
    type: "object" as const,
    properties: {
      task_id: { type: "string", description: "Task ID (e.g., task-1)" },
    },
    required: ["task_id"],
  },
  handler: async (params: { task_id: string }) => {
    const removed = await cancelTaskCrons(params.task_id);
    const result = await runCLI("task", "cancel", [params.task_id]);
    return { ...result, crons_cancelled: removed };
  },
};
