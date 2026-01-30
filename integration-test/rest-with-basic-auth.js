import http from "k6/http";
import encoding from "k6/encoding";
import {
  check,
  group
} from "k6";

import * as createModel from "./rest-create-model-with-basic-auth.js"
import * as queryModel from "./rest-query-model-with-basic-auth.js"
import * as deployModel from "./rest-deploy-model-with-basic-auth.js"
import * as publishModel from "./rest-publish-model-with-basic-auth.js"
import * as updateModel from "./rest-update-model-with-basic-auth.js"
import * as getModelCard from "./rest-model-card-with-basic-auth.js"

import * as constant from "./const.js"

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  // CE edition uses Basic Auth (JWT auth is only available in EE)
  const basicAuth = encoding.b64encode(`${constant.defaultUserId}:${constant.defaultPassword}`);

  var header = {
    "headers": {
      "Authorization": `Basic ${basicAuth}`,
      "Content-Type": "application/json",
    },
    "timeout": "600s",
  }
  return header
}

export default function (header) {
  /*
   * Model API - API CALLS
   */

  // Health check (via API Gateway: /v1beta/health/model -> /v1alpha/health/model)
  {
    group("Model API: health check", () => {
      check(http.request("GET", `${constant.apiPublicHost}/v1beta/health/model`), {
        "GET /v1beta/health/model response status is 200": (r) => r.status === 200,
      });
    });
  }

  // if (!constant.apiGatewayMode) {
  // Create Model API
  // createModel.CreateModelFromLocal(header)
  // createModel.CreateModelFromGitHub(header)

  // Query Model API
  // queryModel.GetModel(header)
  // queryModel.ListModels(header)
  // queryModel.LookupModel(header)

  // Deploy/Undeploy Model API
  // deployModel.DeployUndeployModel(header)

  // Publish/Unpublish Model API
  // publishModel.PublishUnpublishModel(header)

  // Update Model API
  // updateModel.UpdateModel(header)

  // Get model card
  // getModelCard.GetModelCard(header)
  // }
}

export function teardown(header) { }
