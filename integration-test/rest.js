import http from "k6/http";
import { check, group } from "k6";

import * as createModel from "./rest_create_model.js"
import * as queryModel from "./rest_query_model.js"
import * as inferModel from "./rest_infer_model.js"
import * as deployModel from "./rest_deploy_model.js"
import * as publishModel from "./rest_publish_model.js"
import * as updateModel from "./rest_update_model.js"
import * as queryModelDefinition from "./rest_query_model_definition.js"
import * as queryModelInstance from "./rest_query_model_instance.js"
import * as getModelCard from "./rest_model_card.js"

import {
  genHeader,
} from "./helpers.js";

const apiHost = "http://model-backend:8083";

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
    group("Model API: __liveness check", () => {
      check(http.request("GET", `${apiHost}/v1alpha/__liveness`), {
        "GET /__liveness response status is 200": (r) => r.status === 200,
      });
      check(http.request("GET", `${apiHost}/v1alpha/__readiness`), {
        "GET /__readiness response status is 200": (r) => r.status === 200,
      });
    });
  }


  // Create Model API
  createModel.CreateModelFromLocal()
  createModel.CreateModelFromGitHub()

  // Query Model API
  queryModel.GetModel()
  queryModel.ListModel()
  queryModel.LookupModel()

  // Deploy/Undeploy Model API
  deployModel.DeployUndeployModel()

  // Infer Model API
  inferModel.InferModel()

  // Publish/Unpublish Model API
  publishModel.PublishUnpublishModel()

  // Update Model API
  updateModel.UpdateModel()

  // Query Model Definition API
  queryModelDefinition.GetModelDefinition()
  queryModelDefinition.ListModelDefinition()

  // Query Model Instance API
  queryModelInstance.GetModelInstance()
  queryModelInstance.ListModelInstance()
  queryModelInstance.LookupModelInstance()

  // Get model card
  getModelCard.GetModelCard()
}

export function teardown(data) {
  group("Model API: Delete all models created by this test", () => {
    for (const model of http
      .request("GET", `${apiHost}/v1alpha/models`, null, {
        headers: genHeader(
          "application/json"
        ),
      })
      .json("models")) {
      check(model, {
        "GET /clients response contents[*] id": (c) => c.id !== undefined,
      });
      check(
        http.request("DELETE", `${apiHost}/v1alpha/models/${model.id}`, null, {
          headers: genHeader("application/json"),
        }),
        {
          [`DELETE /v1alpha/models/${model.id} response status is 200`]: (r) =>
            r.status === 204,
        }
      );
    }
  });
}
