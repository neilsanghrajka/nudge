import { runCLI } from "../cli-bridge.js";

export const motivationListTool = {
  name: "accountability_motivation_list",
  description: `List motivational quotes. Optionally filter by phase (when the quote is most effective).

Phases:
- task_created: starting energy
- reminder_early: value/purpose focus
- reminder_mid: discipline/persistence focus
- reminder_late: urgency/loss-aversion focus
- task_completed: celebration/identity reinforcement
- task_failed: growth mindset, try again`,
  parameters: {
    type: "object" as const,
    properties: {
      phase: { type: "string", description: "Filter by phase" },
    },
  },
  handler: async (params: { phase?: string }) => {
    const args = params.phase ? ["--phase", params.phase] : [];
    return runCLI("motivation", "list", args);
  },
};

export const motivationAddTool = {
  name: "accountability_motivation_add",
  description: `Add a custom motivational quote that resonates with the user. Ask what quotes or sayings personally motivate them.`,
  parameters: {
    type: "object" as const,
    properties: {
      quote: { type: "string", description: "The quote text" },
      attribution: { type: "string", description: "Who said it" },
      phase: { type: "string", description: "Comma-separated phases: task_created,reminder_early,reminder_mid,reminder_late,task_completed,task_failed" },
    },
    required: ["quote"],
  },
  handler: async (params: { quote: string; attribution?: string; phase?: string }) => {
    const args = ["--quote", params.quote];
    if (params.attribution) args.push("--attribution", params.attribution);
    if (params.phase) args.push("--phase", params.phase);
    return runCLI("motivation", "add", args);
  },
};
