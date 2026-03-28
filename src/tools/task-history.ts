import { runCLI } from "../cli-bridge.js";

export const taskHistoryTool = {
  name: "accountability_task_history",
  description: `Show accountability task history. Use to review the user's track record. Highlight their completion rate and use it to build competence ("You've completed 7/8 tasks this week").`,
  parameters: {
    type: "object" as const,
    properties: {
      limit: { type: "number", description: "Max results to return (default: 10)" },
    },
  },
  handler: async (params: { limit?: number }) => {
    const args = params.limit ? ["--limit", String(params.limit)] : [];
    return runCLI("task", "history", args);
  },
};
