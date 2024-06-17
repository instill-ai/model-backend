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

export function DeployUndeployModel(header) {
  // Model Backend API: load model online
  {
    group("Model Backend API: Load model online", function () {
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
        "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
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

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task cls response mode ID`]: (r) =>
          r.json().model_id === model_id
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, header), {
        [`GET /v1alpha/models/${model_id} online task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id} online task cls response state`]: (r) =>
          r.json().model.state === "STATE_ONLINE",
      })

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

      check(http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header), {
        [`GET /v1alpha/models/${model_id} online task cls watch status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id} online task cls watch state`]: (r) =>
          r.json().state === "STATE_UNSPECIFIED",
      })

      // Check delete model with 422 when model is in unspecified state
      // check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
      //   [`DELETE /v1alpha/models/${model_id} task cls response status 422`]: (r) =>
      //     r.status === 422,
      // })

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/undeploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/undeploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/undeploy online task cls response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, header), {
        [`GET /v1alpha/models/${model_id} online task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id} online task cls response state`]: (r) =>
          r.json().model.state === "STATE_OFFLINE",
      })

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_OFFLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task cls response model ID`]: (r) =>
          r.json().model_id === model_id
      });

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

      check(http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header), {
        [`GET /v1alpha/models/${model_id} online task cls watch status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id} online task cls watch state`]: (r) =>
          r.json().state === "STATE_UNSPECIFIED",
      })

      // Check delete model with 422 when model is in unspecified state
      // check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
      //   [`DELETE /v1alpha/models/${model_id} task cls response status 422`]: (r) =>
      //     r.status === 422,
      // })

      // Check the model state being updated in 120 secs
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
