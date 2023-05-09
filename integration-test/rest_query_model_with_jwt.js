import http from "k6/http";
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import {
  check,
  group,
  sleep,
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

export function GetModel() {
  // Model Backend API: Get model info
  let resp = http.request("GET", `${constant.mgmtApiPrivateHost}/v1alpha/admin/users/${constant.defaultUserId}`, {}, {
    headers: genHeader(`application/json`),
  })
  let userUid = resp.json().user.uid

  let fd_cls = new FormData();
  let model_id = randomString(10)
  let model_description = randomString(20)
  fd_cls.append("id", model_id);
  fd_cls.append("description", model_description);
  fd_cls.append("model_definition", "model-definitions/local");
  fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));

  {
    let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
      headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd_cls.boundary}`, userUid),
    })

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

    group(`Model Backend API: Get model info [with "jwt-sub" header]`, function () {
      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] GET /v1alpha/models/${model_id} task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}?view=VIEW_FULL`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] GET /v1alpha/models/${model_id} task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] GET /v1alpha/models/${model_id} task cls response status 200`]: (r) =>
          r.status === 200,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}?view=VIEW_FULL`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] GET /v1alpha/models/${model_id} task cls response status 200`]: (r) =>
          r.status === 200,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] DELETE /v1alpha/models/${model_id} status 204`]: (r) =>
          r.status === 204
      });
    });
  }
}

export function ListModels() {
  // Model Backend API: Get model list
  let resp = http.request("GET", `${constant.mgmtApiPrivateHost}/v1alpha/admin/users/${constant.defaultUserId}`, {}, {
    headers: genHeader(`application/json`),
  })
  let userUid = resp.json().user.uid

  {
    group(`Model Backend API: Get model list [with "jwt-sub" header]`, function () {
      let resp = http.get(`${constant.apiPublicHost}/v1alpha/models?page_size=1`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      })
      check(resp, {
        [`[with random "jwt-sub" header] GET /v1alpha/models task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models?page_size=1&page_token=${resp.json().next_page_token}`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] GET /v1alpha/models task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models?view=VIEW_FULL`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] GET /v1alpha/models?view=VIEW_FULL task cls response status 404`]: (r) =>
          r.status === 404,
      });

      resp = http.get(`${constant.apiPublicHost}/v1alpha/models?page_size=1`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      })
      check(resp, {
        [`[with default "jwt-sub" header] GET /v1alpha/models task cls response status 200`]: (r) =>
          r.status === 200,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models?page_size=1&page_token=${resp.json().next_page_token}`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] GET /v1alpha/models task cls response status 200`]: (r) =>
          r.status === 200,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models?view=VIEW_FULL`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] GET /v1alpha/models?view=VIEW_FULL task cls response status 200`]: (r) =>
          r.status === 200,
      });
    });
  }
}

export function LookupModel() {
  // Model Backend API: look up model
  let resp = http.request("GET", `${constant.mgmtApiPrivateHost}/v1alpha/admin/users/${constant.defaultUserId}`, {}, {
    headers: genHeader(`application/json`),
  })
  let userUid = resp.json().user.uid

  let fd_cls = new FormData();
  let model_id = randomString(10)
  let model_description = randomString(20)
  fd_cls.append("id", model_id);
  fd_cls.append("description", model_description);
  fd_cls.append("model_definition", "model-definitions/local");
  fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));

  {
    let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
      headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd_cls.boundary}`, userUid),
    })

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

    resp = http.get(`${constant.apiPublicHost}/v1alpha/models/${model_id}`, {
      headers: genHeaderwithJwtSub(`application/json`, userUid),
    })
    let modelUid = resp.json().model.uid

    group(`Model Backend API: Look up model [with "jwt-sub" header]`, function () {
      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${modelUid}/lookUp`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] GET /v1alpha/models/${modelUid}/lookUp task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${modelUid}/lookUp?view=VIEW_FULL`, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] GET /v1alpha/models/${modelUid}/lookUp?view=VIEW_FULL task cls response status 404`]: (r) =>
          r.status === 404,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${modelUid}/lookUp`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] GET /v1alpha/models/${modelUid}/lookUp task cls response status 200`]: (r) =>
          r.status === 200,
      });

      check(http.get(`${constant.apiPublicHost}/v1alpha/models/${modelUid}/lookUp?view=VIEW_FULL`, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] GET /v1alpha/models/${modelUid}/lookUp?view=VIEW_FULL task cls response status 200`]: (r) =>
          r.status === 200,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] DELETE /v1alpha/models/${model_id} status 204`]: (r) =>
          r.status === 204
      });
    });
  }
}
