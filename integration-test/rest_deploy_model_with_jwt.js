import http from "k6/http";
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
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
  genHeader, genHeaderwithJwtSub,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"

export function DeployUndeployModel() {
  // Model Backend API: load model online
  let resp = http.request("GET", `${constant.mgmtApiPrivateHost}/v1alpha/admin/users/${constant.defaultUserId}`, {}, {
    headers: genHeader(`application/json`),
  })
  let userUid = resp.json().user.uid

  {
    group(`Model Backend API: Load model online [with random "jwt-sub" header]`, function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      fd_cls.append("id", model_id);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd_cls.boundary}`, userUid),
      })
      check(createClsModelRes, {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}/watch`, {
          headers: genHeaderwithJwtSub(`application/json`, userUid),
        })
        if (res.json().state === "STATE_OFFLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/deploy`, {}, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/deploy online task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/deploy`, {}, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/deploy online task cls response status 200`]: (r) =>
          r.status === 200,
      });

      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}/watch`, {
          headers: genHeaderwithJwtSub(`application/json`, userUid),
        })
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/undeploy`, {}, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/undeploy online task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/undeploy`, {}, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/undeploy online task cls response status 200`]: (r) =>
          r.status === 200,
      });

      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}/watch`, {
          headers: genHeaderwithJwtSub(`application/json`, userUid),
        })
        if (res.json().state === "STATE_OFFLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] DELETE clean up response status`]: (r) =>
          r.status === 204
      });
    });
  }
}
