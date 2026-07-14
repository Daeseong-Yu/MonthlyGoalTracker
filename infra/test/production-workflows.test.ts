import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const prepare = workflow("serverless-production-prepare.yml");
const frontendDeploy = workflow("serverless-production-frontend.yml");
const migrate = workflow("serverless-production-migrate.yml");
const publicSmoke = workflow("serverless-production-public-smoke.yml");
const localVerification = readFileSync(
  new URL("../../scripts/verify-serverless-local.sh", import.meta.url),
  "utf8",
);

assertManualOnly(prepare, "production prepare");
assertManualOnly(migrate, "production migration");
assertManualOnly(publicSmoke, "production public smoke");
assertAutomaticFrontendDeploy(frontendDeploy);

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
assert.match(frontendDeploy, /^\s+environment: serverless-production-prepare$/m);
assert.match(frontendDeploy, /permissions:\n\s+contents: read\n\s+id-token: write/);
assert.match(frontendDeploy, /pnpm test/);
assert.match(frontendDeploy, /pnpm build/);
assert.match(frontendDeploy, /scripts\/check-frontend-dist\.sh/);
assert.match(frontendDeploy, /e2e\/smoke-preview\.spec\.ts/);
assert.match(frontendDeploy, /aws s3 sync frontend\/dist/);
assert.match(frontendDeploy, /aws s3 cp frontend\/dist\/index\.html/);
assert.match(frontendDeploy, /aws cloudfront create-invalidation/);
assert.match(frontendDeploy, /aws cloudfront wait invalidation-completed/);
assert.match(frontendDeploy, /e2e\/serverless-smoke\.spec\.ts/);
assert.match(frontendDeploy, /SERVERLESS_SITE_BASE_URL/);
assert.match(frontendDeploy, /::add-mask::/);
assert.doesNotMatch(frontendDeploy, /npx cdk deploy/);
assert.doesNotMatch(frontendDeploy, /aws lambda invoke/);
assert.doesNotMatch(frontendDeploy, /change-resource-record-sets/);
assert.doesNotMatch(frontendDeploy, /SERVERLESS_SMOKE_EMAIL/);
assert.doesNotMatch(frontendDeploy, /SERVERLESS_SMOKE_PASSWORD/);
const frontendBuildJob = readJobBlock(frontendDeploy, "build");
const frontendDeployJob = readJobBlock(frontendDeploy, "deploy");
const frontendPublicSmokeJob = readJobBlock(frontendDeploy, "public-smoke");
assert.doesNotMatch(frontendBuildJob, /id-token: write/);
assert.doesNotMatch(frontendBuildJob, /configure-aws-credentials/);
assert.doesNotMatch(frontendBuildJob, /\baws\s/);
assert.match(frontendDeployJob, /actions\/download-artifact/);
assert.match(frontendDeployJob, /id-token: write/);
assert.doesNotMatch(frontendDeployJob, /actions\/checkout/);
assert.doesNotMatch(frontendDeployJob, /\b(?:pnpm|npm|node)\b/);
assert.doesNotMatch(frontendPublicSmokeJob, /id-token: write/);
assert.doesNotMatch(frontendPublicSmokeJob, /configure-aws-credentials/);
assert.doesNotMatch(frontendPublicSmokeJob, /\baws\s/);
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

function assertAutomaticFrontendDeploy(source: string): void {
  const trigger = readTopLevelBlock(source, "on");
  assert.match(trigger, /^  push:/m, "frontend deploy must run on push");
  assert.match(trigger, /^      - main$/m, "frontend deploy must target main only");
  assert.match(trigger, /^      - frontend\/\*\*$/m);
  assert.match(
    trigger,
    /^      - \.github\/workflows\/serverless-production-frontend\.yml$/m,
  );
  assert.match(trigger, /^      - scripts\/check-frontend-dist\.sh$/m);
  assert.match(trigger, /^  workflow_dispatch:/m, "frontend deploy must support manual retry");
  assert.doesNotMatch(trigger, /^  pull_request:/m);
  assert.doesNotMatch(trigger, /^  schedule:/m);
  assert.doesNotMatch(trigger, /^      - (?:backend|infra)\/\*\*$/m);
}

function readJobBlock(source: string, name: string): string {
  const lines = source.split(/\r?\n/);
  const start = lines.findIndex((line) => line === `  ${name}:`);
  assert.notEqual(start, -1, `missing ${name} job`);

  const block: string[] = [];
  for (const line of lines.slice(start + 1)) {
    if (/^  [a-zA-Z0-9_-]+:$/.test(line)) {
      break;
    }
    block.push(line);
  }
  return block.join("\n");
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
