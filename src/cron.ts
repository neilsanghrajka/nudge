/**
 * Cron scheduling for accountability task warnings and deadlines.
 *
 * Creates OpenClaw cron jobs for:
 * - Warning reminders at 50%, 75%, 10min, 5min (local notifications + motivational messages)
 * - Deadline punishment cron at 0min remaining
 *
 * The Python CLI returns warning_intervals in the task data — this module
 * converts those into OpenClaw cron jobs.
 */

import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

interface WarningInterval {
  name: string;
  minutes_from_start: number;
  minutes_remaining: number;
  phase: string;
}

interface TaskData {
  id: string;
  description: string;
  why?: string | null;
  duration_minutes: number;
  punishment_action: string;
  punishment_message?: string | null;
  targets?: string[];
  deadline: string;
  warning_intervals: WarningInterval[];
}

/**
 * Generate ISO 8601 timestamp for N minutes from now.
 */
function isoAt(minutesFromNow: number): string {
  const t = new Date(Date.now() + minutesFromNow * 60 * 1000);
  return t.toISOString().replace(/\.\d{3}Z$/, "Z");
}

/**
 * Add a single OpenClaw cron job.
 */
async function addCron(
  name: string,
  at: string,
  message: string
): Promise<string | null> {
  try {
    const { stdout } = await execFileAsync("openclaw", [
      "cron",
      "add",
      "--name",
      name,
      "--at",
      at,
      "--session",
      "isolated",
      "--message",
      message,
      "--wake",
      "now",
      "--delete-after-run",
    ]);
    return stdout.trim();
  } catch (err) {
    console.error(`[cron] Failed to add ${name}: ${err}`);
    return null;
  }
}

/**
 * Remove all cron jobs matching a prefix.
 */
async function removeCronsByPrefix(prefix: string): Promise<number> {
  let removed = 0;
  try {
    const { stdout } = await execFileAsync("openclaw", [
      "cron",
      "list",
      "--json",
    ]);
    const data = JSON.parse(stdout);
    const jobs = Array.isArray(data)
      ? data
      : data.jobs ?? data.items ?? [];

    for (const job of jobs) {
      const name = job.name ?? "";
      const id = job.jobId ?? job.id ?? "";
      if (name.toLowerCase().includes(prefix.toLowerCase())) {
        try {
          await execFileAsync("openclaw", ["cron", "remove", id]);
          removed++;
        } catch {
          // Best effort
        }
      }
    }
  } catch {
    // Fallback: try non-json list
    try {
      const { stdout } = await execFileAsync("openclaw", ["cron", "list"]);
      for (const line of stdout.split("\n")) {
        if (line.toLowerCase().includes(prefix.toLowerCase())) {
          const jobId = line.split(/\s+/)[0];
          if (jobId) {
            try {
              await execFileAsync("openclaw", ["cron", "remove", jobId]);
              removed++;
            } catch {
              // Best effort
            }
          }
        }
      }
    } catch {
      // Give up
    }
  }
  return removed;
}

/**
 * Build a motivational warning message for a cron reminder.
 */
function buildWarningMessage(
  task: TaskData,
  warning: WarningInterval
): string {
  const parts: string[] = [];

  // Time urgency
  parts.push(
    `Accountability reminder for '${task.description}': ${warning.minutes_remaining} minutes left.`
  );

  // Include the why
  if (task.why) {
    parts.push(`Remember why: "${task.why}"`);
  }

  // Phase-specific framing
  if (warning.phase === "reminder_late") {
    parts.push(
      "This is crunch time. Your punishment will fire if you don't finish."
    );
  }

  // Instruct the agent to deliver as notification
  parts.push(
    `Run: osascript -e 'display notification "${warning.minutes_remaining} min left: ${task.description.replace(/'/g, "")}" with title "Accountability Coach" sound name "Funk"'`
  );

  return parts.join(" ");
}

/**
 * Build the punishment cron message — tells the agent what to do when deadline hits.
 */
function buildPunishmentMessage(task: TaskData): string {
  const parts: string[] = [
    `ACCOUNTABILITY DEADLINE REACHED for '${task.description}'. User FAILED.`,
  ];

  if (task.why) {
    parts.push(`They said this mattered because: "${task.why}"`);
  }

  parts.push(
    `Execute punishment: run python3 scripts/accountability.py task fail ${task.id} --json`
  );

  if (task.punishment_message) {
    parts.push(`Punishment message: ${task.punishment_message}`);
  }

  parts.push("Also send a desktop notification about the failure.");

  return parts.join(" ");
}

/**
 * Schedule all cron jobs for a new task.
 * Returns the list of cron job names created.
 */
export async function scheduleTaskCrons(
  task: TaskData
): Promise<string[]> {
  const prefix = `acc-${task.id}`;
  const cronNames: string[] = [];

  // Warning crons (local notifications + motivational messages)
  for (const warning of task.warning_intervals) {
    const name = `${prefix}-${warning.name}`;
    const at = isoAt(warning.minutes_from_start);
    const message = buildWarningMessage(task, warning);
    const result = await addCron(name, at, message);
    if (result !== null) {
      cronNames.push(name);
    }
  }

  // Punishment cron at deadline
  const punishName = `${prefix}-punish`;
  const punishAt = isoAt(task.duration_minutes);
  const punishMessage = buildPunishmentMessage(task);
  const result = await addCron(punishName, punishAt, punishMessage);
  if (result !== null) {
    cronNames.push(punishName);
  }

  return cronNames;
}

/**
 * Cancel all cron jobs for a task.
 */
export async function cancelTaskCrons(taskId: string): Promise<number> {
  return removeCronsByPrefix(`acc-${taskId}`);
}
