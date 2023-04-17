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

export function setup() {}

export default function (data) {
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

  if (__ENV.MODE != "api-gateway" && __ENV.MODE != "localhost") {
    // Test Model API
    testModel.TestModel()

    // Create Model API
    createModel.CreateModelFromLocal()
    createModel.CreateModelFromGitHub()

    // Query Model API
    queryModel.GetModel()
    queryModel.ListModels()
    queryModel.LookupModel()

    // Deploy/Undeploy Model API
    deployModel.DeployUndeployModel()

    // Publish/Unpublish Model API
    publishModel.PublishUnpublishModel()

    // Update Model API
    updateModel.UpdateModel()

    // Get model card
    getModelCard.GetModelCard()
  }
}

export function teardown(data) {}
