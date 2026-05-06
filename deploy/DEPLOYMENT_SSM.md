# SSM 배포 운영 가이드

이 문서는 Monthly Goal Tracker의 권장 프로덕션 배포 절차를 정리합니다.
기준 방식은 GitHub Actions OIDC, S3 artifact, AWS Systems Manager Run
Command입니다. GitHub Actions는 EC2 SSH key나 장기 AWS access key를 보관하지
않습니다.

실제 계정 ID, instance ID, bucket 이름, domain, IP 주소는 repository에
기록하지 않습니다. 문서와 예시는 placeholder만 사용합니다.

## 배포 구조

```text
developer
  -> develop push
  -> Backend CI / Frontend CI
  -> Pull Request: develop -> main
  -> main merge
  -> Deploy workflow
  -> GitHub OIDC role
  -> S3 release artifacts
  -> SSM Run Command
  -> EC2 release symlink switch
```

EC2 안에서는 아래 구조를 사용합니다.

```text
/opt/monthly-goal-tracker/backend/monthly-goal-api -> backend/releases/<release-id>/monthly-goal-api
/opt/monthly-goal-tracker/frontend/dist -> frontend/releases/<release-id>
/etc/monthly-goal-tracker/api.env
/etc/monthly-goal-tracker/postgres.env
/etc/caddy/Caddyfile
```

## 사전 준비 체크리스트

### GitHub

- `production` environment를 만듭니다.
- `production` environment의 deployment branch rule을 `main`으로 제한합니다.
- repository rule은 `main` 직접 push를 막고 PR merge만 허용합니다.
- `production` environment variables에 아래 값을 설정합니다.

```text
AWS_ROLE_TO_ASSUME=arn:aws:iam::<AWS_ACCOUNT_ID>:role/<GITHUB_DEPLOY_ROLE_NAME>
AWS_REGION=<AWS_REGION>
DEPLOY_ARTIFACT_BUCKET=<DEPLOY_ARTIFACT_BUCKET>
EC2_INSTANCE_ID=<EC2_INSTANCE_ID>
```

### AWS

- GitHub OIDC provider가 있어야 합니다.
- GitHub deploy role trust policy는 `production` environment subject로
  제한합니다.

```text
repo:<GITHUB_OWNER>/<GITHUB_REPO>:environment:production
```

- GitHub deploy role은 아래 작업만 허용합니다.
  - S3 `releases/*` artifact write
  - 대상 EC2 instance에 `AWS-RunShellScript` SSM command 실행
  - SSM command result 조회
- EC2 instance profile에는 아래 권한이 필요합니다.
  - `AmazonSSMManagedInstanceCore`
  - S3 `releases/*` artifact read
- S3 artifact bucket은 public access block을 켭니다.
- S3 artifact bucket에는 lifecycle rule을 설정해 오래된 release artifact를
  만료시킵니다.

예시 policy와 lifecycle 파일:

```text
deploy/aws/github-oidc-trust-policy.example.json
deploy/aws/github-deploy-role-policy.example.json
deploy/aws/ec2-instance-policy.example.json
deploy/aws/s3-lifecycle.example.json
```

### EC2

- SSM Agent가 실행 중이어야 합니다.
- AWS CLI가 설치되어 있어야 합니다.
- `monthly-goal-api.service` systemd unit이 등록되어 있어야 합니다.
- Caddy가 `/opt/monthly-goal-tracker/frontend/dist`를 static root로 사용해야
  합니다.
- API는 `127.0.0.1:8080`에만 bind합니다.
- PostgreSQL은 Docker Compose로 실행하고 host port는 loopback에만 bind합니다.
- 보안 그룹은 `80/tcp`, `443/tcp`만 public으로 엽니다.
- `22/tcp`는 운영자 고정 IP `/32`로 제한하거나 Session Manager로 대체합니다.
- `8080`, `5432`, `5433`은 public으로 열지 않습니다.

SSM 등록 상태 확인:

```bash
aws ssm describe-instance-information \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --filters Key=InstanceIds,Values=<EC2_INSTANCE_ID>
```

정상이라면 `PingStatus`가 `Online`입니다.

## 해피패스

### 1. feature 작업을 develop에 반영

```bash
git checkout develop
git status --short --branch
git push
```

`develop` push 후 아래 CI가 성공해야 합니다.

```bash
gh run list --branch develop --limit 5
```

기대 결과:

```text
Backend CI   success
Frontend CI  success
```

### 2. develop에서 main으로 PR 생성

```bash
gh pr create \
  --base main \
  --head develop \
  --title "<PR_TITLE>" \
  --body "<PR_BODY>"
```

PR checks 확인:

```bash
gh pr checks <PR_NUMBER> --watch
```

모든 check가 `pass`가 되면 merge합니다.

```bash
gh pr merge <PR_NUMBER> --merge
```

`main` 직접 push는 repository rule로 막혀 있어야 합니다. 이 제약은 의도된
운영 정책입니다.

### 3. production Deploy workflow 확인

PR merge 후 `main` push event로 `Deploy` workflow가 실행됩니다.

```bash
gh run list --branch main --limit 8
gh run watch <DEPLOY_RUN_ID> --exit-status
```

기대 job 순서:

```text
Detect changes
Build frontend       # frontend 배포 대상일 때
Build backend        # backend 배포 대상일 때
Deploy frontend      # frontend 배포 대상일 때
Deploy backend       # backend 배포 대상일 때
```

성공 기준:

- build job이 artifact를 생성합니다.
- deploy job이 artifact를 S3에 업로드합니다.
- SSM command가 EC2에서 배포 script를 실행합니다.
- backend deploy는 `/api/health` 확인까지 통과합니다.
- frontend deploy는 `dist` symlink를 새 release로 교체합니다.

### 4. 배포 후 EC2 내부 확인

```bash
aws ssm send-command \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --instance-ids <EC2_INSTANCE_ID> \
  --document-name AWS-RunShellScript \
  --comment "Monthly Goal Tracker post-deploy health" \
  --parameters '{
    "commands": [
      "set -euo pipefail",
      "systemctl is-active monthly-goal-api",
      "curl -fsS http://127.0.0.1:8080/api/health",
      "readlink -f /opt/monthly-goal-tracker/backend/monthly-goal-api",
      "readlink -f /opt/monthly-goal-tracker/frontend/dist"
    ]
  }' \
  --query Command.CommandId \
  --output text
```

이후 command 결과를 확인합니다.

```bash
aws ssm wait command-executed \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --command-id <COMMAND_ID> \
  --instance-id <EC2_INSTANCE_ID>

aws ssm get-command-invocation \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --command-id <COMMAND_ID> \
  --instance-id <EC2_INSTANCE_ID> \
  --query '{Status:Status,Output:StandardOutputContent,Error:StandardErrorContent}' \
  --output json
```

정상 출력 예시:

```text
active
{"message":"ok"}
/opt/monthly-goal-tracker/backend/releases/<release-id>/monthly-goal-api
/opt/monthly-goal-tracker/frontend/releases/<release-id>
```

### 5. 외부 접속 확인

```bash
curl -I https://example.com
curl -u <BASIC_AUTH_USER> https://example.com/api/health
```

브라우저에서 아래 흐름을 확인합니다.

- Basic Auth 로그인
- 현재 월 열기
- 목표 생성
- 일일 완료 체크
- 일일 메모 저장
- 다음 달 준비 흐름

## 수동 배포

GitHub Actions의 `Deploy` workflow에서 `workflow_dispatch`로 실행합니다.

선택 가능한 component:

```text
all
backend
frontend
```

수동 프로덕션 배포도 `main` 브랜치에서만 허용합니다. 다른 브랜치에서 수동
배포를 실행하면 workflow가 실패해야 합니다.

## Release 보관과 정리

EC2 release directory는 배포 script가 최근 5개만 유지합니다.

```bash
aws ssm send-command \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --instance-ids <EC2_INSTANCE_ID> \
  --document-name AWS-RunShellScript \
  --comment "Monthly Goal Tracker release inventory" \
  --parameters '{
    "commands": [
      "set -euo pipefail",
      "echo backend",
      "find /opt/monthly-goal-tracker/backend/releases -mindepth 1 -maxdepth 1 -type d -printf \"%f\\n\" | sort",
      "echo frontend",
      "find /opt/monthly-goal-tracker/frontend/releases -mindepth 1 -maxdepth 1 -type d -printf \"%f\\n\" | sort"
    ]
  }' \
  --query Command.CommandId \
  --output text
```

S3 artifact는 bucket lifecycle rule로 정리합니다. 기본 예시는
`releases/` 아래 artifact를 30일 후 만료시킵니다.

## Rollback

backend 배포는 restart 또는 health check 실패 시 이전 binary symlink로 자동
rollback을 시도합니다.

수동 rollback이 필요하면 EC2에서 이전 release를 확인한 뒤 symlink를 되돌립니다.

```bash
sudo ln -sfn /opt/monthly-goal-tracker/backend/releases/<PREVIOUS_RELEASE>/monthly-goal-api \
  /opt/monthly-goal-tracker/backend/monthly-goal-api.next
sudo mv -Tf /opt/monthly-goal-tracker/backend/monthly-goal-api.next \
  /opt/monthly-goal-tracker/backend/monthly-goal-api
sudo systemctl restart monthly-goal-api
curl -fsS http://127.0.0.1:8080/api/health
```

frontend rollback은 `dist` symlink만 이전 release로 되돌리면 됩니다.

```bash
sudo ln -sfn /opt/monthly-goal-tracker/frontend/releases/<PREVIOUS_RELEASE> \
  /opt/monthly-goal-tracker/frontend/dist.next
sudo mv -Tf /opt/monthly-goal-tracker/frontend/dist.next \
  /opt/monthly-goal-tracker/frontend/dist
```

## 트러블슈팅

### main 직접 push가 거부됨

증상:

```text
Changes must be made through a pull request.
```

원인:

- repository rule이 `main` 직접 push를 막고 있습니다.

해결:

- `develop -> main` PR을 만들고 checks 통과 후 merge합니다.
- 이 동작은 정상 운영 정책입니다.

### GitHub Actions에서 SSH timeout 또는 publickey 오류

증상:

```text
ssh: connect to host <host> port 22: Connection timed out
Permission denied (publickey)
```

원인:

- 구버전 SSH 배포 방식에서 발생하던 오류입니다.
- 현재 권장 방식은 SSH를 사용하지 않습니다.

해결:

- workflow가 OIDC, S3, SSM 방식인지 확인합니다.
- GitHub에 SSH private key secret을 추가하지 않습니다.
- GitHub Actions 접속을 위해 SSH를 `0.0.0.0/0`으로 열지 않습니다.
- SSH는 운영자 고정 IP `/32`로 제한하거나 Session Manager로 대체합니다.

### AWS CLI credential이 invalid

증상:

```text
InvalidClientTokenId
Unable to locate credentials
```

원인:

- 로컬 AWS profile의 credential이 만료되었거나 잘못 설정되었습니다.

해결:

```bash
aws configure list-profiles
aws sts get-caller-identity --profile <AWS_PROFILE>
```

유효한 profile을 찾고 모든 AWS CLI 명령에 `--profile <AWS_PROFILE>`을
명시합니다.

### GitHub OIDC assume role 실패

원인 후보:

- GitHub `production` environment variables가 비어 있습니다.
- deploy job에 `id-token: write` permission이 없습니다.
- IAM role trust policy의 subject가 GitHub environment와 맞지 않습니다.
- `production` environment branch rule이 `main`으로 제한되어 있지 않습니다.

확인:

```bash
gh variable list --env production
gh api repos/<GITHUB_OWNER>/<GITHUB_REPO>/environments/production
```

trust policy의 subject는 아래 형태여야 합니다.

```text
repo:<GITHUB_OWNER>/<GITHUB_REPO>:environment:production
```

### SSM managed instance가 Online이 아님

증상:

```text
InstanceInformationList: []
PingStatus가 Online이 아님
```

원인 후보:

- EC2에 instance profile이 연결되지 않았습니다.
- instance role에 `AmazonSSMManagedInstanceCore`가 없습니다.
- SSM Agent가 설치되지 않았거나 실행 중이 아닙니다.
- EC2 outbound HTTPS가 막혀 있습니다.
- IAM role 연결 직후라 전파 시간이 필요합니다.

확인:

```bash
aws ec2 describe-iam-instance-profile-associations \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --filters Name=instance-id,Values=<EC2_INSTANCE_ID>

aws ssm describe-instance-information \
  --profile <AWS_PROFILE> \
  --region <AWS_REGION> \
  --filters Key=InstanceIds,Values=<EC2_INSTANCE_ID>
```

EC2 내부 확인이 필요하면 운영자 접속으로 아래를 봅니다.

```bash
systemctl is-active snap.amazon-ssm-agent.amazon-ssm-agent.service
curl -fsS http://169.254.169.254/latest/meta-data/iam/security-credentials/
```

### SSM command에서 `aws: not found`

증상:

```text
aws: not found
```

원인:

- EC2 내부에 AWS CLI가 설치되어 있지 않습니다.

해결:

SSM Run Command 또는 운영자 접속으로 AWS CLI를 설치합니다.

```bash
sudo apt-get update
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y awscli
aws --version
```

### SSH 접속에서는 sudo가 막히지만 SSM은 동작함

증상:

```text
sudo: A terminal is required to authenticate
```

원인:

- 운영자 SSH 계정의 sudo 정책 때문입니다.

해결:

- 배포는 SSH sudo에 의존하지 않습니다.
- SSM Run Command는 root로 실행되므로 배포 script는 SSM 경로로 실행합니다.

### backend deploy 실패 또는 health check 실패

원인 후보:

- `/etc/monthly-goal-tracker/api.env` 값이 잘못되었습니다.
- PostgreSQL container가 내려가 있습니다.
- `DATABASE_URL`이 잘못되었습니다.
- `monthly-goal-api.service` unit이 잘못되었습니다.
- API가 `127.0.0.1:8080`에 bind하지 못했습니다.

확인:

```bash
systemctl status monthly-goal-api --no-pager
journalctl -u monthly-goal-api -n 100 --no-pager
curl -fsS http://127.0.0.1:8080/api/health
docker compose --env-file /etc/monthly-goal-tracker/postgres.env \
  -f /opt/monthly-goal-tracker/deploy/docker-compose.postgres.yml ps
```

backend deploy script는 실패 시 이전 binary로 자동 rollback을 시도합니다.
자동 rollback 후에도 실패하면 systemd log와 DB 상태를 먼저 확인합니다.

### frontend deploy는 성공했는데 화면이 바뀌지 않음

확인:

```bash
readlink -f /opt/monthly-goal-tracker/frontend/dist
test -f /opt/monthly-goal-tracker/frontend/dist/index.html
sudo caddy validate --config /etc/caddy/Caddyfile
sudo systemctl reload caddy
```

Caddy가 `/opt/monthly-goal-tracker/frontend/dist`를 root로 보고 있는지 확인합니다.

### Caddy `basic_auth` directive 오류

증상:

```text
unrecognized directive: basic_auth
```

원인:

- 설치된 Caddy 버전과 Caddyfile directive가 맞지 않습니다.

해결:

- Caddy 2.8 이상으로 업데이트합니다.
- `caddy version`과 `sudo caddy validate --config /etc/caddy/Caddyfile`을
  확인합니다.

### Dependabot Updates가 바로 실행됨

원인:

- `.github/dependabot.yml`에 `github-actions` ecosystem update가 추가되면
  GitHub가 dependency graph와 update scan을 실행합니다.

기대 결과:

- `Dependabot Updates` workflow가 `success`로 끝납니다.
- 이후 action SHA pin 업데이트 PR은 Dependabot이 생성합니다.

### Node.js 20 deprecation warning

원인:

- GitHub JavaScript action이 Node 20 runtime을 사용하는 구버전입니다.

해결:

- Node 24 runtime을 사용하는 action release로 업데이트합니다.
- `uses:`는 full commit SHA로 pinning합니다.
- Dependabot이 버전 주석을 갱신할 수 있도록 같은 줄에 `# vX.Y.Z` 주석을 둡니다.

```yaml
uses: actions/checkout@<FULL_COMMIT_SHA> # vX.Y.Z
```

## 보안 운영 기준

- AWS access key를 GitHub secret으로 넣지 않습니다.
- EC2 SSH private key를 GitHub secret으로 넣지 않습니다.
- `.env`, `.env.local`, `*.pem`, `*.key`는 commit하지 않습니다.
- 실제 account ID, ARN, instance ID, bucket 이름, private hostname, IP 주소는
  public 문서에 기록하지 않습니다.
- Basic Auth는 임시 접근 보호입니다. 애플리케이션 인증이 들어가기 전까지만
  사용합니다.
- API와 DB는 loopback으로만 노출합니다.
- SSM, OIDC, S3 권한은 대상 repository, production environment, artifact
  bucket, target instance로 제한합니다.

## 참고

- GitHub OIDC with AWS: https://github.com/aws-actions/configure-aws-credentials
- AWS Systems Manager Run Command setup: https://docs.aws.amazon.com/systems-manager/latest/userguide/run-command-setting-up.html
- AWS Systems Manager permissions reference: https://docs.aws.amazon.com/service-authorization/latest/reference/list_awssystemsmanager.html
- Dependabot GitHub Actions updates: https://docs.github.com/en/code-security/dependabot
