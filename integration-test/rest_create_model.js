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


export function CreateModelFromLocal() {
  // Model Backend API: upload model
  {
    group("Model Backend API: CreateModelFromLocal", function () {
      let fd_cls = new FormData();
      let model_id_cls = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id_cls);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", "model-definitions/local");
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
      check(http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`), {
        [`GET v1alpha/${createClsModelRes.json().operation.name} task cls status`]: (r) => r.status === 200,
        [`GET v1alpha/${createClsModelRes.json().operation.name} task cls operation.done`]: (r) => r.json().operation.done === true,
      });

      let fd_det = new FormData();
      let model_id_det = randomString(10)
      model_description = randomString(20)
      fd_det.append("id", model_id_det);
      fd_det.append("description", model_description);
      fd_det.append("model_definition", "model-definitions/local");
      fd_det.append("content", http.file(constant.det_model, "dummy-det-model.zip"));
      let createDetModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      })
      check(createDetModelRes, {
        "POST /v1alpha/models/multipart task det response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task det response operation.name": (r) =>
          r.json().operation.name !== undefined
      });

      // Check model creation finished
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${createDetModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }
      check(http.get(`${constant.apiPublicHost}/v1alpha/${createDetModelRes.json().operation.name}`), {
        [`GET v1alpha/${createDetModelRes.json().operation.name} task det status`]: (r) => r.status === 200,
        [`GET v1alpha/${createDetModelRes.json().operation.name} task det operation.done`]: (r) => r.json().operation.done === true,
      });

      let fd_keypoint = new FormData();
      let model_id_keypoint = randomString(10)
      model_description = randomString(20)
      fd_keypoint.append("id", model_id_keypoint);
      fd_keypoint.append("description", model_description);
      fd_keypoint.append("model_definition", "model-definitions/local");
      fd_keypoint.append("content", http.file(constant.keypoint_model, "dummy-keypoint-model.zip"));
      let createKpModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_keypoint.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_keypoint.boundary}`),
      })
      check(createKpModelRes, {
        "POST /v1alpha/models/multipart task keypoint response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task keypoint response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${createKpModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }
      check(http.get(`${constant.apiPublicHost}/v1alpha/${createKpModelRes.json().operation.name}`), {
        [`GET v1alpha/${createKpModelRes.json().operation.name} task keypoint status`]: (r) => r.status === 200,
        [`GET v1alpha/${createKpModelRes.json().operation.name} task keypoint operation.done`]: (r) => r.json().operation.done === true,
      });

      let fd_unspecified = new FormData();
      let model_id_unspecified = randomString(10)
      model_description = randomString(20)
      fd_unspecified.append("id", model_id_unspecified);
      fd_unspecified.append("description", model_description);
      fd_unspecified.append("model_definition", "model-definitions/local");
      fd_unspecified.append("content", http.file(constant.unspecified_model, "dummy-unspecified-model.zip"));
      let createUnspecifiedModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      })
      check(createUnspecifiedModelRes, {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task unspecified response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${createUnspecifiedModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }
      check(http.get(`${constant.apiPublicHost}/v1alpha/${createUnspecifiedModelRes.json().operation.name}`), {
        [`GET v1alpha/${createUnspecifiedModelRes.json().operation.name} task unspecified status`]: (r) => r.status === 200,
        [`GET v1alpha/${createUnspecifiedModelRes.json().operation.name} task unspecified operation.done`]: (r) => r.json().operation.done === true,
      });

      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /v1alpha/models/multipart already existed response status 409": (r) =>
          r.status === 409,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_cls}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_det}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_keypoint}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_unspecified}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });

    group("Model Backend API: Upload a model which exceed max batch size limitation", function () {
      let fd_cls = new FormData();
      let model_id_cls = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id_cls);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", "model-definitions/local");
      fd_cls.append("content", http.file(constant.cls_model_bz17, "dummy-cls-model-bz17.zip"));
      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 400,
      });

      let fd_det = new FormData();
      let model_id_det = randomString(10)
      model_description = randomString(20)
      fd_det.append("id", model_id_det);
      fd_det.append("description", model_description);
      fd_det.append("model_definition", "model-definitions/local");
      fd_det.append("content", http.file(constant.det_model_bz9, "dummy-det-model-bz9.zip"));
      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task det response status": (r) =>
          r.status === 400,
      });

      let fd_keypoint = new FormData();
      let model_id_keypoint = randomString(10)
      model_description = randomString(20)
      fd_keypoint.append("id", model_id_keypoint);
      fd_keypoint.append("description", model_description);
      fd_keypoint.append("model_definition", "model-definitions/local");
      fd_keypoint.append("content", http.file(constant.keypoint_model_bz9, "dummy-keypoint-model-bz9.zip"));
      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_keypoint.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_keypoint.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task keypoint response status": (r) =>
          r.status === 400,
      });

      let fd_unspecified = new FormData();
      let model_id_unspecified = randomString(10)
      model_description = randomString(20)
      fd_unspecified.append("id", model_id_unspecified);
      fd_unspecified.append("description", model_description);
      fd_unspecified.append("model_definition", "model-definitions/local");
      fd_unspecified.append("content", http.file(constant.unspecified_model_bz3, "dummy-unspecified-model-bz3.zip"));
      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 400,
      });

      let fd_semantic = new FormData();
      let model_id_semantic = randomString(10)
      model_description = randomString(20)
      fd_semantic.append("id", model_id_semantic);
      fd_semantic.append("description", model_description);
      fd_semantic.append("model_definition", "model-definitions/local");
      fd_semantic.append("content", http.file(constant.semantic_segmentation_model_bz9, "dummy-semantic-segmentation-model-bz9.zip"));
      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_semantic.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_semantic.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 400,
      });

      let fd_instance = new FormData();
      let model_id_instance = randomString(10)
      model_description = randomString(20)
      fd_instance.append("id", model_id_instance);
      fd_instance.append("description", model_description);
      fd_instance.append("model_definition", "model-definitions/local");
      fd_instance.append("content", http.file(constant.instance_segmentation_model_bz9, "dummy-instance-segmentation-model-bz9.zip"));
      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_instance.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_instance.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 400,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_cls}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_det}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_keypoint}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });

      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/models/${model_id_unspecified}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });
    });
  }
}

export function CreateModelFromGitHub() {
  // Model Backend API: upload model by GitHub
  {
    group("Model Backend API: Upload a model by GitHub", function () {
      let model_id = randomString(10)
      let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-dummy-cls",
          "tag": "v1.0-cpu"
        },
      }), {
        headers: genHeader("application/json"),
      })

      check(createClsModelRes, {
        "POST /v1alpha/models task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models task cls response operation.name": (r) =>
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
      check(http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`), {
        [`GET v1alpha/${createClsModelRes.json().operation.name} task cls status`]: (r) => r.status === 200,
        [`GET v1alpha/${createClsModelRes.json().operation.name} task cls operation.done`]: (r) => r.json().operation.done === true,
      });

      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition": randomString(10),
        "configuration": {
          "repository": "instill-ai/model-dummy-cls"
        },
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github invalid url status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models`, JSON.stringify({
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-dummy-cls"
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github missing name status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${constant.apiPublicHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition": "model-definitions/github",
        "configuration": {}
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github missing github_url status": (r) =>
          r.status === 400,
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