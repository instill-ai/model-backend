import http, { head } from "k6/http";
import {
  check,
  group
} from "k6";

import * as createModel from "./rest_create_model.js"
import * as queryModel from "./rest_query_model.js"
import * as queryModelPrivate from "./rest_query_model_private.js"
import * as deployModelPrivate from "./rest_deploy_model_private.js"
import * as inferModel from "./rest_infer_model.js"
import * as deployModel from "./rest_deploy_model.js"
import * as publishModel from "./rest_publish_model.js"
import * as updateModel from "./rest_update_model.js"
import * as queryModelDefinition from "./rest_query_model_definition.js"
import * as getModelCard from "./rest_model_card.js"
import * as longrunningOperation from "./rest_longrunning_operation.js"

import {
  genHeader,
} from "./helpers.js";

import * as constant from "./const.js"

export let options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
  scenarios: {
    contacts: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '20m',
    },
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
      check(http.request("GET", `${constant.apiPublicHost}/v1alpha/health/model`, null, header), {
        "GET /v1alpha/health/model response status is 200": (r) => r.status === 200,
      });
    });
  }

  // Query Model API by admin
  if (!constant.apiGatewayMode) {
    queryModelPrivate.ListModelsAdmin(header)
    queryModelPrivate.LookupModelAdmin(header)
    // private deploy will be trigger by public deploy
    // deployModelPrivate.DeployUndeployModel()
  }

  // Infer Model API
  inferModel.InferModel(header)

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

  // Query Model Definition API
  queryModelDefinition.GetModelDefinition(header)
  queryModelDefinition.ListModelDefinitions(header)

  // Get model card
  getModelCard.GetModelCard(header)

  // Long-running Operation
  longrunningOperation.GetLongRunningOperation(header)
}

export function teardown(header) {
  group("Model API: Delete all models created by this test", () => {
    for (const model of http
      .request("GET", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, null, header)
      .json("models")) {
      check(model, {
        "GET /models response contents[*] id": (c) => c.id !== undefined,
      });
      check(
        http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model.id}`, null, header), {
        [`DELETE /v1alpha/models/${model.id} response status is 204`]: (r) =>
          r.status === 204,
      }
      );
    }
  });
}
