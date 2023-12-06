import http from "k6/http";
import {
  check,
  group
} from "k6";

import * as createModel from "./rest_create_model_with_jwt.js"
import * as queryModel from "./rest_query_model_with_jwt.js"
import * as testModel from "./rest_infer_model_with_jwt.js"
import * as deployModel from "./rest_deploy_model_with_jwt.js"
import * as publishModel from "./rest_publish_model_with_jwt.js"
import * as updateModel from "./rest_update_model_with_jwt.js"
import * as getModelCard from "./rest_model_card_with_jwt.js"

import * as constant from "./const.js"

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
  var loginResp = http.request("POST", `${constant.mgmtPublicHost}/v1beta/auth/login`, JSON.stringify({
    "username": constant.defaultUserId,
    "password": constant.defaultPassword,
  }))

  check(loginResp, {
    [`POST ${constant.mgmtPublicHost}/v1beta/auth/login response status is 200`]: (
      r
    ) => r.status === 200,
  });

  var header = {
    "headers": {
      "Authorization": `Bearer ${loginResp.json().access_token}`
    },
    "timeout": "600s",
  }
  return header
}

export default function (header) {
  /*
   * Model API - API CALLS
   */

  // Health check
  {
    group("Model API: health check", () => {
      check(http.request("GET", `${constant.apiPublicHost}/v1alpha/health/model`), {
        "GET /v1alpha/health/model response status is 200": (r) => r.status === 200,
      });
    });
  }

  if (!constant.apiGatewayMode) {
    // Test Model API
    testModel.TestModel(header)

    // Create Model API
    createModel.CreateModelFromLocal(header)
    createModel.CreateModelFromGitHub(header)

    // Query Model API
    queryModel.GetModel(header)
    queryModel.ListModels(header)
    queryModel.LookupModel(header)

    // Deploy/Undeploy Model API
    deployModel.DeployUndeployModel(header)

    // Publish/Unpublish Model API
    publishModel.PublishUnpublishModel(header)

    // Update Model API
    updateModel.UpdateModel(header)

    // Get model card
    getModelCard.GetModelCard(header)
  }
}

export function teardown(header) {}
