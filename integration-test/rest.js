import http, { head } from "k6/http";
import {
  check,
  group
} from "k6";

import * as createModel from "./rest-create-model.js"
import * as queryModel from "./rest-query-model.js"
import * as queryModelPrivate from "./rest-query-model-private.js"
import * as deployModelPrivate from "./rest-deploy-model-private.js"
import * as inferModel from "./rest-infer-model.js"
import * as deployModel from "./rest-deploy-model.js"
import * as publishModel from "./rest-publish-model.js"
import * as updateModel from "./rest-update-model.js"
import * as queryModelDefinition from "./rest-query-model-definition.js"
import * as getModelCard from "./rest-model-card.js"
import * as longrunningOperation from "./rest-longrunning-operation.js"
import * as restInvariants from "./rest-invariants.js"

import {
  genHeader,
  getBasicAuthHeader,
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
  // CE uses Basic Auth (JWT auth is only available in EE)
  const authHeader = getBasicAuthHeader(constant.defaultUserId, constant.defaultPassword);

  var header = {
    "headers": {
      "Authorization": authHeader
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
      check(http.request("GET", `${constant.apiPublicHost}/v1beta/health/model`, null, header), {
        "GET /v1beta/health/model response status is 200": (r) => r.status === 200,
      });
    });
  }

  // Query Model API by admin
  // if (!constant.apiGatewayMode) {
  // queryModelPrivate.ListModelsAdmin(header)
  // queryModelPrivate.LookupModelAdmin(header)
  // private deploy will be trigger by public deploy
  // deployModelPrivate.DeployUndeployModel()
  // }

  // Infer Model API
  // inferModel.InferModel(header)

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

  // Query Model Definition API
  // queryModelDefinition.GetModelDefinition(header)
  // queryModelDefinition.ListModelDefinitions(header)

  // Get model card
  // getModelCard.GetModelCard(header)

  // Long-running Operation
  // longrunningOperation.GetLongRunningOperation(header)

  // AIP Resource Refactoring Invariants
  if (constant.apiGatewayMode) {
    restInvariants.checkInvariants(header);
  }
}

export function teardown(header) {
  group("Model API: Delete all models created by this test", () => {
    const models = http
      .request("GET", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, null, header)
      .json("models");

    if (!models) return;

    for (const model of models) {
      // Skip models without valid IDs
      if (!model || !model.id) continue;

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
