import { runCLI } from "../cli-bridge.js";
import { cancelTaskCrons } from "../cron.js";

export const taskCompleteTool = {
  name: "accountability_task_complete",
  description: `Mark an accountability task as completed. Cancels all warning/punishment crons and sends all-clear messages.

WORKFLOW — before calling this tool:
1. Ask the user for proof of completion
2. Review the proof critically — don't let them off easy
3. If the proof is insufficient, tell them what's missing and DON'T call this tool
4. Once satisfied, call this tool with the task_id
5. After success, celebrate! Reference their 'why' and reinforce their identity as someone who follows through.`,
  parameters: {
    type: "object" as const,
    properties: {
      task_id: { type: "string", description: "Task ID (e.g., task-1)" },
    },
    required: ["task_id"],
  },
  handler: async (params: { task_id: string }) => {
    // Cancel crons first
    const removed = await cancelTaskCrons(params.task_id);

    // Mark complete in CLI
    const result = await runCLI("task", "complete", [params.task_id]);

    return { ...result, crons_cancelled: removed };
  },
};
