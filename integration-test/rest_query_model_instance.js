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

export function GetModelInstance() {
  // Model Backend API: Get model instance
  {
    group("Model Backend API: Get model instance", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      let createClsModelRes = http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
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
        let res = http.get(`${constant.apiHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration === null,
      });

      check(http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest task cls response instance.content`]: (r) =>
          r.json().instance.configuration.content === "dummy-cls-model.zip",
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

export function ListModelInstances() {
  // Model Backend API: Get model instance list - local model
  {
    group("Model Backend API: Get model instance list local model", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      let createClsModelRes = http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
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
        let res = http.get(`${constant.apiHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response total_size`]: (r) =>
          r.json().total_size == 1,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances.length`]: (r) =>
          r.json().instances.length === 1,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].name`]: (r) =>
          r.json().instances[0].name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "latest",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].state`]: (r) =>
          r.json().instances[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].task`]: (r) =>
          r.json().instances[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].model_definition`]: (r) =>
          r.json().instances[0].model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].create_time`]: (r) =>
          r.json().instances[0].create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].update_time`]: (r) =>
          r.json().instances[0].update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].configuration`]: (r) =>
          r.json().instances[0].configuration === null,
      });

      check(http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response total_size`]: (r) =>
          r.json().total_size == 1,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances.length`]: (r) =>
          r.json().instances.length === 1,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].name`]: (r) =>
          r.json().instances[0].name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "latest",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].state`]: (r) =>
          r.json().instances[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].task`]: (r) =>
          r.json().instances[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].model_definition`]: (r) =>
          r.json().instances[0].model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].create_time`]: (r) =>
          r.json().instances[0].create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].update_time`]: (r) =>
          r.json().instances[0].update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].configuration.content`]: (r) =>
          r.json().instances[0].configuration.content === "dummy-cls-model.zip",
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

  // Model Backend API: Get model instance list - github model
  {
    group("Model Backend API: Get model instance list github model", function () {
      let model_id = randomString(10)
      let createClsModelRes = http.request("POST", `${constant.apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-dummy-cls"
        }
      }), {
        headers: genHeader("application/json"),
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
        let res = http.get(`${constant.apiHost}/v1alpha/${createClsModelRes.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }
      let resp = http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances?page_size=1`, {
        headers: genHeader(`application/json`),
      })
      check(resp, {
        [`GET /v1alpha/models/${model_id}/instances task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response total_size`]: (r) =>
          r.json().total_size == 2,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances.length`]: (r) =>
          r.json().instances.length == 1,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].name`]: (r) =>
          r.json().instances[0].name === `models/${model_id}/instances/v1.1-cpu`,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "v1.1-cpu",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].state`]: (r) =>
          r.json().instances[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].task`]: (r) =>
          r.json().instances[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].model_definition`]: (r) =>
          r.json().instances[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].create_time`]: (r) =>
          r.json().instances[0].create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].update_time`]: (r) =>
          r.json().instances[0].update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].configuration`]: (r) =>
          r.json().instances[0].configuration === null,
      });

      check(http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances?page_size=1&page_token=${resp.json().next_page_token}`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances task cls response next_page_token`]: (r) =>
          r.json().next_page_token === "",
        [`GET /v1alpha/models/${model_id}/instances task cls response total_size`]: (r) =>
          r.json().total_size == 2,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances.length`]: (r) =>
          r.json().instances.length == 1,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].name`]: (r) =>
          r.json().instances[0].name === `models/${model_id}/instances/v1.0-cpu`,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "v1.0-cpu",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].state`]: (r) =>
          r.json().instances[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].task`]: (r) =>
          r.json().instances[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].model_definition`]: (r) =>
          r.json().instances[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].create_time`]: (r) =>
          r.json().instances[0].create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].update_time`]: (r) =>
          r.json().instances[0].update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].configuration`]: (r) =>
          r.json().instances[0].configuration === null,
      });

      check(http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response total_size`]: (r) =>
          r.json().total_size == 2,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances.length`]: (r) =>
          r.json().instances.length === 2,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].name`]: (r) =>
          r.json().instances[0].name === `models/${model_id}/instances/v1.1-cpu`,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "v1.1-cpu",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].state`]: (r) =>
          r.json().instances[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].task`]: (r) =>
          r.json().instances[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].model_definition`]: (r) =>
          r.json().instances[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].create_time`]: (r) =>
          r.json().instances[0].create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].update_time`]: (r) =>
          r.json().instances[0].update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].configuration.repository`]: (r) =>
          r.json().instances[0].configuration.repository === "instill-ai/model-dummy-cls",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].configuration.html_url`]: (r) =>
          r.json().instances[0].configuration.html_url === "https://github.com/instill-ai/model-dummy-cls/tree/v1.1-cpu",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].configuration.tag`]: (r) =>
          r.json().instances[0].configuration.tag === "v1.1-cpu",
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

export function LookupModelInstance() {
  // Model Backend API: Look up model instance
  {
    group("Model Backend API: Look up model instance", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
      let resCreateModel = http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(resCreateModel, {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiHost}/v1alpha/${resCreateModel.json().operation.name}`, {
          headers: genHeader(`application/json`),
        })
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      let resModel = http.get(`${constant.apiHost}/v1alpha/models/${model_id}`, {
        headers: genHeader(`application/json`),
      })
      check(resModel, {
        "GET /v1alpha/models/${model_id} response status": (r) =>
          r.status === 200
      });

      let resModelInstance = http.get(`${constant.apiHost}/v1alpha/models/${model_id}/instances/latest`, {
        headers: genHeader(`application/json`),
      })
      check(resModelInstance, {
        "GET /v1alpha/models/${model_id}/instances/latest response status": (r) =>
          r.status === 200
      });

      check(http.get(`${constant.apiHost}/v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration === null,
      });

      check(http.get(`${constant.apiHost}/v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}/lookUp task cls response instance.configuration.content`]: (r) =>
          r.json().instance.configuration.content === "dummy-cls-model.zip",
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