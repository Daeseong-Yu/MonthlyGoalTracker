#!/usr/bin/env node
import { App } from "aws-cdk-lib";

import { ServerlessAppStack } from "../lib/serverless-app-stack.js";
import { stageConfigFromContext } from "../lib/stage-config.js";

const app = new App();
const stageConfig = stageConfigFromContext(app);

new ServerlessAppStack(app, `MonthlyGoalTracker-${stageConfig.stage}`, {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: stageConfig.awsRegion,
  },
  stageConfig,
});
