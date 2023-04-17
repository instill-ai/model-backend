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
  genHeader,
  base64_image,
  genHeaderwithJwtSub,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"


export function TestModel() {
  // Model Backend API: Predict Model with classification model
  // Model Backend API: load model online
  let resp = http.request("GET", `${constant.mgmtApiPrivateHost}/v1alpha/admin/users/${constant.defaultUserId}`, {}, {
    headers: genHeader(`application/json`),
  })
  let userUid = resp.json().user.uid

  let fd_cls = new FormData();
  let model_id = randomString(10)
  let model_description = randomString(20)
  fd_cls.append("id", model_id);
  fd_cls.append("description", model_description);
  fd_cls.append("model_definition", model_def_name);
  fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));

  {
    group(`Model Backend API: Predict Model with classification model [with random "jwt-sub" header]`, function () {
      http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd_cls.boundary}`, userUid),
      })

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

      http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/deploy`, {}, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      })

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

      // Predict with url
      let payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }]
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/test url cls status 401`]: (r) =>
          r.status === 401,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/test url cls status 200`]: (r) =>
          r.status === 200,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/tiff-sample.tiff"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/test url cls multiple images status 401`]: (r) =>
          r.status === 401,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/test url cls multiple images status 200`]: (r) =>
          r.status === 200,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        }]
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/test base64 cls status 401`]: (r) =>
          r.status === 401,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/test base64 cls status 200`]: (r) =>
          r.status === 200,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
          }
        }
        ]
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/test base64 cls multiple images status 401`]: (r) =>
          r.status === 401,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test`, payload, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/test base64 cls multiple images status 200`]: (r) =>
          r.status === 200,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd.boundary}`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/test-multipart cls status 401`]: (r) =>
          r.status === 401,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd.boundary}`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/test-multipart cls status 200`]: (r) =>
          r.status === 200,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd.boundary}`, uuidv4()),
      }), {
        [`[with random "jwt-sub" header] POST /v1alpha/models/${model_id}/test-multipart cls multiple images status 401`]: (r) =>
          r.status === 401,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeaderwithJwtSub(`multipart/form-data; boundary=${fd.boundary}`, userUid),
      }), {
        [`[with default "jwt-sub" header] POST /v1alpha/models/${model_id}/test-multipart cls multiple images status 200`]: (r) =>
          r.status === 200,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeaderwithJwtSub(`application/json`, userUid),
      }), {
        [`[with default "jwt-sub" header] DELETE clean up response status 204`]: (r) =>
          r.status === 204
      });

    });
  }
}
