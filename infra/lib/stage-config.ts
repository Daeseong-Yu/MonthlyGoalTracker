import type { App } from "aws-cdk-lib";

export type DeploymentStage = "staging" | "production";

export type StageConfig = {
  readonly stage: DeploymentStage;
  readonly awsRegion: string;
  readonly backupRetentionDays: number;
  readonly deletionProtection: boolean;
  readonly removalPolicy: "destroy" | "retain";
  readonly emailFrom: string;
  readonly siteBaseUrl: string;
  readonly signupDisabled: boolean;
  readonly domainName?: string;
  readonly certificateArn?: string;
};

const stageConfigs: Record<DeploymentStage, StageConfig> = {
  staging: {
    stage: "staging",
    awsRegion: "us-east-1",
    backupRetentionDays: 1,
    deletionProtection: false,
    removalPolicy: "destroy",
    emailFrom: "no-reply@example.invalid",
    siteBaseUrl: "https://monthly-goal-tracker-staging.example.invalid",
    signupDisabled: false,
  },
  production: {
    stage: "production",
    awsRegion: "us-east-1",
    backupRetentionDays: 7,
    deletionProtection: true,
    removalPolicy: "retain",
    emailFrom: "no-reply@example.invalid",
    siteBaseUrl: "https://monthly-goal-tracker.example.invalid",
    signupDisabled: false,
  },
};

export function stageConfigFromContext(app: App): StageConfig {
  const rawStage = app.node.tryGetContext("stage") ?? app.node.tryGetContext("defaultStage") ?? "staging";
  if (!isDeploymentStage(rawStage)) {
    throw new Error(`Unsupported stage '${String(rawStage)}'. Expected 'staging' or 'production'.`);
  }

  const baseConfig = stageConfigs[rawStage];
  const domainName = optionalStringContext(app, "domainName");
  const certificateArn = optionalStringContext(app, "certificateArn");
  validateCustomDomainConfig(domainName, certificateArn);

  const siteBaseUrlContext = optionalStringContext(app, "siteBaseUrl");
  const siteBaseUrl = resolveSiteBaseUrl(siteBaseUrlContext, domainName, baseConfig.siteBaseUrl);
  validateSiteBaseUrl(siteBaseUrl, domainName);

  return {
    ...baseConfig,
    awsRegion: stringContext(app, "awsRegion", baseConfig.awsRegion),
    emailFrom: validateEmailFrom(stringContext(app, "emailFrom", baseConfig.emailFrom)),
    siteBaseUrl,
    signupDisabled: booleanContext(app, "signupDisabled", baseConfig.signupDisabled),
    domainName,
    certificateArn,
  };
}

function isDeploymentStage(value: unknown): value is DeploymentStage {
  return value === "staging" || value === "production";
}

function stringContext(app: App, key: string, fallback: string): string {
  return optionalStringContext(app, key) ?? fallback;
}

function optionalStringContext(app: App, key: string): string | undefined {
  const rawValue = app.node.tryGetContext(key);
  const value = rawValue === undefined || rawValue === null ? "" : String(rawValue).trim();
  if (value === "") {
    return undefined;
  }

  return value;
}

function booleanContext(app: App, key: string, fallback: boolean): boolean {
  const rawValue = app.node.tryGetContext(key);
  if (rawValue === undefined || rawValue === null || rawValue === "") {
    return fallback;
  }
  if (rawValue === true || rawValue === "true") {
    return true;
  }
  if (rawValue === false || rawValue === "false") {
    return false;
  }

  throw new Error(`${key} context must be true or false.`);
}

function validateCustomDomainConfig(domainName: string | undefined, certificateArn: string | undefined): void {
  if ((domainName === undefined) !== (certificateArn === undefined)) {
    throw new Error("domainName and certificateArn context values must be provided together.");
  }
  if (domainName !== undefined && (domainName.includes("/") || domainName.includes("://"))) {
    throw new Error("domainName context must be a DNS host name without scheme or path.");
  }
  if (certificateArn !== undefined && !certificateArn.startsWith("arn:")) {
    throw new Error("certificateArn context must be an ACM certificate ARN.");
  }
  if (certificateArn !== undefined && !certificateArn.includes(":acm:us-east-1:")) {
    throw new Error("CloudFront certificateArn context must reference an ACM certificate in us-east-1.");
  }
}

function resolveSiteBaseUrl(siteBaseUrl: string | undefined, domainName: string | undefined, fallback: string): string {
  return trimTrailingSlash(siteBaseUrl ?? (domainName === undefined ? fallback : `https://${domainName}`));
}

function validateSiteBaseUrl(siteBaseUrl: string, domainName: string | undefined): void {
  let parsedURL: URL;
  try {
    parsedURL = new URL(siteBaseUrl);
  } catch {
    throw new Error("siteBaseUrl context must be an absolute HTTPS URL.");
  }
  if (parsedURL.protocol !== "https:") {
    throw new Error("siteBaseUrl context must use https.");
  }
  if (parsedURL.username !== "" || parsedURL.password !== "") {
    throw new Error("siteBaseUrl context must not contain credentials.");
  }
  if (parsedURL.pathname !== "/" || parsedURL.search !== "" || parsedURL.hash !== "") {
    throw new Error("siteBaseUrl context must not contain path, query, or fragment.");
  }
  if (domainName !== undefined && parsedURL.hostname.toLowerCase() !== domainName.toLowerCase()) {
    throw new Error("siteBaseUrl host must match domainName when custom domain context is provided.");
  }
}

function validateEmailFrom(emailFrom: string): string {
  const value = emailFrom.trim();
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    throw new Error("emailFrom context must be a valid email address.");
  }

  return value;
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}
