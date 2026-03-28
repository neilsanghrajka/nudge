/**
 * CLI Bridge — spawns the Python CLI with --json and parses results.
 * Single integration point between the TypeScript plugin and the Python CLI.
 */

import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const execFileAsync = promisify(execFile);

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
// scripts/ is one level up from dist/
const CLI_PATH = join(__dirname, "..", "scripts", "accountability.py");

export interface CLIResult<T = unknown> {
  ok: boolean;
  command: string;
  data?: T;
  error?: string;
  code?: string;
}

/**
 * Run a CLI command and parse the JSON result.
 *
 * @param resource - Top-level resource (task, secrets, motivation, punishment, config, cleanup)
 * @param action - Action within resource (add, list, etc.)
 * @param args - Additional CLI arguments as flat array
 * @returns Parsed JSON envelope from CLI stdout
 */
export async function runCLI<T = unknown>(
  resource: string,
  action?: string,
  args: string[] = []
): Promise<CLIResult<T>> {
  const cliArgs = [CLI_PATH, "--json", resource];
  if (action) {
    cliArgs.push(action);
  }
  cliArgs.push(...args);

  try {
    const { stdout, stderr } = await execFileAsync("python3", cliArgs, {
      timeout: 30_000,
      maxBuffer: 1024 * 1024,
    });

    if (stderr) {
      // CLI warnings go to stderr; log but don't fail
      console.error(`[accountability-cli] ${stderr.trim()}`);
    }

    const result = JSON.parse(stdout.trim()) as CLIResult<T>;
    return result;
  } catch (err: unknown) {
    // Try to parse JSON from stdout even on non-zero exit
    const execErr = err as { stdout?: string; stderr?: string; code?: number };
    if (execErr.stdout) {
      try {
        return JSON.parse(execErr.stdout.trim()) as CLIResult<T>;
      } catch {
        // Not JSON, fall through
      }
    }

    return {
      ok: false,
      command: `${resource} ${action ?? ""}`.trim(),
      error: execErr.stderr?.trim() || String(err),
      code: "CLI_ERROR",
    };
  }
}
