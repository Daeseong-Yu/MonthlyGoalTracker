# SSM 배포 가이드

이 배포 방식은 GitHub Actions가 AWS OIDC로 짧은 수명의 권한을 받고, 빌드 산출물을 S3에 올린 뒤, EC2에는 Systems Manager Run Command로 배포 명령만 전달하는 방식입니다.

## 배포 흐름

1. `main` 브랜치에 push되면 변경 파일을 기준으로 backend/frontend 배포 대상을 감지합니다.
2. backend는 Linux `amd64`, `arm64` 바이너리를 빌드합니다.
3. frontend는 정적 빌드 결과를 `frontend-dist.tar.gz`로 묶습니다.
4. GitHub Actions가 OIDC로 AWS role을 assume합니다.
5. 산출물을 S3 `releases/<release-id>/...` 경로에 업로드합니다.
6. SSM Run Command가 EC2에서 `deploy/scripts`의 배포 스크립트를 실행합니다.
7. backend는 systemd 재시작과 health check를 수행하고, 실패하면 이전 바이너리로 rollback합니다.
8. frontend는 release 디렉터리를 만들고 `dist` symlink를 교체합니다.

수동 실행은 GitHub Actions의 `Deploy` workflow에서 `workflow_dispatch`로 실행합니다. 수동 프로덕션 배포도 `main` 브랜치에서만 허용합니다.

## GitHub 설정

`Settings > Environments > production`을 만들고, deployment branch rule을 `main`으로 제한합니다.

`Settings > Secrets and variables > Actions > Variables`에 아래 값을 추가합니다.

| 이름 | 예시 |
| --- | --- |
| `AWS_ROLE_TO_ASSUME` | `arn:aws:iam::<AWS_ACCOUNT_ID>:role/<GITHUB_DEPLOY_ROLE_NAME>` |
| `AWS_REGION` | `<AWS_REGION>` |
| `DEPLOY_ARTIFACT_BUCKET` | `<DEPLOY_ARTIFACT_BUCKET>` |
| `EC2_INSTANCE_ID` | `<EC2_INSTANCE_ID>` |

AWS 접근키 secret은 사용하지 않습니다.

## AWS OIDC provider

계정에 GitHub OIDC provider가 없으면 한 번만 생성합니다.

```bash
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com
```

이미 존재하는지는 아래 명령으로 확인합니다.

```bash
aws iam list-open-id-connect-providers
```

## GitHub deploy role

GitHub Actions가 assume할 IAM role을 만들고 trust policy를 설정합니다.

- 예시: `deploy/aws/github-oidc-trust-policy.example.json`
- `<AWS_ACCOUNT_ID>`, `<GITHUB_OWNER>`, `<GITHUB_REPO>`를 실제 값으로 교체합니다.
- trust policy는 `production` environment subject를 사용합니다. 그래서 GitHub environment의 deployment branch rule을 `main`으로 제한해야 합니다.

권한 policy는 아래 예시를 기준으로 만듭니다.

- 예시: `deploy/aws/github-deploy-role-policy.example.json`
- `<DEPLOY_ARTIFACT_BUCKET>`, `<AWS_REGION>`, `<AWS_ACCOUNT_ID>`, `<EC2_INSTANCE_ID>`를 실제 값으로 교체합니다.
- `ssm:GetCommandInvocation`은 AWS Systems Manager에서 resource-level permission을 지원하지 않으므로 `Resource: "*"`가 필요합니다. 대신 action을 읽기 전용 1개로 제한하고 `aws:RequestedRegion` condition을 둡니다.

## S3 artifact bucket

배포 산출물 전용 S3 bucket을 만듭니다. Public access block은 켜둡니다.

권장 lifecycle 설정:

```bash
aws s3api put-bucket-lifecycle-configuration \
  --bucket <DEPLOY_ARTIFACT_BUCKET> \
  --lifecycle-configuration file://deploy/aws/s3-lifecycle.example.json
```

기본 예시는 `releases/` 아래 산출물을 30일 후 만료시킵니다.

## EC2 instance role

EC2 instance profile에는 아래가 필요합니다.

- AWS managed policy: `AmazonSSMManagedInstanceCore`
- 추가 inline policy: `deploy/aws/ec2-instance-policy.example.json`

inline policy의 `<DEPLOY_ARTIFACT_BUCKET>`를 실제 artifact bucket 이름으로 교체합니다.

EC2에서 아래가 준비되어 있어야 합니다.

- SSM Agent 실행 중
- AWS CLI 설치
- `monthly-goal-api.service` systemd unit 활성화
- `/opt/monthly-goal-tracker/backend` 쓰기 가능
- `/opt/monthly-goal-tracker/frontend` 쓰기 가능
- backend health check: `http://127.0.0.1:8080/api/health`

SSM 연결 상태는 아래 명령으로 확인합니다.

```bash
aws ssm describe-instance-information \
  --filters Key=InstanceIds,Values=<EC2_INSTANCE_ID>
```

## Push 이후 확인

1. GitHub Actions에서 `Deploy` workflow가 실행되는지 확인합니다.
2. `Detect changes` job에서 배포 대상이 의도대로 감지됐는지 봅니다.
3. `Build backend` 또는 `Build frontend` job이 산출물을 만들었는지 봅니다.
4. deploy job에서 S3 upload가 성공했는지 확인합니다.
5. SSM command 결과에서 `Status`가 `Success`인지 확인합니다.
6. 사이트 접속과 backend health check를 확인합니다.

`deploy/scripts` 또는 `.github/workflows/deploy.yml` 변경이 `main`에 들어가면 workflow 검증을 위해 관련 component가 배포 대상으로 잡힙니다.

## 참고

- GitHub OIDC with AWS: https://github.com/aws-actions/configure-aws-credentials
- AWS Systems Manager Run Command setup: https://docs.aws.amazon.com/systems-manager/latest/userguide/run-command-setting-up.html
- AWS Systems Manager permissions reference: https://docs.aws.amazon.com/service-authorization/latest/reference/list_awssystemsmanager.html
