# Domain Docs

Monthly Goal Tracker is a single-context repository. Engineering skills should
use the local `.ai` documents as the durable product and architecture context
before proposing implementation work.

## Required Context

Read these files when the task touches product behavior, architecture,
operations, authentication, data isolation, deployment, or serverless migration:

- `.ai/core/PRD.md`
- `.ai/core/ARCHITECTURE.md`
- `.ai/core/ADR.md`
- `.ai/execution/current.md`, when present

For the current serverless migration work, also read:

- `.ai/docs/SERVERLESS_LAMBDA_POSTGRES_PLAN.md`
- `.ai/phases/serverless-lambda-postgres-migration/index.json`
- `.ai/phases/serverless-lambda-postgres-migration/step6.md`

## Vocabulary

Use the terms from the `.ai/core` documents when writing issues, PRDs, briefs,
or implementation notes. Important project terms include:

- monthly goal
- daily check
- memo
- account
- authenticated user
- anonymous preview mode
- HttpOnly session cookie
- CSRF token
- serverless staging gate
- production public exposure

Do not rename these concepts casually in issue titles or implementation plans.
If a new term is needed, define it in the relevant `.ai/core` or phase document
before relying on it across issues.

## Conflict Handling

If a proposed change conflicts with `.ai/core/ADR.md` or
`.ai/core/ARCHITECTURE.md`, surface the conflict in the issue or brief and stop
before implementation unless the user explicitly approves the architecture,
dependency, schema, security policy, or public API contract change.
