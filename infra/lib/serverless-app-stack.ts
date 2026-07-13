import { Aws, CfnOutput, Duration, RemovalPolicy, Stack, Tags } from "aws-cdk-lib";
import type { StackProps } from "aws-cdk-lib";
import * as apigatewayv2 from "aws-cdk-lib/aws-apigatewayv2";
import * as integrations from "aws-cdk-lib/aws-apigatewayv2-integrations";
import * as acm from "aws-cdk-lib/aws-certificatemanager";
import * as cloudfront from "aws-cdk-lib/aws-cloudfront";
import * as ecrAssets from "aws-cdk-lib/aws-ecr-assets";
import * as origins from "aws-cdk-lib/aws-cloudfront-origins";
import * as ec2 from "aws-cdk-lib/aws-ec2";
import * as iam from "aws-cdk-lib/aws-iam";
import * as lambda from "aws-cdk-lib/aws-lambda";
import * as logs from "aws-cdk-lib/aws-logs";
import * as rds from "aws-cdk-lib/aws-rds";
import * as s3 from "aws-cdk-lib/aws-s3";
import type { Construct } from "constructs";

import { protectedApiPaths, publicApiPaths, serverlessInvariants } from "./public-api.js";
import type { StageConfig } from "./stage-config.js";

export type ServerlessAppStackProps = StackProps & {
  readonly stageConfig: StageConfig;
};

export class ServerlessAppStack extends Stack {
  constructor(scope: Construct, id: string, props: ServerlessAppStackProps) {
    super(scope, id, props);

    const { stageConfig } = props;

    Tags.of(this).add("Application", "MonthlyGoalTracker");
    Tags.of(this).add("Stage", stageConfig.stage);
    Tags.of(this).add("Architecture", "serverless-lambda-postgres");

    const vpc = new ec2.Vpc(this, "Vpc", {
      maxAzs: 2,
      natGateways: 0,
      subnetConfiguration: [
        {
          cidrMask: 24,
          name: "private",
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
        },
      ],
    });

    const lambdaSecurityGroup = new ec2.SecurityGroup(this, "ApiLambdaSecurityGroup", {
      vpc,
      allowAllOutbound: false,
      description: "Restricted outbound access for Monthly Goal Tracker Lambda workloads.",
    });

    const databaseSecurityGroup = new ec2.SecurityGroup(this, "DatabaseSecurityGroup", {
      vpc,
      allowAllOutbound: false,
      description: "Inbound PostgreSQL access from private Lambda workloads only.",
    });

    const endpointSecurityGroup = new ec2.SecurityGroup(this, "EndpointSecurityGroup", {
      vpc,
      allowAllOutbound: false,
      description: "Interface VPC endpoint access from private Lambda workloads only.",
    });

    databaseSecurityGroup.addIngressRule(lambdaSecurityGroup, ec2.Port.tcp(5432), "Allow private Lambda workloads to connect to PostgreSQL.");
    lambdaSecurityGroup.addEgressRule(databaseSecurityGroup, ec2.Port.tcp(5432), "Allow private Lambda workloads to connect to PostgreSQL.");
    endpointSecurityGroup.addIngressRule(lambdaSecurityGroup, ec2.Port.tcp(443), "Allow private Lambda workloads to call AWS APIs through VPC endpoints.");
    lambdaSecurityGroup.addEgressRule(endpointSecurityGroup, ec2.Port.tcp(443), "Allow private Lambda workloads to call AWS APIs through VPC endpoints.");

    vpc.addInterfaceEndpoint("SecretsManagerEndpoint", {
      service: ec2.InterfaceVpcEndpointAwsService.SECRETS_MANAGER,
      securityGroups: [endpointSecurityGroup],
      open: false,
      subnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
    });

    vpc.addInterfaceEndpoint("SesApiEndpoint", {
      service: ec2.InterfaceVpcEndpointAwsService.EMAIL,
      securityGroups: [endpointSecurityGroup],
      open: false,
      subnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
    });

    const databaseCredentials = new rds.DatabaseSecret(this, "DatabaseCredentials", {
      username: "mgtadmin",
    });
    databaseCredentials.applyRemovalPolicy(removalPolicyFor(stageConfig));

    const database = new rds.DatabaseInstance(this, "Database", {
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
      securityGroups: [databaseSecurityGroup],
      engine: rds.DatabaseInstanceEngine.postgres({
        version: rds.PostgresEngineVersion.VER_17_9,
      }),
      databaseName: "monthly_goal_tracker",
      credentials: rds.Credentials.fromSecret(databaseCredentials),
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.T4G, ec2.InstanceSize.MICRO),
      allocatedStorage: 20,
      backupRetention: Duration.days(stageConfig.backupRetentionDays),
      deletionProtection: stageConfig.deletionProtection,
      multiAz: false,
      publiclyAccessible: false,
      storageEncrypted: true,
      removalPolicy: removalPolicyFor(stageConfig),
    });

    const apiLogGroup = new logs.LogGroup(this, "ApiLogGroup", {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: removalPolicyFor(stageConfig),
    });
    const migrationLogGroup = new logs.LogGroup(this, "MigrationLogGroup", {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: removalPolicyFor(stageConfig),
    });
    const auditLogGroup = new logs.LogGroup(this, "AuditLogGroup", {
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: removalPolicyFor(stageConfig),
    });

    const apiFunction = new lambda.DockerImageFunction(this, "ApiFunction", {
      code: lambda.DockerImageCode.fromImageAsset("../backend", {
        file: "Dockerfile.lambda",
        platform: ecrAssets.Platform.LINUX_ARM64,
        buildArgs: {
          APP_COMMAND: "lambda",
        },
      }),
      architecture: lambda.Architecture.ARM_64,
      memorySize: 512,
      timeout: Duration.seconds(15),
      reservedConcurrentExecutions: 4,
      logGroup: apiLogGroup,
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
      securityGroups: [lambdaSecurityGroup],
      environment: {
        APP_AUTH_RATE_LIMIT_MAX_BUCKETS: "10000",
        APP_COOKIE_SAMESITE: "lax",
        APP_COOKIE_SECURE: "true",
        APP_CSRF_COOKIE_NAME: "mgt_csrf",
        APP_EMAIL_FROM: stageConfig.emailFrom,
        APP_EMAIL_PROVIDER: "ses",
        APP_EMAIL_VERIFICATION_TTL_HOURS: "24",
        APP_EMAIL_VERIFICATION_BASE_URL: `${stageConfig.siteBaseUrl}/`,
        APP_LOGIN_RATE_LIMIT_PER_MINUTE: "10",
        APP_PASSWORD_RESET_BASE_URL: `${stageConfig.siteBaseUrl}/`,
        APP_SESSION_COOKIE_NAME: "mgt_session",
        APP_SESSION_TTL_HOURS: "720",
        APP_SES_REGION: stageConfig.awsRegion,
        APP_SIGNUP_DISABLED: String(stageConfig.signupDisabled),
        APP_SIGNUP_RATE_LIMIT_PER_MINUTE: "5",
        DATABASE_HOST: database.instanceEndpoint.hostname,
        DATABASE_NAME: "monthly_goal_tracker",
        DATABASE_SECRET_ARN: databaseCredentials.secretArn,
        DATABASE_SSLMODE: "require",
      },
    });

    const migrationFunction = new lambda.DockerImageFunction(this, "MigrationFunction", {
      code: lambda.DockerImageCode.fromImageAsset("../backend", {
        file: "Dockerfile.lambda",
        platform: ecrAssets.Platform.LINUX_ARM64,
        buildArgs: {
          APP_COMMAND: "migrate",
        },
      }),
      architecture: lambda.Architecture.ARM_64,
      memorySize: 512,
      timeout: Duration.seconds(120),
      reservedConcurrentExecutions: 1,
      logGroup: migrationLogGroup,
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
      securityGroups: [lambdaSecurityGroup],
      environment: {
        DATABASE_HOST: database.instanceEndpoint.hostname,
        DATABASE_NAME: "monthly_goal_tracker",
        DATABASE_SECRET_ARN: databaseCredentials.secretArn,
        DATABASE_SSLMODE: "require",
      },
    });
    const migrationFunctionResource = migrationFunction.node.defaultChild as lambda.CfnFunction;
    migrationFunctionResource.overrideLogicalId("MigrationFunction");

    const auditFunction = new lambda.DockerImageFunction(this, "AuditFunction", {
      code: lambda.DockerImageCode.fromImageAsset("../backend", {
        file: "Dockerfile.lambda",
        platform: ecrAssets.Platform.LINUX_ARM64,
        buildArgs: {
          APP_COMMAND: "audit",
        },
      }),
      architecture: lambda.Architecture.ARM_64,
      memorySize: 512,
      timeout: Duration.seconds(60),
      reservedConcurrentExecutions: 1,
      logGroup: auditLogGroup,
      vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
      securityGroups: [lambdaSecurityGroup],
      environment: {
        DATABASE_HOST: database.instanceEndpoint.hostname,
        DATABASE_NAME: "monthly_goal_tracker",
        DATABASE_SECRET_ARN: databaseCredentials.secretArn,
        DATABASE_SSLMODE: "require",
      },
    });
    const auditFunctionResource = auditFunction.node.defaultChild as lambda.CfnFunction;
    auditFunctionResource.overrideLogicalId("AuditFunction");

    databaseCredentials.grantRead(apiFunction);
    databaseCredentials.grantRead(migrationFunction);
    databaseCredentials.grantRead(auditFunction);
    apiFunction.addToRolePolicy(
      new iam.PolicyStatement({
        actions: ["ses:SendEmail"],
        resources: [`arn:${Aws.PARTITION}:ses:${stageConfig.awsRegion}:${Aws.ACCOUNT_ID}:identity/*`],
        conditions: {
          StringEquals: {
            "ses:FromAddress": stageConfig.emailFrom,
          },
        },
      }),
    );

    const apiIntegration = new integrations.HttpLambdaIntegration("ApiIntegration", apiFunction);
    const httpApi = new apigatewayv2.HttpApi(this, "HttpApi", {
      createDefaultStage: false,
      disableExecuteApiEndpoint: false,
    });
    addContractRoutes(httpApi, apiIntegration);
    new apigatewayv2.HttpStage(this, "HttpDefaultStage", {
      httpApi,
      stageName: "$default",
      autoDeploy: true,
      throttle: {
        burstLimit: 20,
        rateLimit: 10,
      },
    });

    const frontendBucket = new s3.Bucket(this, "FrontendBucket", {
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      enforceSSL: true,
      removalPolicy: removalPolicyFor(stageConfig),
      versioned: stageConfig.stage === "production",
    });
    const frontendBucketResource = frontendBucket.node.defaultChild as s3.CfnBucket;
    frontendBucketResource.overrideLogicalId("FrontendBucket");

    const spaRewriteFunction = new cloudfront.Function(this, "SpaRewriteFunction", {
      code: cloudfront.FunctionCode.fromInline(`
function handler(event) {
  var request = event.request;
  var uri = request.uri;
  if (uri.endsWith("/") || uri.indexOf(".") === -1) {
    request.uri = "/index.html";
  }
  return request;
}
`),
    });

    const apiOrigin = new origins.HttpOrigin(`${httpApi.apiId}.execute-api.${stageConfig.awsRegion}.${Aws.URL_SUFFIX}`, {
      protocolPolicy: cloudfront.OriginProtocolPolicy.HTTPS_ONLY,
      readTimeout: Duration.seconds(30),
    });
    const apiBehavior: cloudfront.BehaviorOptions = {
      origin: apiOrigin,
      allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
      cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
      originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
      viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
    };

    const distribution = new cloudfront.Distribution(this, "FrontendDistribution", {
      comment: `Monthly Goal Tracker ${stageConfig.stage} frontend and API`,
      defaultRootObject: "index.html",
      ...(stageConfig.domainName === undefined || stageConfig.certificateArn === undefined
        ? {}
        : {
            certificate: acm.Certificate.fromCertificateArn(this, "FrontendCertificate", stageConfig.certificateArn),
            domainNames: [stageConfig.domainName],
          }),
      defaultBehavior: {
        origin: origins.S3BucketOrigin.withOriginAccessControl(frontendBucket),
        cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
        functionAssociations: [
          {
            eventType: cloudfront.FunctionEventType.VIEWER_REQUEST,
            function: spaRewriteFunction,
          },
        ],
        responseHeadersPolicy: cloudfront.ResponseHeadersPolicy.SECURITY_HEADERS,
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
      },
      additionalBehaviors: {
        "/api": apiBehavior,
        "/api/*": apiBehavior,
      },
      priceClass: cloudfront.PriceClass.PRICE_CLASS_100,
    });
    const distributionResource = distribution.node.defaultChild as cloudfront.CfnDistribution;
    distributionResource.overrideLogicalId("FrontendDistribution");

    new CfnOutput(this, "Stage", {
      value: stageConfig.stage,
      description: "Deployment stage for this serverless stack.",
    });

    new CfnOutput(this, "TargetArchitecture", {
      value: "CloudFront/S3 frontend, API Gateway HTTP API, Go Lambda, direct RDS PostgreSQL, Secrets Manager, SES",
      description: "Approved target architecture baseline.",
    });

    new CfnOutput(this, "PublicApiPathCount", {
      value: String(publicApiPaths.length),
      description: "Count of public API route contracts preserved by the migration.",
    });

    new CfnOutput(this, "ProtectedApiPathCount", {
      value: String(protectedApiPaths.length),
      description: "Count of protected API route contracts preserved by the migration.",
    });

    new CfnOutput(this, "ServerlessInvariantCount", {
      value: String(serverlessInvariants.length),
      description: "Count of baseline invariants for the serverless migration.",
    });

    new CfnOutput(this, "DatabaseEngine", {
      value: "postgresql-17",
      description: "PostgreSQL engine family used by the serverless migration.",
    });

    new CfnOutput(this, "DatabaseConnectionMode", {
      value: "direct-rds",
      description: "Lambda connects directly to private RDS PostgreSQL with bounded concurrency.",
    });

    new CfnOutput(this, "HttpApiEnabled", {
      value: "true",
      description: "HTTP API is enabled for /api and /api/* routing through CloudFront.",
    });

    new CfnOutput(this, "MigrationLambdaEnabled", {
      value: "true",
      description: "Private migration Lambda is enabled for explicit database migration runs.",
    });

    new CfnOutput(this, "AuditLambdaEnabled", {
      value: "true",
      description: "Private audit Lambda is enabled for explicit migration rehearsal checks.",
    });

    new CfnOutput(this, "FrontendStaticHostingEnabled", {
      value: "true",
      description: "Private S3 static frontend origin is enabled behind CloudFront.",
    });

    new CfnOutput(this, "CloudFrontRoutingEnabled", {
      value: "true",
      description: "CloudFront routes frontend assets and /api plus /api/* API requests under one site boundary.",
    });

    new CfnOutput(this, "CustomDomainEnabled", {
      value: stageConfig.domainName === undefined ? "false" : "true",
      description: "Whether CloudFront custom domain alias support is enabled for this stack.",
    });
  }
}

function removalPolicyFor(stageConfig: StageConfig): RemovalPolicy {
  return stageConfig.removalPolicy === "destroy" ? RemovalPolicy.DESTROY : RemovalPolicy.RETAIN;
}

function addContractRoutes(httpApi: apigatewayv2.HttpApi, integration: integrations.HttpLambdaIntegration): void {
  for (const contract of [...publicApiPaths, ...protectedApiPaths]) {
    const [rawMethod, rawPath] = contract.split(" ");
    const method = httpMethod(rawMethod);
    httpApi.addRoutes({
      path: rawPath.replace(/:([A-Za-z0-9_]+)/g, "{$1}"),
      methods: [method],
      integration,
    });
  }
}

function httpMethod(rawMethod: string): apigatewayv2.HttpMethod {
  const methods: Record<string, apigatewayv2.HttpMethod> = {
    DELETE: apigatewayv2.HttpMethod.DELETE,
    GET: apigatewayv2.HttpMethod.GET,
    PATCH: apigatewayv2.HttpMethod.PATCH,
    POST: apigatewayv2.HttpMethod.POST,
    PUT: apigatewayv2.HttpMethod.PUT,
  };

  const method = methods[rawMethod];
  if (method === undefined) {
    throw new Error(`Unsupported HTTP method '${rawMethod}' in serverless API contract.`);
  }

  return method;
}
