import http from "k6/http";
import {
  check,
  group,
  sleep
} from "k6";

import {
  FormData
} from "https://jslib.k6.io/formdata/0.0.2/index.js";
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"

export function GetModelCard() {
  // Model Backend API: Get model card
  {
    group("Model Backend API: Get model card", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(createClsModelRes, {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}/readme`), {
        [`GET /v1alpha/models/${model_id}/readme response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/readme response readme.name`]: (r) =>
          r.json().readme.name === `models/${model_id}/readme`,
        [`GET /v1alpha/models/${model_id}/readme response readme.size`]: (r) =>
          r.json().readme.size !== undefined,
        [`GET /v1alpha/models/${model_id}/readme response readme.type`]: (r) =>
          r.json().readme.type === "file",
        [`GET /v1alpha/models/${model_id}/readme response readme.encoding`]: (r) =>
          r.json().readme.encoding === "base64",
        [`GET /v1alpha/models/${model_id}/readme response readme.content`]: (r) =>
          r.json().readme.content !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
  // Model Backend API: Get model card without readme
  {
    group("Model Backend API: Get model card without readme", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_no_readme_model, "dummy-cls-no-readme.zip"));
      let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(createClsModelRes, {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}/readme`), {
        [`GET /v1alpha/models/${model_id}/readme response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/readme no readme response readme.name`]: (r) =>
          r.json().readme.name === `models/${model_id}/readme`,
        [`GET /v1alpha/models/${model_id}/readme no readme response readme.size`]: (r) =>
          r.json().readme.size === 0,
        [`GET /v1alpha/models/${model_id}/readme no readme response readme.type`]: (r) =>
          r.json().readme.type === "file",
        [`GET /v1alpha/models/${model_id}/readme no readme response readme.encoding`]: (r) =>
          r.json().readme.encoding === "base64",
        [`GET /v1alpha/models/${model_id}/readme no readme response readme.content`]: (r) =>
          r.json().readme.content === "",
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
}
