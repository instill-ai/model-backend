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
  isValidOwner,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"


export function ListModelsAdmin(header) {
  // Model Backend API: Get model list by admin
  {
    group("Model Backend API: Get model list by admin", function () {
      let model_id_1 = randomString(10)
      let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id_1,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-dummy-cls",
          "tag": "v1.0-cpu"
        }
      }), header)
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

      let model_id_2 = randomString(10)
      createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id_2,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-dummy-cls",
          "tag": "v1.0-cpu"
        }
      }), header)
      check(createClsModelRes, {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }
      let resp = http.get(`${constant.apiPrivateHost}/v1alpha/admin/models?page_size=1`, header)
      check(resp, {
        [`GET /v1alpha/admin/models task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/models task cls response total_size`]: (r) =>
          r.json().total_size >= 2,
        [`GET /v1alpha/admin/models task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/admin/models task cls response models.length`]: (r) =>
          r.json().models.length === 1,
        [`GET /v1alpha/admin/models task cls response models[0].name`]: (r) =>
          r.json().models[0].name === `${constant.namespace}/models/${model_id_2}`,
        [`GET /v1alpha/admin/models task cls response models[0].uid`]: (r) =>
          r.json().models[0].uid !== undefined,
        [`GET /v1alpha/admin/models task cls response models[0].id`]: (r) =>
          r.json().models[0].id === model_id_2,
        [`GET /v1alpha/admin/models task cls response models[0].description`]: (r) =>
          r.json().models[0].description !== undefined,
        [`GET /v1alpha/admin/models task cls response models[0].task`]: (r) =>
          r.json().models[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/admin/models task cls response models[0].state`]: (r) =>
          r.json().models[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/admin/models task cls response models[0].model_definition`]: (r) =>
          r.json().models[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/admin/models task cls response models[0].configuration`]: (r) =>
          r.json().models[0].configuration === null,
        [`GET /v1alpha/admin/models task cls response models[0].visibility`]: (r) =>
          r.json().models[0].visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/admin/models task cls response models[0].owner_name`]: (r) =>
          isValidOwner(r.json().models[0].owner_name),
        [`GET /v1alpha/admin/models task cls response models[0].create_time`]: (r) =>
          r.json().models[0].create_time !== undefined,
        [`GET /v1alpha/admin/models task cls response models[0].update_time`]: (r) =>
          r.json().models[0].update_time !== undefined,
      });

      check(http.get(`${constant.apiPrivateHost}/v1alpha/admin/models?page_size=1&page_token=${resp.json().next_page_token}`, header), {
        [`GET /v1alpha/admin/models task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/models task cls response total_size`]: (r) =>
          r.json().total_size >= 2,
        [`GET /v1alpha/admin/models task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/admin/models task cls response models.length`]: (r) =>
          r.json().models.length === 1,
        [`GET /v1alpha/admin/models task cls response models[0].name`]: (r) =>
          r.json().models[0].name === `${constant.namespace}/models/${model_id_1}`,
        [`GET /v1alpha/admin/models task cls response models[0].uid`]: (r) =>
          r.json().models[0].uid !== undefined,
        [`GET /v1alpha/admin/models task cls response models[0].id`]: (r) =>
          r.json().models[0].id === model_id_1,
        [`GET /v1alpha/admin/models task cls response models[0].description`]: (r) =>
          r.json().models[0].description !== undefined,
        [`GET /v1alpha/admin/models task cls response models[0].task`]: (r) =>
          r.json().models[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/admin/models task cls response models[0].state`]: (r) =>
          r.json().models[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/admin/models task cls response models[0].model_definition`]: (r) =>
          r.json().models[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/admin/models task cls response models[0].configuration`]: (r) =>
          r.json().models[0].configuration === null,
        [`GET /v1alpha/admin/models task cls response models[0].visibility`]: (r) =>
          r.json().models[0].visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/admin/models task cls response models[0].owner_name`]: (r) =>
          isValidOwner(r.json().models[0].owner_name),
        [`GET /v1alpha/admin/models task cls response models[0].create_time`]: (r) =>
          r.json().models[0].create_time !== undefined,
        [`GET /v1alpha/admin/models task cls response models[0].update_time`]: (r) =>
          r.json().models[0].update_time !== undefined,
      });

      check(http.get(`${constant.apiPrivateHost}/v1alpha/admin/models?view=VIEW_FULL`, header), {
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response total_size`]: (r) =>
          r.json().total_size == 2,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models.length`]: (r) =>
          r.json().models.length === 2,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].name`]: (r) =>
          r.json().models[0].name === `${constant.namespace}/models/${model_id_2}`,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].uid`]: (r) =>
          r.json().models[0].uid !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].id`]: (r) =>
          r.json().models[0].id === model_id_2,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].description`]: (r) =>
          r.json().models[0].description !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].task`]: (r) =>
          r.json().models[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].state`]: (r) =>
          r.json().models[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].model_definition`]: (r) =>
          r.json().models[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].configuration.repository`]: (r) =>
          r.json().models[0].configuration.repository === "admin/model-dummy-cls",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].configuration.html_url`]: (r) =>
          r.json().models[0].configuration.html_url === "https://github.com/admin/model-dummy-cls",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].visibility`]: (r) =>
          r.json().models[0].visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].owner_name`]: (r) =>
          isValidOwner(r.json().models[0].owner_name),
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].create_time`]: (r) =>
          r.json().models[0].create_time !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[0].update_time`]: (r) =>
          r.json().models[0].update_time !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].name`]: (r) =>
          r.json().models[1].name === `${constant.namespace}/models/${model_id_1}`,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].uid`]: (r) =>
          r.json().models[1].uid !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].id`]: (r) =>
          r.json().models[1].id === model_id_1,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].description`]: (r) =>
          r.json().models[1].description !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].task`]: (r) =>
          r.json().models[1].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].state`]: (r) =>
          r.json().models[1].state === "STATE_OFFLINE",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].model_definition`]: (r) =>
          r.json().models[1].model_definition === "model-definitions/github",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].configuration.repository`]: (r) =>
          r.json().models[1].configuration.repository === "admin/model-dummy-cls",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].configuration.html_url`]: (r) =>
          r.json().models[1].configuration.html_url === "https://github.com/admin/model-dummy-cls",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].visibility`]: (r) =>
          r.json().models[1].visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].owner_name`]: (r) =>
          isValidOwner(r.json().models[1].owner_name),
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].create_time`]: (r) =>
          r.json().models[1].create_time !== undefined,
        [`GET /v1alpha/admin/models?view=VIEW_FULL task cls response models[1].update_time`]: (r) =>
          r.json().models[1].update_time !== undefined,
      });

      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res_1 = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id_1}/watch`, header)
        let res_2 = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id_2}/watch`, header)
        if (res_1.json().state !== "STATE_UNSPECIFIED" && res_2.json().state !== "STATE_UNSPECIFIED") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id_1}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id_2}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
}

export function LookupModelAdmin(header) {
  // Model Backend API: look up a model by admin
  {
    group("Model Backend API: Look up a model by admin", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
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
        let r = http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, header)
        if (r.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      let res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, header)
      let modelUid = res.json().model.uid

      check(http.get(`${constant.apiPrivateHost}/v1alpha/admin/models/${modelUid}/lookUp`, header), {
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.name`]: (r) =>
          r.json().model.name === `${constant.namespace}/models/${model_id}`,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.uid`]: (r) =>
          r.json().model.uid !== undefined,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.id`]: (r) =>
          r.json().model.id === model_id,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.description`]: (r) =>
          r.json().model.description === model_description,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.model_definition`]: (r) =>
          r.json().model.model_definition === model_def_name,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.task`]: (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.state`]: (r) =>
          r.json().model.state === "STATE_OFFLINE",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.configuration`]: (r) =>
          r.json().model.configuration === null,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.owner_name`]: (r) =>
          isValidOwner(r.json().model.owner_name),
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.create_time`]: (r) =>
          r.json().model.create_time !== undefined,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.update_time`]: (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${constant.apiPrivateHost}/v1alpha/admin/models/${modelUid}/lookUp?view=VIEW_FULL`, header), {
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.name`]: (r) =>
          r.json().model.name === `${constant.namespace}/models/${model_id}`,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.uid`]: (r) =>
          r.json().model.uid !== undefined,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.id`]: (r) =>
          r.json().model.id === model_id,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.description`]: (r) =>
          r.json().model.description === model_description,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.task`]: (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.state`]: (r) =>
          r.json().model.state === "STATE_OFFLINE",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.model_definition`]: (r) =>
          r.json().model.model_definition === model_def_name,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.configuration.content`]: (r) =>
          r.json().model.configuration.content === "dummy-cls-model.zip",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.owner_name`]: (r) =>
          isValidOwner(r.json().model.owner_name),
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.create_time`]: (r) =>
          r.json().model.create_time !== undefined,
        [`GET /v1alpha/admin/models/${modelUid}/lookUp task cls response model.update_time`]: (r) =>
          r.json().model.update_time !== undefined,
      });

      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state !== "STATE_UNSPECIFIED") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state !== "STATE_UNSPECIFIED") {
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
