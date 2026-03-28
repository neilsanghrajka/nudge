/**
 * Cron scheduling via OpenClaw for task warnings and deadlines.
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
  why?: string;
  duration_minutes: number;
  warning_intervals: WarningInterval[];
}

function isoAt(minutesFromNow: number): string {
  return new Date(Date.now() + minutesFromNow * 60_000).toISOString().replace(/\.\d{3}Z$/, "Z");
}

async function addCron(name: string, at: string, message: string): Promise<boolean> {
  try {
    await execFileAsync("openclaw", [
      "cron", "add", "--name", name, "--at", at,
      "--session", "isolated", "--message", message,
      "--wake", "now", "--delete-after-run",
    ]);
    return true;
  } catch { return false; }
}

async function removeCronsByPrefix(prefix: string): Promise<number> {
  let removed = 0;
  try {
    const { stdout } = await execFileAsync("openclaw", ["cron", "list", "--json"]);
    const data = JSON.parse(stdout);
    const jobs = Array.isArray(data) ? data : data.jobs ?? data.items ?? [];
    for (const job of jobs) {
      if ((job.name ?? "").toLowerCase().includes(prefix.toLowerCase())) {
        try {
          await execFileAsync("openclaw", ["cron", "remove", job.jobId ?? job.id]);
          removed++;
        } catch {}
      }
    }
  } catch {}
  return removed;
}

export async function scheduleTaskCrons(task: TaskData): Promise<string[]> {
  const prefix = `acc-${task.id}`;
  const names: string[] = [];

  for (const w of task.warning_intervals) {
    const name = `${prefix}-${w.name}`;
    const msg = `Accountability reminder for '${task.description}': ${w.minutes_remaining} min left.${task.why ? ` Why: "${task.why}"` : ""} Run: osascript -e 'display notification "${w.minutes_remaining} min left" with title "Stake" sound name "Funk"'`;
    if (await addCron(name, isoAt(w.minutes_from_start), msg)) names.push(name);
  }

  const punishMsg = `DEADLINE REACHED for '${task.description}'. Execute: stake task fail ${task.id}`;
  if (await addCron(`${prefix}-punish`, isoAt(task.duration_minutes), punishMsg)) names.push(`${prefix}-punish`);

  return names;
}

export async function cancelTaskCrons(taskId: string): Promise<number> {
  return removeCronsByPrefix(`acc-${taskId}`);
}
