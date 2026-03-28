import { runCLI } from "../cli-bridge.js";
import { cancelTaskCrons } from "../cron.js";

export const taskFailTool = {
  name: "accountability_task_fail",
  description: `Mark an accountability task as failed and execute the punishment.

This sends the punishment message (from the secrets bank or custom message) via the configured action (e.g., WhatsApp, desktop notification).

WORKFLOW:
- This is called either manually by the user, or automatically when the deadline cron fires.
- When the deadline fires: the cron message instructs the agent to call this tool.
- After failure, use a growth-mindset message. Reference the task's 'why' — "You said this mattered because X. What will you do differently next time?"`,
  parameters: {
    type: "object" as const,
    properties: {
      task_id: { type: "string", description: "Task ID (e.g., task-1)" },
    },
    required: ["task_id"],
  },
  handler: async (params: { task_id: string }) => {
    const removed = await cancelTaskCrons(params.task_id);
    const result = await runCLI("task", "fail", [params.task_id]);
    return { ...result, crons_cancelled: removed };
  },
};
