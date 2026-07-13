import assert from "node:assert/strict";

import { App } from "aws-cdk-lib";
import { Template } from "aws-cdk-lib/assertions";

import { protectedApiPaths, publicApiPaths } from "../lib/public-api.js";
import { ServerlessAppStack } from "../lib/serverless-app-stack.js";
import { stageConfigFromContext } from "../lib/stage-config.js";
import type { StageConfig } from "../lib/stage-config.js";

type CloudFormationResource = {
  readonly Type: string;
  readonly Properties?: Record<string, unknown>;
  readonly DeletionPolicy?: string;
  readonly UpdateReplacePolicy?: string;
};

type CloudFormationTemplate = {
  readonly Resources?: Record<string, CloudFormationResource>;
};

const stagingConfig: StageConfig = {
  stage: "staging",
  awsRegion: "us-east-1",
  backupRetentionDays: 1,
  deletionProtection: false,
  removalPolicy: "destroy",
  emailFrom: "no-reply@example.invalid",
  siteBaseUrl: "https://monthly-goal-tracker-staging.example.invalid",
  signupDisabled: false,
};

const productionConfig: StageConfig = {
  ...stagingConfig,
  stage: "production",
  backupRetentionDays: 7,
  deletionProtection: true,
  removalPolicy: "retain",
  siteBaseUrl: "https://monthly-goal-tracker.example.invalid",
};

const customDomainConfig: StageConfig = {
  ...stagingConfig,
  domainName: "serverless.example.invalid",
  certificateArn: "arn:aws:acm:us-east-1:000000000000:certificate/example",
  siteBaseUrl: "https://serverless.example.invalid",
};

run("staging template preserves serverless routing and runtime boundaries", () => {
  const template = synthTemplate(stagingConfig);

  assertExplicitApiContractRoutes(template);
  assertLambdaDatabaseSecretResolution(template);
  assertDirectRdsConnectionBoundary(template);
  assertApiLambdaApplicationConfiguration(template, stagingConfig);
  assertLambdaIamPermissions(template, stagingConfig);
  assertPrivateMigrationAndAuditFunctions(template);
  assertStagingRequestAndLogLimits(template);
  assertCloudFrontSiteBoundary(template);
  assertVpcAwsApiEndpoints(template);
  assertPrivateNetworkBoundary(template);
});

run("stage lifecycle policies separate staging and production", () => {
  const stagingTemplate = synthTemplate(stagingConfig);
  const productionTemplate = synthTemplate(productionConfig);

  assertDatabaseLifecycle(stagingTemplate, {
    deletionProtection: false,
    deletionPolicy: "Delete",
    updateReplacePolicy: "Delete",
    backupRetentionDays: 1,
  });
  assertDatabaseLifecycle(productionTemplate, {
    deletionProtection: true,
    deletionPolicy: "Retain",
    updateReplacePolicy: "Retain",
    backupRetentionDays: 7,
  });
});

run("custom domain context configures CloudFront alias and certificate", () => {
  const template = synthTemplate(customDomainConfig);
  const distribution = resourcesOfType(template, "AWS::CloudFront::Distribution")[0];
  const distributionConfig = objectProperty(distribution, "DistributionConfig");
  const viewerCertificate = objectProperty(distributionConfig, "ViewerCertificate");

  assert.deepEqual(distributionConfig.Aliases, [customDomainConfig.domainName]);
  assert.equal(viewerCertificate.AcmCertificateArn, customDomainConfig.certificateArn);
  assert.equal(viewerCertificate.SslSupportMethod, "sni-only");
});

run("stage config keeps custom domain and app base URL in the same site boundary", () => {
  const validConfig = stageConfigFromContext(
    new App({
      context: {
        stage: "staging",
        domainName: customDomainConfig.domainName,
        certificateArn: customDomainConfig.certificateArn,
      },
    }),
  );
  assert.equal(validConfig.siteBaseUrl, customDomainConfig.siteBaseUrl);

  assert.throws(() =>
    stageConfigFromContext(
      new App({
        context: {
          stage: "staging",
          domainName: customDomainConfig.domainName,
          certificateArn: customDomainConfig.certificateArn,
          siteBaseUrl: "https://different.example.invalid",
        },
      }),
    ),
  );
  assert.throws(() =>
    stageConfigFromContext(
      new App({
        context: {
          stage: "staging",
          siteBaseUrl: "http://serverless.example.invalid",
        },
      }),
    ),
  );
  assert.throws(() =>
    stageConfigFromContext(
      new App({
        context: {
          stage: "staging",
          emailFrom: "not-an-email",
        },
      }),
    ),
  );
});

run("signup emergency switch is an explicit stage context", () => {
  const defaultConfig = stageConfigFromContext(new App({ context: { stage: "production" } }));
  assert.equal(defaultConfig.signupDisabled, false);

  const disabledConfig = stageConfigFromContext(
    new App({ context: { stage: "production", signupDisabled: "true" } }),
  );
  assert.equal(disabledConfig.signupDisabled, true);

  assert.throws(() =>
    stageConfigFromContext(
      new App({ context: { stage: "production", signupDisabled: "yes" } }),
    ),
  );
});

console.log("infra template invariant tests passed");

function run(name: string, test: () => void): void {
  try {
    test();
  } catch (error) {
    throw new Error(`Failed: ${name}`, { cause: error });
  }
}

function synthTemplate(stageConfig: StageConfig): CloudFormationTemplate {
  const app = new App();
  const stack = new ServerlessAppStack(app, `MonthlyGoalTracker-${stageConfig.stage}`, {
    env: {
      account: "000000000000",
      region: stageConfig.awsRegion,
    },
    stageConfig,
  });

  return Template.fromStack(stack).toJSON() as CloudFormationTemplate;
}

function resourcesOfType(template: CloudFormationTemplate, type: string): CloudFormationResource[] {
  return Object.values(template.Resources ?? {}).filter((resource) => resource.Type === type);
}

function resourceEntriesOfType(template: CloudFormationTemplate, type: string): Array<[string, CloudFormationResource]> {
  return Object.entries(template.Resources ?? {}).filter(([, resource]) => resource.Type === type);
}

function assertExplicitApiContractRoutes(template: CloudFormationTemplate): void {
  const routeKeys = resourcesOfType(template, "AWS::ApiGatewayV2::Route")
    .map((resource) => String(resource.Properties?.RouteKey))
    .sort();
  const expectedRouteKeys = [...publicApiPaths, ...protectedApiPaths].map(apiGatewayRouteKey).sort();

  assert.deepEqual(routeKeys, expectedRouteKeys);
  assert.ok(!routeKeys.includes("$default"));
}

function apiGatewayRouteKey(contract: string): string {
  return contract.replace(/:([A-Za-z0-9_]+)/g, "{$1}");
}

function assertLambdaDatabaseSecretResolution(template: CloudFormationTemplate): void {
  const functions = resourcesOfType(template, "AWS::Lambda::Function");
  assert.equal(functions.length, 3);

  for (const lambdaFunction of functions) {
    const variables = environmentVariables(lambdaFunction);
    assert.ok(!Object.hasOwn(variables, "DATABASE_URL"));
    assert.ok(Object.hasOwn(variables, "DATABASE_SECRET_ARN"));
    assert.ok(Object.hasOwn(variables, "DATABASE_HOST"));
    assert.ok(Object.hasOwn(variables, "DATABASE_NAME"));
    assert.equal(variables.DATABASE_SSLMODE, "require");
    assert.ok(JSON.stringify(variables.DATABASE_HOST).includes("Database"));
    assert.ok(!JSON.stringify(variables.DATABASE_HOST).includes("Proxy"));
  }
}

function assertDirectRdsConnectionBoundary(template: CloudFormationTemplate): void {
  assert.equal(resourcesOfType(template, "AWS::RDS::DBProxy").length, 0);
  assert.ok(!JSON.stringify(template).includes("DatabaseProxy"));

  const databases = resourcesOfType(template, "AWS::RDS::DBInstance");
  assert.equal(databases.length, 1);
  assert.equal(databases[0].Properties?.AllocatedStorage, "20");
  assert.ok(!Object.hasOwn(databases[0].Properties ?? {}, "MaxAllocatedStorage"));

  const postgresIngress = resourcesOfType(template, "AWS::EC2::SecurityGroupIngress").filter((rule) => {
    return rule.Properties?.FromPort === 5432 && rule.Properties?.ToPort === 5432;
  });
  assert.equal(postgresIngress.length, 1);

  const directDatabaseRule = JSON.stringify(postgresIngress[0].Properties);
  assert.ok(directDatabaseRule.includes("ApiLambdaSecurityGroup"));
  assert.ok(directDatabaseRule.includes("DatabaseSecurityGroup"));
}

function assertApiLambdaApplicationConfiguration(template: CloudFormationTemplate, stageConfig: StageConfig): void {
  const apiFunction = resourcesOfType(template, "AWS::Lambda::Function").find((lambdaFunction) => {
    const variables = environmentVariables(lambdaFunction);
    return variables.APP_EMAIL_PROVIDER === "ses";
  });
  assert.ok(apiFunction);

  const variables = environmentVariables(apiFunction);
  assert.equal(variables.APP_COOKIE_SECURE, "true");
  assert.equal(variables.APP_COOKIE_SAMESITE, "lax");
  assert.equal(variables.APP_SESSION_COOKIE_NAME, "mgt_session");
  assert.equal(variables.APP_CSRF_COOKIE_NAME, "mgt_csrf");
  assert.equal(variables.APP_EMAIL_PROVIDER, "ses");
  assert.equal(variables.APP_EMAIL_FROM, stageConfig.emailFrom);
  assert.equal(variables.APP_EMAIL_VERIFICATION_BASE_URL, `${stageConfig.siteBaseUrl}/`);
  assert.equal(variables.APP_PASSWORD_RESET_BASE_URL, `${stageConfig.siteBaseUrl}/`);
  assert.equal(variables.APP_SES_REGION, stageConfig.awsRegion);
  assert.equal(variables.APP_SIGNUP_DISABLED, String(stageConfig.signupDisabled));
}

function assertLambdaIamPermissions(template: CloudFormationTemplate, stageConfig: StageConfig): void {
  const policies = resourceEntriesOfType(template, "AWS::IAM::Policy");

  assertSecretReadPolicy(policies, "ApiFunctionServiceRoleDefaultPolicy");
  assertSecretReadPolicy(policies, "MigrationFunctionServiceRoleDefaultPolicy");
  assertSecretReadPolicy(policies, "AuditFunctionServiceRoleDefaultPolicy");

  const apiPolicy = policyByLogicalIdPrefix(policies, "ApiFunctionServiceRoleDefaultPolicy");
  const sesStatement = policyStatements(apiPolicy).find((statement) =>
    statementActions(statement).includes("ses:SendEmail"),
  );
  assert.ok(sesStatement);

  const condition = objectProperty(sesStatement, "Condition");
  const stringEquals = objectProperty(condition, "StringEquals");
  assert.equal(stringEquals["ses:FromAddress"], stageConfig.emailFrom);

  const sesResource = JSON.stringify(propertyValue(sesStatement, "Resource"));
  assert.ok(sesResource.includes(`:ses:${stageConfig.awsRegion}:`));
  assert.ok(sesResource.includes(":identity/*"));
}

function assertStagingRequestAndLogLimits(template: CloudFormationTemplate): void {
  const apiFunction = resourcesOfType(template, "AWS::Lambda::Function").find((lambdaFunction) => {
    return environmentVariables(lambdaFunction).APP_EMAIL_PROVIDER === "ses";
  });
  assert.ok(apiFunction);
  assert.equal(apiFunction.Properties?.ReservedConcurrentExecutions, 4);
  assert.equal(template.Resources?.MigrationFunction?.Properties?.ReservedConcurrentExecutions, 1);
  assert.equal(template.Resources?.AuditFunction?.Properties?.ReservedConcurrentExecutions, 1);
  for (const lambdaFunction of resourcesOfType(template, "AWS::Lambda::Function")) {
    assert.ok(objectProperty(lambdaFunction, "LoggingConfig").LogGroup);
  }

  const apiStage = resourcesOfType(template, "AWS::ApiGatewayV2::Stage").find((stage) => {
    return stage.Properties?.StageName === "$default";
  });
  assert.ok(apiStage);
  const routeSettings = objectProperty(apiStage, "DefaultRouteSettings");
  assert.equal(routeSettings.ThrottlingBurstLimit, 20);
  assert.equal(routeSettings.ThrottlingRateLimit, 10);

  const logGroups = resourcesOfType(template, "AWS::Logs::LogGroup");
  assert.equal(logGroups.length, 3);
  for (const logGroup of logGroups) {
    assert.equal(logGroup.Properties?.RetentionInDays, 7);
  }
}

function assertSecretReadPolicy(
  policies: Array<[string, CloudFormationResource]>,
  logicalIdPrefix: string,
): void {
  const policy = policyByLogicalIdPrefix(policies, logicalIdPrefix);
  const secretStatement = policyStatements(policy).find((statement) => {
    const actions = statementActions(statement);
    const resource = JSON.stringify(propertyValue(statement, "Resource"));

    return (
      actions.includes("secretsmanager:GetSecretValue") &&
      actions.includes("secretsmanager:DescribeSecret") &&
      resource.includes("DatabaseCredentials")
    );
  });

  assert.ok(secretStatement);
}

function policyByLogicalIdPrefix(
  policies: Array<[string, CloudFormationResource]>,
  logicalIdPrefix: string,
): CloudFormationResource {
  const policy = policies.find(([logicalId]) => logicalId.startsWith(logicalIdPrefix))?.[1];
  assert.ok(policy);
  return policy;
}

function policyStatements(policy: CloudFormationResource): Array<Record<string, unknown>> {
  const policyDocument = objectProperty(policy, "PolicyDocument");
  const statements = propertyValue(policyDocument, "Statement");
  if (Array.isArray(statements)) {
    return statements.map(objectValue);
  }

  return [objectValue(statements)];
}

function statementActions(statement: Record<string, unknown>): string[] {
  const actions = propertyValue(statement, "Action");
  if (Array.isArray(actions)) {
    return actions.map(String);
  }

  return [String(actions)];
}

function assertPrivateMigrationAndAuditFunctions(template: CloudFormationTemplate): void {
  const resources = template.Resources ?? {};
  assert.equal(resources.MigrationFunction?.Type, "AWS::Lambda::Function");
  assert.equal(resources.AuditFunction?.Type, "AWS::Lambda::Function");

  const routeTargets = JSON.stringify(resourcesOfType(template, "AWS::ApiGatewayV2::Route"));
  assert.ok(!routeTargets.includes("MigrationFunction"));
  assert.ok(!routeTargets.includes("AuditFunction"));
}

function assertCloudFrontSiteBoundary(template: CloudFormationTemplate): void {
  const distributions = resourcesOfType(template, "AWS::CloudFront::Distribution");
  assert.equal(distributions.length, 1);

  const distributionConfig = objectProperty(distributions[0], "DistributionConfig");
  const origins = arrayProperty(distributionConfig, "Origins");
  const defaultBehavior = objectProperty(distributionConfig, "DefaultCacheBehavior");
  const cacheBehaviors = arrayProperty(distributionConfig, "CacheBehaviors");
  const apiBehavior = cacheBehaviors.find((behavior) => objectValue(behavior).PathPattern === "/api/*");
  const apiRootBehavior = cacheBehaviors.find((behavior) => objectValue(behavior).PathPattern === "/api");

  assert.equal(distributionConfig.DefaultRootObject, "index.html");
  assert.ok(defaultBehavior.FunctionAssociations);
  assert.ok(defaultBehavior.ResponseHeadersPolicyId);
  assert.ok(apiBehavior);
  assert.ok(apiRootBehavior);

  const apiBehaviorConfig = assertApiCacheBehavior(apiBehavior);
  assertApiCacheBehavior(apiRootBehavior);

  const apiOrigin = origins.find((origin) => objectValue(origin).Id === apiBehaviorConfig.TargetOriginId);
  assert.ok(apiOrigin);
  const apiOriginConfig = objectValue(apiOrigin);
  const apiOriginDomain = JSON.stringify(apiOriginConfig.DomainName);
  assert.ok(apiOriginDomain.includes("execute-api"));
  assert.equal(objectProperty(apiOriginConfig, "CustomOriginConfig").OriginProtocolPolicy, "https-only");

  const frontendOrigin = origins.find((origin) => objectValue(origin).Id === defaultBehavior.TargetOriginId);
  assert.ok(frontendOrigin);
  const frontendOriginConfig = objectValue(frontendOrigin);
  assert.ok(frontendOriginConfig.OriginAccessControlId);
  assert.ok(frontendOriginConfig.S3OriginConfig);
}

function assertApiCacheBehavior(behavior: unknown): Record<string, unknown> {
  const apiBehaviorConfig = objectValue(behavior);
  assert.equal(apiBehaviorConfig.ViewerProtocolPolicy, "redirect-to-https");
  assert.deepEqual(apiBehaviorConfig.AllowedMethods, [
    "GET",
    "HEAD",
    "OPTIONS",
    "PUT",
    "PATCH",
    "POST",
    "DELETE",
  ]);
  assert.equal(apiBehaviorConfig.CachePolicyId, "4135ea2d-6df8-44a3-9df3-4b5a84be39ad");
  assert.equal(apiBehaviorConfig.OriginRequestPolicyId, "b689b0a8-53d0-40ab-baf2-68738e2966ac");
  assert.ok(!Object.hasOwn(apiBehaviorConfig, "FunctionAssociations"));

  return apiBehaviorConfig;
}

function assertVpcAwsApiEndpoints(template: CloudFormationTemplate): void {
  const endpoints = resourcesOfType(template, "AWS::EC2::VPCEndpoint");
  assert.equal(endpoints.length, 2);

  const endpointConfig = endpoints.map((endpoint) => endpoint.Properties ?? {});
  const endpointServiceNames = JSON.stringify(endpointConfig.map((endpoint) => endpoint.ServiceName));

  assert.ok(endpointServiceNames.includes("secretsmanager"));
  assert.ok(endpointServiceNames.includes("email"));
  for (const endpoint of endpointConfig) {
    assert.equal(endpoint.VpcEndpointType, "Interface");
    assert.equal(endpoint.PrivateDnsEnabled, true);
  }
}

function assertPrivateNetworkBoundary(template: CloudFormationTemplate): void {
  assert.equal(resourcesOfType(template, "AWS::EC2::NatGateway").length, 0);
  assert.equal(resourcesOfType(template, "AWS::EC2::InternetGateway").length, 0);
  assert.equal(resourcesOfType(template, "AWS::EC2::VPCGatewayAttachment").length, 0);

  for (const route of resourcesOfType(template, "AWS::EC2::Route")) {
    assert.notEqual(route.Properties?.DestinationCidrBlock, "0.0.0.0/0");
    assert.notEqual(route.Properties?.DestinationIpv6CidrBlock, "::/0");
  }

  for (const securityGroup of resourcesOfType(template, "AWS::EC2::SecurityGroup")) {
    assertNoPublicEgress(securityGroup.Properties?.SecurityGroupEgress);
  }
  for (const egressRule of resourcesOfType(template, "AWS::EC2::SecurityGroupEgress")) {
    assertNoPublicEgress([egressRule.Properties]);
  }
}

function assertNoPublicEgress(rawRules: unknown): void {
  if (rawRules === undefined) {
    return;
  }
  assert.ok(Array.isArray(rawRules));
  for (const rawRule of rawRules) {
    const rule = objectValue(rawRule);
    assert.notEqual(rule.CidrIp, "0.0.0.0/0");
    assert.notEqual(rule.CidrIpv6, "::/0");
  }
}

function assertDatabaseLifecycle(
  template: CloudFormationTemplate,
  expected: {
    readonly deletionProtection: boolean;
    readonly deletionPolicy: string;
    readonly updateReplacePolicy: string;
    readonly backupRetentionDays: number;
  },
): void {
  const databases = resourcesOfType(template, "AWS::RDS::DBInstance");
  assert.equal(databases.length, 1);

  const database = databases[0];
  assert.equal(database.Properties?.DeletionProtection, expected.deletionProtection);
  assert.equal(database.DeletionPolicy, expected.deletionPolicy);
  assert.equal(database.UpdateReplacePolicy, expected.updateReplacePolicy);
  assert.equal(database.Properties?.BackupRetentionPeriod, expected.backupRetentionDays);
}

function environmentVariables(lambdaFunction: CloudFormationResource): Record<string, unknown> {
  const environment = objectProperty(lambdaFunction, "Environment");
  return objectProperty(environment, "Variables");
}

function objectProperty(resource: CloudFormationResource | Record<string, unknown>, key: string): Record<string, unknown> {
  return objectValue(propertyValue(resource, key));
}

function arrayProperty(resource: CloudFormationResource | Record<string, unknown>, key: string): unknown[] {
  const value = propertyValue(resource, key);
  assert.ok(Array.isArray(value));
  return value;
}

function propertyValue(resource: CloudFormationResource | Record<string, unknown>, key: string): unknown {
  const record = resource as Record<string, unknown>;
  const properties = record.Properties;
  if (isRecord(properties) && Object.hasOwn(properties, key)) {
    return properties[key];
  }

  return record[key];
}

function objectValue(value: unknown): Record<string, unknown> {
  assert.equal(typeof value, "object");
  assert.ok(value !== null);
  assert.ok(!Array.isArray(value));
  return value as Record<string, unknown>;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
