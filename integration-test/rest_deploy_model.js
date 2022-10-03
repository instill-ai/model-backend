import http from "k6/http";
import { check, group } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"

export function DeployUndeployModel() {
  // Model Backend API: load model online
  {
    group("Model Backend API: Load model online", function () {
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
        "POST /v1alpha/models/multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models/multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models/multipart task cls response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models/multipart task cls response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models/multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models/multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models/multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models/multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models/multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.post(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest/deploy`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });

      check(http.post(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest/undeploy`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
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
