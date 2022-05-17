import http from "k6/http";
import { sleep, check, group, fail } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import {
  genHeader,
  base64_image,
} from "./helpers.js";

const apiHost = "http://localhost:8083";

const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");

const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const model_def_name = "model-definitions/github"
const model_def_uid = "909c3278-f7d1-461c-9352-87741bef11d3"

export function GetModelInstance() {
  // Model Backend API: Get model instance
  {
    group("Model Backend API: Get model instance", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/" + model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /v1alpha/models:multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task cls response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task cls response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${model_id}/instances/latest`, {
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
          r.json().instance.configuration !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
}

export function ListModelInstance() {
  // Model Backend API: Get model instance list - local model
  {
    group("Model Backend API: Get model instance list local model", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/" + model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /v1alpha/models:multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task cls response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task cls response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      check(http.get(`${apiHost}/v1alpha/models/${model_id}/instances`, {
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
          r.json().instances[0].configuration === "",
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
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
      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": model_def_name,
        "configuration": JSON.stringify({
          "repository": "Phelan164/test-repo",
          "html_url": ""
        })
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models:multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task cls response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task cls response model.description": (r) =>
          r.json().model.description !== undefined,
        "POST /v1alpha/models:multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task cls response model.configuration.repository": (r) =>
          JSON.parse(r.json().model.configuration).repository === "Phelan164/test-repo",
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PUBLIC",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${model_id}/instances`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id}/instances task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response total_size`]: (r) =>
          r.json().total_size == 2,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances.length`]: (r) =>
          r.json().instances.length == 2,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].name`]: (r) =>
          r.json().instances[0].name === `models/${model_id}/instances/v1.1`,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "v1.1",
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
          r.json().instances[0].configuration === "",
      });

      check(http.get(`${apiHost}/v1alpha/models/${model_id}/instances?view=VIEW_FULL`, {
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
          r.json().instances[0].name === `models/${model_id}/instances/v1.1`,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].uid`]: (r) =>
          r.json().instances[0].uid !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].id`]: (r) =>
          r.json().instances[0].id === "v1.1",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].state`]: (r) =>
          r.json().instances[0].state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].task`]: (r) =>
          r.json().instances[0].task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].model_definition`]: (r) =>
          r.json().instances[0].model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].create_time`]: (r) =>
          r.json().instances[0].create_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].update_time`]: (r) =>
          r.json().instances[0].update_time !== undefined,
        [`GET /v1alpha/models/${model_id}/instances?view=VIEW_FULL task cls response instances[0].configuration`]: (r) =>
          r.json().instances[0].configuration !== null,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
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
      fd_cls.append("name", "models/" + model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      let resModel = http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(resModel, {
        "POST /v1alpha/models:multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task cls response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task cls response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let resModelInstance = http.get(`${apiHost}/v1alpha/models/${model_id}/instances/latest`, {
        headers: genHeader(`application/json`),
      })
      check(resModelInstance, {
        "GET /v1alpha/models/${model_id}/instances/latest response status": (r) =>
          r.status === 200
      });

      check(http.get(`${apiHost}/v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`GET /v1alpha/models/${resModel.json().model.uid}/instances/${resModelInstance.json().instance.uid}:lookUp task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
}
