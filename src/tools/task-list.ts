import { runCLI } from "../cli-bridge.js";

export const taskListTool = {
  name: "accountability_task_list",
  description: `List accountability tasks. Shows active tasks by default. Use --all to include completed/failed/cancelled history.`,
  parameters: {
    type: "object" as const,
    properties: {
      all: { type: "boolean", description: "Include history (completed/failed/cancelled)" },
    },
  },
  handler: async (params: { all?: boolean }) => {
    const args = params.all ? ["--all"] : [];
    return runCLI("task", "list", args);
  },
};
