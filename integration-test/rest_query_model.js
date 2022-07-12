import http from "k6/http";
import { check, group } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
} from "./helpers.js";

const apiHost = "http://model-backend:8083";
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const model_def_name = "model-definitions/local"

export function GetModel() {
  // Model Backend API: Get model info
  {
    group("Model Backend API: Get model info", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
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
        "POST /v1alpha/models:multipart task cls response model.configuration.content": (r) =>
          r.json().model.configuration.content === "dummy-cls-model.zip",
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${model_id}`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id} task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id} task cls response model.name`]: (r) =>
          r.json().model.name === `models/${model_id}`,
        [`GET /v1alpha/models/${model_id} task cls response model.uid`]: (r) =>
          r.json().model.uid !== undefined,
        [`GET /v1alpha/models/${model_id} task cls response model.id`]: (r) =>
          r.json().model.id === model_id,
        [`GET /v1alpha/models/${model_id} task cls response model.description`]: (r) =>
          r.json().model.description === model_description,
        [`GET /v1alpha/models/${model_id} task cls response model.model_definition`]: (r) =>
          r.json().model.model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id} task cls response model.configuration`]: (r) =>
          r.json().model.configuration ===  null,
        [`GET /v1alpha/models/${model_id} task cls response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/models/${model_id} task cls response model.owner`]: (r) =>
          r.json().model.user === 'users/local-user',
        [`GET /v1alpha/models/${model_id} task cls response model.create_time`]: (r) =>
          r.json().model.create_time !== undefined,
        [`GET /v1alpha/models/${model_id} task cls response model.update_time`]: (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${model_id}?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id} task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id} task cls response model.name`]: (r) =>
          r.json().model.name === `models/${model_id}`,
        [`GET /v1alpha/models/${model_id} task cls response model.uid`]: (r) =>
          r.json().model.uid !== undefined,
        [`GET /v1alpha/models/${model_id} task cls response model.id`]: (r) =>
          r.json().model.id === model_id,
        [`GET /v1alpha/models/${model_id} task cls response model.description`]: (r) =>
          r.json().model.description === model_description,
        [`GET /v1alpha/models/${model_id} task cls response model.model_definition`]: (r) =>
          r.json().model.model_definition === model_def_name,
        [`GET /v1alpha/models/${model_id} task cls response model.configuration.content`]: (r) =>
          r.json().model.configuration.content === "dummy-cls-model.zip",
        [`GET /v1alpha/models/${model_id} task cls response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/models/${model_id} task cls response model.owner`]: (r) =>
          r.json().model.user === 'users/local-user',
        [`GET /v1alpha/models/${model_id} task cls response model.create_time`]: (r) =>
          r.json().model.create_time !== undefined,
        [`GET /v1alpha/models/${model_id} task cls response model.update_time`]: (r) =>
          r.json().model.update_time !== undefined,
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

export function ListModel() {
  // Model Backend API: Get model list
  {
    group("Model Backend API: Get model list", function () {
      let model_id = randomString(10)
      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-dummy-cls"
        }
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
          r.json().model.model_definition === "model-definitions/github",
        "POST /v1alpha/models:multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task cls response model.configuration.repository": (r) =>
          r.json().model.configuration.repository === "instill-ai/model-dummy-cls",
        "POST /v1alpha/models:multipart task cls response model.configuration.html_url": (r) =>
          r.json().model.configuration.html_url === "https://github.com/instill-ai/model-dummy-cls",
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PUBLIC",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models task cls response total_size`]: (r) =>
          r.json().total_size == 1,
        [`GET /v1alpha/models task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models task cls response models.length`]: (r) =>
          r.json().models.length === 1,
        [`GET /v1alpha/models task cls response models[0].name`]: (r) =>
          r.json().models[0].name === `models/${model_id}`,
        [`GET /v1alpha/models task cls response models[0].uid`]: (r) =>
          r.json().models[0].uid !== undefined,
        [`GET /v1alpha/models task cls response models[0].id`]: (r) =>
          r.json().models[0].id === model_id,
        [`GET /v1alpha/models task cls response models[0].description`]: (r) =>
          r.json().models[0].description !== undefined,
        [`GET /v1alpha/models task cls response models[0].model_definition`]: (r) =>
          r.json().models[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/models task cls response models[0].configuration`]: (r) =>
          r.json().models[0].configuration ===  null,
        [`GET /v1alpha/models task cls response models[0].visibility`]: (r) =>
          r.json().models[0].visibility === "VISIBILITY_PUBLIC",
        [`GET /v1alpha/models task cls response models[0].owner`]: (r) =>
          r.json().models[0].user === 'users/local-user',
        [`GET /v1alpha/models task cls response models[0].create_time`]: (r) =>
          r.json().models[0].create_time !== undefined,
        [`GET /v1alpha/models task cls response models[0].update_time`]: (r) =>
          r.json().models[0].update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models?view=VIEW_FULL task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response total_size`]: (r) =>
          r.json().total_size == 1,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models.length`]: (r) =>
          r.json().models.length === 1,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].name`]: (r) =>
          r.json().models[0].name === `models/${model_id}`,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].uid`]: (r) =>
          r.json().models[0].uid !== undefined,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].id`]: (r) =>
          r.json().models[0].id === model_id,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].description`]: (r) =>
          r.json().models[0].description !== undefined,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].model_definition`]: (r) =>
          r.json().models[0].model_definition === "model-definitions/github",
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].configuration.repository`]: (r) =>
          r.json().models[0].configuration.repository  === "instill-ai/model-dummy-cls",
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].configuration.html_url`]: (r) =>
          r.json().models[0].configuration.html_url  === "https://github.com/instill-ai/model-dummy-cls",
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].visibility`]: (r) =>
          r.json().models[0].visibility === "VISIBILITY_PUBLIC",
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].owner`]: (r) =>
          r.json().models[0].user === 'users/local-user',
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].create_time`]: (r) =>
          r.json().models[0].create_time !== undefined,
        [`GET /v1alpha/models?view=VIEW_FULL task cls response models[0].update_time`]: (r) =>
          r.json().models[0].update_time !== undefined,
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

export function LookupModel() {
  // Model Backend API: look up model
  {
    group("Model Backend API: Look up model", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      let res = http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(res, {
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
        "POST /v1alpha/models:multipart task cls response model.configuration.content": (r) =>
          r.json().model.configuration.content === "dummy-cls-model.zip",
        "POST /v1alpha/models:multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${res.json().model.uid}:lookUp`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.name`]: (r) =>
          r.json().model.name === `models/${model_id}`,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.uid`]: (r) =>
          r.json().model.uid !== undefined,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.id`]: (r) =>
          r.json().model.id === model_id,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.description`]: (r) =>
          r.json().model.description === model_description,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.model_definition`]: (r) =>
          r.json().model.model_definition === model_def_name,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.configuration`]: (r) =>
          r.json().model.configuration ===  null,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.owner`]: (r) =>
          r.json().model.user === 'users/local-user',
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.create_time`]: (r) =>
          r.json().model.create_time !== undefined,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.update_time`]: (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${res.json().model.uid}:lookUp?view=VIEW_FULL`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.name`]: (r) =>
          r.json().model.name === `models/${model_id}`,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.uid`]: (r) =>
          r.json().model.uid !== undefined,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.id`]: (r) =>
          r.json().model.id === model_id,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.description`]: (r) =>
          r.json().model.description === model_description,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.model_definition`]: (r) =>
          r.json().model.model_definition === model_def_name,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.configuration.content`]: (r) =>
          r.json().model.configuration.content === "dummy-cls-model.zip",
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.owner`]: (r) =>
          r.json().model.user === 'users/local-user',
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.create_time`]: (r) =>
          r.json().model.create_time !== undefined,
        [`GET /v1alpha/models/${res.json().model.uid}:lookUp task cls response model.update_time`]: (r) =>
          r.json().model.update_time !== undefined,
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
