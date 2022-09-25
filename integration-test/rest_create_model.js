import http from "k6/http";
import { check, group } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
} from "./helpers.js";

const apiHost = __ENV.HOSTNAME ? `http://${__ENV.HOSTNAME}:8083` : "http://model-backend:8083";

const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
const keypoint_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-keypoint-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const cls_model_bz17 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model-bz17.zip`, "b");
const det_model_bz9 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model-bz9.zip`, "b");
const keypoint_model_bz9 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-keypoint-model-bz9.zip`, "b");
const unspecified_model_bz3 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model-bz3.zip`, "b");


export function CreateModelFromLocal() {
  // Model Backend API: upload model
  {
    group("Model Backend API: Upload a model", function () {
      let fd_cls = new FormData();
      let model_id_cls = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("id", model_id_cls);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition", "model-definitions/local");
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id_cls}`,
        "POST /v1alpha/models/multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models/multipart task cls response model.id": (r) =>
          r.json().model.id === model_id_cls,
        "POST /v1alpha/models/multipart task cls response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models/multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === "model-definitions/local",
        "POST /v1alpha/models/multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models/multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models/multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models/multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let fd_det = new FormData();
      let model_id_det = randomString(10)
      model_description = randomString(20)
      fd_det.append("id", model_id_det);
      fd_det.append("description", model_description);
      fd_det.append("model_definition", "model-definitions/local");
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task det response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task det response model.name": (r) =>
          r.json().model.name === `models/${model_id_det}`,
        "POST /v1alpha/models/multipart task det response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models/multipart task det response model.id": (r) =>
          r.json().model.id === model_id_det,
        "POST /v1alpha/models/multipart task det response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models/multipart task det response model.model_definition": (r) =>
          r.json().model.model_definition === "model-definitions/local",
        "POST /v1alpha/models/multipart task det response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models/multipart task det response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models/multipart task det response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/multipart task det response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models/multipart task det response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let fd_keypoint = new FormData();
      let model_id_keypoint = randomString(10)
      model_description = randomString(20)
      fd_keypoint.append("id", model_id_keypoint);
      fd_keypoint.append("description", model_description);
      fd_keypoint.append("model_definition", "model-definitions/local");
      fd_keypoint.append("content", http.file(keypoint_model, "dummy-keypoint-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_keypoint.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_keypoint.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task keypoint response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task keypoint response model.name": (r) =>
          r.json().model.name === `models/${model_id_keypoint}`,
        "POST /v1alpha/models/multipart task keypoint response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models/multipart task keypoint response model.id": (r) =>
          r.json().model.id === model_id_keypoint,
        "POST /v1alpha/models/multipart task keypoint response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models/multipart task keypoint response model.model_definition": (r) =>
          r.json().model.model_definition === "model-definitions/local",
        "POST /v1alpha/models/multipart task keypoint response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models/multipart task keypoint response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models/multipart task keypoint response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/multipart task keypoint response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models/multipart task keypoint response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let fd_unspecified = new FormData();
      let model_id_unspecified = randomString(10)
      model_description = randomString(20)
      fd_unspecified.append("id", model_id_unspecified);
      fd_unspecified.append("description", model_description);
      fd_unspecified.append("model_definition", "model-definitions/local");
      fd_unspecified.append("content", http.file(unspecified_model, "dummy-unspecified-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task unspecified response model.name": (r) =>
          r.json().model.name === `models/${model_id_unspecified}`,
        "POST /v1alpha/models/multipart task unspecified response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models/multipart task unspecified response model.id": (r) =>
          r.json().model.id === model_id_unspecified,
        "POST /v1alpha/models/multipart task unspecified response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models/multipart task unspecified response model.model_definition": (r) =>
          r.json().model.model_definition === "model-definitions/local",
        "POST /v1alpha/models/multipart task unspecified response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models/multipart task unspecified response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models/multipart task unspecified response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/multipart task unspecified response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models/multipart task unspecified response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /v1alpha/models/multipart already existed response status 409": (r) =>
          r.status === 409,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_cls}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_det}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_keypoint}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_unspecified}`, null, {
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
      fd_cls.append("content", http.file(cls_model_bz17, "dummy-cls-model-bz17.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
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
      fd_det.append("content", http.file(det_model_bz9, "dummy-det-model-bz9.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_det.body(), {
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
      fd_keypoint.append("content", http.file(keypoint_model_bz9, "dummy-keypoint-model-bz9.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_keypoint.body(), {
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
      fd_unspecified.append("content", http.file(unspecified_model_bz3, "dummy-unspecified-model-bz3.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/multipart`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 400,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_cls}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_det}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_keypoint}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 404
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_unspecified}`, null, {
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
      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-dummy-cls"
        },
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models/multipart task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models/multipart task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models/multipart task cls response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models/multipart task cls response model.description": (r) =>
          r.json().model.description !== undefined,
        "POST /v1alpha/models/multipart task cls response model.model_definition": (r) =>
          r.json().model.model_definition === "model-definitions/github",
        "POST /v1alpha/models/multipart task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models/multipart task cls response model.configuration.repository": (r) =>
          r.json().model.configuration.repository === "instill-ai/model-dummy-cls",
        "POST /v1alpha/models/multipart task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PUBLIC",
        "POST /v1alpha/models/multipart task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models/multipart task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models/multipart task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0/deploy`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/v1.0`,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "v1.0",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === "model-definitions/github",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/deploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [
          { "image_url": "https://artifacts.instill.tech/dog.jpg" }
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0/trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "inputs": [
          { "image_url": "https://artifacts.instill.tech/dog.jpg" },
          { "image_url": "https://artifacts.instill.tech/dog.jpg" }
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0/trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/v1.0/trigger url cls task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score === 1,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
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

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "Phelan164/non-exited"
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github invalid url status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
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

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition": "model-definitions/github",
        "configuration": {
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github missing github_url status": (r) =>
          r.status === 400,
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
