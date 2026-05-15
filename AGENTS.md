# Project: Monthly Goal Tracker

이 저장소는 Monthly Goal Tracker의 애플리케이션 코드 source of truth다.
AI-driven workflow 상태는 이 working tree의 local `.ai/` 디렉터리를 기준으로 사용한다.

## 저장소

- Repository path: `.`
- Local product docs: `.ai/core/PRD.md`, `.ai/core/ARCHITECTURE.md`, `.ai/core/ADR.md`
- Local operational reference docs: `.ai/docs/*`
- Local phase registry: `.ai/phases/index.json`
- Local phase plans: `.ai/phases/{task}/*`
- Local current task, if present: `.ai/execution/current.md`
- Historical design/log workspace: `../../codex/MonthlyGoalTracker` 읽기 전용 참고
- Workflow template workspace: `../workflow` 읽기 전용 참고

## Stack

- Application: Monthly goal, daily check, memo, account, authentication service
- Frontend: React, TypeScript, Vite
- Backend: Go, Gin, GORM
- Data: PostgreSQL
- Hosting: EC2, Caddy, systemd, Docker Compose
- Automation: GitHub Actions
- Observability: health check, deployment logs, smoke test checklist

## 중요 아키텍처 규칙

- 이 저장소 root를 working directory로 사용한다.
- `.ai/`는 local workflow state이며 public repository에 없을 수 있다. 존재하는 경우 현재 작업 기준으로 사용한다.
- 구현 변경은 이 저장소에서 수행한다. 문서 저장소와 템플릿 저장소는 사용자가 명시적으로 요청하지 않으면 수정하지 않는다.
- 명시적 승인 없이 architecture, dependency, database schema breaking change, infra/security policy, public API contract를 변경하지 않는다.
- 명시적 승인 없이 commit, push, merge, deploy를 수행하지 않는다.
- Public endpoint는 `.ai/core/ARCHITECTURE.md`에 문서화된 API path로 제한한다.
- Backend는 인증, 세션, 사용자별 데이터 격리, DB transaction을 책임진다.
- Frontend는 API client와 화면 상태를 분리하고, protected write API를 우회하지 않는다.
- 비로그인 preview mode는 frontend-only 상태다. 서버에 저장하지 않고 로그인 사용자 데이터와 자동 병합하지 않는다.
- 모든 목표, 메모, 체크 데이터 접근은 인증된 user ID 범위로 제한한다.
- Session token은 HttpOnly cookie로 전달하고, unsafe HTTP method는 CSRF token을 요구한다.
- Token 원문, cookie, credential, private key, secret, 계정 식별자, 인프라 식별자, command ID를 출력하거나 문서화하지 않는다.

## 작업 프로세스

Non-trivial implementation 전에는 다음을 수행한다.

1. `.ai/core/PRD.md`를 읽는다.
2. `.ai/core/ARCHITECTURE.md`를 읽는다.
3. `.ai/core/ADR.md`를 읽는다.
4. `.ai/execution/current.md`가 있으면 확인한다.
5. 없으면 `.ai/phases/index.json`, 관련 task index, 관련 phase document를 따른다.

현재 작업 우선순위:

1. 사용자의 최신 지시
2. `.ai/execution/current.md`
3. `.ai/phases/index.json`
4. 관련 `.ai/phases/{task}/index.json`
5. 관련 phase document
6. `.ai/core/ADR.md`
7. `.ai/core/ARCHITECTURE.md`
8. `.ai/core/PRD.md`

명시적으로 요청받지 않은 archived 또는 completed phase file을 현재 지시로 취급하지 않는다.

## 구현 규칙

- 기존 코드 패턴과 계층 경계를 우선한다.
- Backend package 책임은 `.ai/core/ARCHITECTURE.md`의 `cmd/api`, `internal/config`, `internal/db`, `internal/domain`, `internal/repository`, `internal/service`, `internal/handler`, `internal/router`, `internal/principal` 경계를 따른다.
- Frontend는 bootstrap, session, CSRF, locale, preview state 흐름을 보존한다.
- 사용자 데이터 격리나 인증 흐름을 건드리는 변경은 테스트를 함께 추가하거나 갱신한다.
- 운영 환경 파일은 key 존재 여부만 확인하고 값 전체를 출력하지 않는다.
- 문서는 한국어를 기본으로 작성하되 code name, API path, command, env var는 영어 원문을 유지한다.

## 보안 규칙

- Secret을 절대 출력, 기록, commit, 노출하지 않는다.
- Secret을 generated static file, client bundle, public JSON, log, example에 두지 않는다.
- 계정 존재 여부 노출 완화, rate limit, token hash 저장 정책을 약화하지 않는다.
- Production API는 loopback bind와 Caddy `/api/*` reverse proxy 경계를 유지한다.
- Production cookie는 secure 설정을 사용한다.
- 커밋 메시지 제안만 요청받은 경우에는 변경 파일과 diff만 확인하고 제목과 짧은 본문만 제안한다. 이때 pre-commit security review나 전체 검증은 실행하지 않고, 커밋 가능 판단도 하지 않는다.
- 실제 커밋 요청 또는 "커밋 준비 완료" 판단 전에는 pre-commit security review를 수행한다. 커밋 요청 시 reviewer subagent를 사용하고, blocking finding이 있으면 커밋하지 않는다.

## 검증

Workflow structure:

```bash
python3 .ai/scripts/validate_workflow.py
python3 .ai/scripts/execute.py production-deploy-recovery --check
```

Backend:

```bash
cd backend
go test ./...
```

Frontend:

```bash
cd frontend
pnpm test
pnpm build
```

Repository hygiene:

```bash
git diff --check
```

배포 검증은 `.ai/docs/PRODUCTION_DEPLOYMENT_RUNBOOK.md`와 `.ai/docs/SMOKE_TEST_CHECKLIST.md`를 따른다.

## Notes

- 이 파일은 stable project rule에 집중해서 유지한다.
- Temporary 또는 current implementation detail은 `.ai/execution/current.md`에 둔다.
- Durable project context는 `.ai/core`에 둔다.
- Detailed phase sequencing은 `.ai/phases/{task}/index.json`에 둔다.
