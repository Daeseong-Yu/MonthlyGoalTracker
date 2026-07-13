import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const prepare = workflow("serverless-production-prepare.yml");
const migrate = workflow("serverless-production-migrate.yml");
const publicSmoke = workflow("serverless-production-public-smoke.yml");
const localVerification = readFileSync(
  new URL("../../scripts/verify-serverless-local.sh", import.meta.url),
  "utf8",
);

assertManualOnly(prepare, "production prepare");
assertManualOnly(migrate, "production migration");
assertManualOnly(publicSmoke, "production public smoke");

assert.match(prepare, /^\s+environment: serverless-production-prepare$/m);
assert.match(prepare, /SERVERLESS_STAGE: production/);
assert.match(prepare, /type: choice/);
assert.match(prepare, /- infra-only/);
assert.match(prepare, /- full/);
assert.match(prepare, /scripts\/check-serverless-aws-readiness\.sh --require-ses-production-access/);
assert.match(prepare, /npx cdk deploy "MonthlyGoalTracker-\$\{SERVERLESS_STAGE\}"/);
assert.match(prepare, /bash \.\.\/scripts\/classify-serverless-cdk-failure\.sh "\$\{deploy_log\}"/);
assert.match(prepare, /--query 'Stacks\[0\]\.StackStatus'/);
assert.doesNotMatch(prepare, /cat "\$\{deploy_log\}"/);
assert.match(prepare, /github\.event\.inputs\.mode == 'full'/);
assert.doesNotMatch(prepare, /Run migration Lambda/);
assert.doesNotMatch(prepare, /Run migration audit Lambda/);
assert.doesNotMatch(prepare, /change-resource-record-sets/);

assert.match(migrate, /^\s+environment: serverless-production-migrate$/m);
assert.match(migrate, /confirmation/);
assert.match(migrate, /PRODUCTION_MIGRATION_CONFIRMATION: \$\{\{ github\.event\.inputs\.confirmation \}\}/);
assert.doesNotMatch(migrate, /if \[\[ "\$\{\{ github\.event\.inputs\.confirmation/);
assert.match(migrate, /aws lambda invoke/);
const migrationPreflight = migrate.indexOf("Preflight migration permissions");
const migrationExecution = migrate.indexOf("Run migration Lambda");
assert.notEqual(migrationPreflight, -1);
assert.notEqual(migrationExecution, -1);
assert.ok(migrationPreflight < migrationExecution, "permission preflight must run before migration");
assert.match(migrate, /--invocation-type DryRun/);
assert.match(migrate, /for function_name in "\$\{migration_function\}" "\$\{audit_function\}"/);
assert.match(migrate, /Production Lambda invoke permission preflight failed/);
assert.doesNotMatch(migrate, /npx cdk deploy/);
assert.doesNotMatch(migrate, /aws s3 /);
assert.doesNotMatch(migrate, /create-invalidation/);
assert.doesNotMatch(migrate, /change-resource-record-sets/);

assert.match(publicSmoke, /^\s+environment: serverless-production-public$/m);
assert.match(publicSmoke, /pnpm exec playwright test --grep "serverless"/);
assert.doesNotMatch(publicSmoke, /id-token: write/);
assert.doesNotMatch(publicSmoke, /configure-aws-credentials/);
assert.doesNotMatch(publicSmoke, /\baws\s/);
assert.doesNotMatch(publicSmoke, /npx cdk deploy/);
assert.doesNotMatch(
  localVerification,
  /\.ai\//,
  "tracked deployment verification must not depend on local workflow state",
);

console.log("ok - production workflows preserve independent approval and credential boundaries");

function workflow(name: string): string {
  return readFileSync(new URL(`../../.github/workflows/${name}`, import.meta.url), "utf8");
}

function assertManualOnly(source: string, label: string): void {
  const trigger = readTopLevelBlock(source, "on");
  assert.match(trigger, /^  workflow_dispatch:/m, `${label} must be manually dispatched`);
  assert.doesNotMatch(trigger, /^  push:/m, `${label} must not run on push`);
  assert.doesNotMatch(trigger, /^  pull_request:/m, `${label} must not run on pull request`);
  assert.doesNotMatch(trigger, /^  schedule:/m, `${label} must not run on a schedule`);
}

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
