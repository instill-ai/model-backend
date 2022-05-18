import http from "k6/http";
import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import * as createModel from "./rest_create_model.js"
import * as queryModel from "./rest_query_model.js"
import * as inferModel from "./rest_infer_model.js"
import * as deployModel from "./rest_deploy_model.js"
import * as publishModel from "./rest_publish_model.js"
import * as updateModel from "./rest_update_model.js"
import * as queryModelDefinition from "./rest_query_model_definition.js"
import * as queryModelInstance from "./rest_query_model_instance.js"

import {
  genHeader,
  base64_image,
} from "./helpers.js";

const apiHost = "http://localhost:8083";
const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const model_def_name = "model-definitions/github"
const model_def_uid = "909c3278-f7d1-461c-9352-87741bef11d3"

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
  
  // // Query Model API
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
  // queryModelDefinition.GetModelDefinition()
  queryModelDefinition.ListModelDefinition()

  // Query Model Instance API
  queryModelInstance.GetModelInstance()
  queryModelInstance.ListModelInstance()
  queryModelInstance.LookupModelInstance()
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
