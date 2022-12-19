import http from "k6/http";
import { check, group, sleep } from "k6";
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
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.done`]: (r) =>
          r.json().operation.done === false,
        [`POST /v1alpha/models/${model_id}/instances/latest/deploy online task cls response operation.response`]: (r) =>
          r.json().operation.response !== undefined,
      });

      // Check the model instance state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
          var res = http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest`, {
            headers: genHeader(`application/json`),
          })
          if (res.json().instance.state === "STATE_ONLINE") {
              break
          }
          sleep(1)
          currentTime = new Date().getTime();
      }      

      check(http.post(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest/undeploy`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response operation.name`]: (r) =>
          r.json().operation.name !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response operation.metadata`]: (r) =>
          r.json().operation.metadata === null,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response operation.done`]: (r) =>
          r.json().operation.done === false,
        [`POST /v1alpha/models/${model_id}/instances/latest/undeploy online task cls response operation.response`]: (r) =>
          r.json().operation.response !== undefined,          
      });

      // Check the model instance state being updated in 120 secs
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
          var res = http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest`, {
            headers: genHeader(`application/json`),
          })
          if (res.json().instance.state === "STATE_OFFLINE") {
              break
          }
          sleep(1)
          currentTime = new Date().getTime();
      }        

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
