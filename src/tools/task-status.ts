import { runCLI } from "../cli-bridge.js";

export const taskStatusTool = {
  name: "accountability_task_status",
  description: `Show active accountability tasks with time remaining. Include the task's 'why' in your response to the user to reinforce motivation. If time is running low, increase urgency.`,
  parameters: {
    type: "object" as const,
    properties: {
      task_id: { type: "string", description: "Specific task ID (omit for all active tasks)" },
    },
  },
  handler: async (params: { task_id?: string }) => {
    const args = params.task_id ? [params.task_id] : [];
    return runCLI("task", "status", args);
  },
};
