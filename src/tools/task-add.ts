import { runCLI } from "../cli-bridge.js";
import { scheduleTaskCrons } from "../cron.js";

export const taskAddTool = {
  name: "accountability_task_add",
  description: `Create a new accountability task with a deadline and real consequences.

WORKFLOW — before calling this tool:
1. Ask the user: "What do you need to get done, and how long do you have?"
2. Ask: "Why does this matter to you? What happens if you don't do it?" (the 'why' drives motivational reminders)
3. List saved secrets (accountability_secrets_list) and suggest one as the punishment
4. If no secrets exist, ask the user to share something embarrassing (accountability_secrets_add)
5. Confirm the user is ready: recap task, deadline, and consequence
6. Then call this tool

The 'why' field is critical — it's used in every reminder to keep the user motivated.`,
  parameters: {
    type: "object" as const,
    properties: {
      desc: { type: "string", description: "Task description" },
      duration: { type: "number", description: "Duration in minutes" },
      why: { type: "string", description: "Why this task matters to the user (used in motivational reminders)" },
      punishment: { type: "string", description: "Punishment action name (e.g., post_to_beeper_whatsapp). Omit for default." },
      targets: { type: "string", description: "Comma-separated recipient IDs for the punishment action" },
      secret_id: { type: "string", description: "ID of a secret from the secrets bank to use as punishment content" },
      custom_punishment_message: { type: "string", description: "Custom punishment message (if not using a secret)" },
    },
    required: ["desc", "duration"],
  },
  handler: async (params: {
    desc: string;
    duration: number;
    why?: string;
    punishment?: string;
    targets?: string;
    secret_id?: string;
    custom_punishment_message?: string;
  }) => {
    const args: string[] = ["--desc", params.desc, "--duration", String(params.duration)];
    if (params.why) args.push("--why", params.why);
    if (params.punishment) args.push("--punishment", params.punishment);
    if (params.targets) args.push("--targets", params.targets);
    if (params.secret_id) args.push("--secret-id", params.secret_id);
    if (params.custom_punishment_message) args.push("--custom-punishment-message", params.custom_punishment_message);

    const result = await runCLI("task", "add", args);

    if (result.ok && result.data) {
      // Schedule cron jobs for warnings and deadline
      const taskData = result.data as Record<string, unknown>;
      try {
        const cronNames = await scheduleTaskCrons(taskData as any);
        return { ...result, crons_scheduled: cronNames };
      } catch (err) {
        return { ...result, cron_warning: `Crons failed to schedule: ${err}` };
      }
    }

    return result;
  },
};
