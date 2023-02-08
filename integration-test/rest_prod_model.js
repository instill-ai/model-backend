import http from "k6/http";
import {
  check,
  group
} from "k6";

import * as inferModel from "./rest_infer_github_model.js"

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

export function setup() {}

export default function (data) {
  /*
   * Model API - API CALLS
   */

  // Infer Model API
  inferModel.InferGitHubModel()
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
        "GET /clients response contents[*] id": (c) => c.id !== undefined,
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