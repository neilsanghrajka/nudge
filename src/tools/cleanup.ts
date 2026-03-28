import { runCLI } from "../cli-bridge.js";

export const cleanupTool = {
  name: "accountability_cleanup",
  description: `Cancel all active tasks and clean up temp files. This is a nuclear option — use sparingly. Requires explicit confirmation from the user.`,
  parameters: { type: "object" as const, properties: {} },
  handler: async () => runCLI("cleanup", undefined, ["--yes"]),
};
