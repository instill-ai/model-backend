import http from "k6/http";
import {
  check,
  group
} from "k6";

import * as createModel from "./rest_create_model.js"
import * as queryModel from "./rest_query_model.js"
import * as inferModel from "./rest_infer_model.js"
import * as deployModel from "./rest_deploy_model.js"
import * as publishModel from "./rest_publish_model.js"
import * as updateModel from "./rest_update_model.js"
import * as queryModelDefinition from "./rest_query_model_definition.js"
import * as queryModelInstance from "./rest_query_model_instance.js"
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
};

export function setup() { }

export default function (data) {
  /*
   * Model API - API CALLS
   */

  // Health check
  {
    group("Model API: health check", () => {
      check(http.request("GET", `${constant.apiHost}/v1alpha/health/model`), {
        "GET /v1alpha/health/model response status is 200": (r) => r.status === 200,
      });
    });
  }

  // Infer Model API
  inferModel.InferModel()

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

  // Query Model Definition API
  queryModelDefinition.GetModelDefinition()
  queryModelDefinition.ListModelDefinitions()

  // Query Model Instance API
  queryModelInstance.GetModelInstance()
  queryModelInstance.ListModelInstances()
  queryModelInstance.LookupModelInstance()

  // Get model card
  getModelCard.GetModelCard()

  // Long-running Operation
  longrunningOperation.GetLongRunningOperation()
  longrunningOperation.ListLongRunningOperation()
  longrunningOperation.CancelLongRunningOperation()
}

export function teardown(data) {
  group("Model API: Delete all models created by this test", () => {
    for (const model of http
      .request("GET", `${constant.apiHost}/v1alpha/models`, null, {
        headers: genHeader(
          "application/json"
        ),
      })
      .json("models")) {
      check(model, {
        "GET /models response contents[*] id": (c) => c.id !== undefined,
      });
      check(
        http.request("DELETE", `${constant.apiHost}/v1alpha/models/${model.id}`, null, {
          headers: genHeader("application/json"),
        }), {
        [`DELETE /v1alpha/models/${model.id} response status is 204`]: (r) =>
          r.status === 204,
      }
      );
    }
  });
}
