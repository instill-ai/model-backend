import http from "k6/http";
import { check, group, sleep } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
  base64_image,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"


export function GetLongRunningOperation() {
  // Model Backend API: Predict Model with classification model
  {
    group("Model Backend API: Get LongRunning Operation", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
      });

      let operationRes = http.post(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest/deploy`, {}, {
        headers: genHeader(`application/json`),
      })
      check(operationRes, {
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.done`]: (r) =>
          r.json().operation.done === false,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.response`]: (r) =>
          r.json().operation.response !== undefined,
      });

      sleep(1) // take time to execute in Temporal
      check(http.get(`${constant.apiHost}/v1alpha/${operationRes.json().operation.name}`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`GET v1alpha/${operationRes.json().operation.name} response status`]: (r) =>
          r.status === 200,
        [`GET v1alpha/${operationRes.json().operation.name} response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`GET v1alpha/${operationRes.json().operation.name} response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`GET v1alpha/${operationRes.json().operation.name} response operation.done`]: (r) =>
          r.json().operation.done !== undefined,
        [`GET v1alpha/${operationRes.json().operation.name} response operation.response`]: (r) =>
          r.json().operation.response !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }
}

export function ListLongRunningOperation() {
  // Model Backend API: Predict Model with classification model
  {
    group("Model Backend API: List LongRunning Operation", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201
      });

      let operationRes = http.post(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest/deploy`, {}, {
        headers: genHeader(`application/json`),
      })
      check(operationRes, {
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.done`]: (r) =>
          r.json().operation.done === false,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.response`]: (r) =>
          r.json().operation.response !== undefined,
      });

      let listRes = http.get(`${constant.apiHost}/v1alpha/operations`, {}, {
        headers: genHeader(`application/json`),
      })
      check(listRes, {
        [`GET ${constant.apiHost}/v1alpha/operations response status`]: (r) =>
          r.status === 200,
        [`GET ${constant.apiHost}/v1alpha/operations response operations.length`]: (r) =>
          r.json().operations.length >= 1,          
      });

      // clean up
      check(http.request("DELETE", `${constant.apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }
}

export function CancelLongRunningOperation() {
  // Model Backend API: Predict Model with classification model
  {
    group("Model Backend API: Cancel LongRunning Operation", function () {
      let model_id = randomString(10)

      check(http.request("POST", `${constant.apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-yolov4"
        },
      }), {
        headers: genHeader("application/json"),
      }), {
        [`POST /v1alpha/models response status`]: (r) =>
          r.status == 201,
      });

      let operationRes = http.post(`${constant.apiHost}/v1alpha/models/${model_id}/instances/v1.0-cpu/deploy`, {}, {
        headers: genHeader(`application/json`),
      })
      check(operationRes, {
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.done`]: (r) =>
          r.json().operation.done === false,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.response`]: (r) =>
          r.json().operation.response !== undefined,
      });

      sleep(1)
      check(http.post(`${constant.apiHost}/v1alpha/${operationRes.json().operation.name}/cancel`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/${operationRes.json().operation.name}/cancel response status`]: (r) =>
          r.status === 200
      });

      check(http.get(`${constant.apiHost}/v1alpha/${operationRes.json().operation.name}`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST v1alpha/${operationRes.json().operation.name} response status`]: (r) =>
          r.status === 200,
        [`POST v1alpha/${operationRes.json().operation.name} response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`POST v1alpha/${operationRes.json().operation.metadata} response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`POST v1alpha/${operationRes.json().operation.done} response operation.done`]: (r) =>
          r.json().operation.done !== undefined,
        [`POST v1alpha/${operationRes.json().operation.done} response operation.response`]: (r) =>
          r.json().operation.response !== undefined,
      });      

      // clean up
      check(http.request("DELETE", `${constant.apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }
}
