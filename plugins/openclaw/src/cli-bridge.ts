/**
 * CLI Bridge — spawns the `stake` binary with --json and parses results.
 */

import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

export interface CLIResult<T = unknown> {
  ok: boolean;
  command: string;
  data?: T;
  error?: string;
  code?: string;
}

export async function runCLI<T = unknown>(
  args: string[]
): Promise<CLIResult<T>> {
  try {
    const { stdout, stderr } = await execFileAsync("stake", ["--json", ...args], {
      timeout: 30_000,
      maxBuffer: 1024 * 1024,
    });
    if (stderr) console.error(`[stake] ${stderr.trim()}`);
    return JSON.parse(stdout.trim()) as CLIResult<T>;
  } catch (err: unknown) {
    const execErr = err as { stdout?: string; stderr?: string };
    if (execErr.stdout) {
      try { return JSON.parse(execErr.stdout.trim()) as CLIResult<T>; } catch {}
    }
    return { ok: false, command: args.join(" "), error: String(err), code: "CLI_ERROR" };
  }
}
