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
  base64_image,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"


export function GetLongRunningOperation(header) {
  // Model Backend API: Predict Model with classification model
  {
    group("Model Backend API: Get LongRunning Operation", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("modelDefinition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`, header.headers.Authorization),
      })
      check(createClsModelRes, {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // TODO: public endpoint of deploy/undeploy is not longrunning anymore, need test revise

      let operationRes = http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, {
        headers: genHeader(`application/json`, header.headers.Authorization),
      })
      check(operationRes, {
        [`POST /v1alpha/models/${model_id}/deploy online task semantic response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      sleep(1) // take time to execute in Temporal

      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_UNSPECIFIED") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // check(http.get(`${constant.apiPublicHost}/v1alpha/${operationRes.json().operation.name}`, {}, {
      //   headers: genHeader(`application/json`, header.headers.Authorization),
      // }), {
      //   [`GET v1alpha/${operationRes.json().operation.name} response status`]: (r) =>
      //     r.status === 200,
      //   [`GET v1alpha/${operationRes.json().operation.name} response operation.name`]: (r) =>
      //     r.json().operation.name !== undefined,
      //   [`GET v1alpha/${operationRes.json().operation.name} response operation.metadata`]: (r) =>
      //     r.json().operation.metadata === null,
      //   [`GET v1alpha/${operationRes.json().operation.name} response operation.done`]: (r) =>
      //     r.json().operation.done !== undefined,
      // });

      // model can only be deleted after operation done
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }
}
