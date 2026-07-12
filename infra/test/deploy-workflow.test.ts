import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const workflowUrl = new URL(
  "../../.github/workflows/deploy.yml",
  import.meta.url,
);
const workflowSource = readFileSync(workflowUrl, "utf8");
const triggerBlock = readTopLevelBlock(workflowSource, "on");

assert.match(
  triggerBlock,
  /^  workflow_dispatch:/m,
  "legacy production deploy must retain an explicit manual trigger",
);
assert.doesNotMatch(
  triggerBlock,
  /^  push:/m,
  "legacy production deploy must not run automatically after a main merge",
);

console.log("ok - legacy production deploy is manual-only");

function readTopLevelBlock(source: string, key: string): string {
  const lines = source.split(/\r?\n/);
  const start = lines.findIndex((line) => line === `${key}:`);

  assert.notEqual(start, -1, `missing top-level ${key} block`);

  const block: string[] = [];
  for (const line of lines.slice(start + 1)) {
    if (line !== "" && !/^\s/.test(line)) {
      break;
    }
    block.push(line);
  }

  return block.join("\n");
}
