import http from "k6/http";
import {sleep, check, group, fail} from "k6";
import {FormData} from "https://jslib.k6.io/formdata/0.0.2/index.js";
import {randomString} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import {URL} from "https://jslib.k6.io/url/1.0.0/index.js";

import {
  genHeader,
  base64_image,
} from "./helpers.js";

const apiHost = "http://localhost:8080";

const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");

const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const model_def_name = "model-definitions/github"
const model_def_uid = "909c3278-f7d1-461c-9352-87741bef11d3"

export let options = {
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

export function setup() {
}

export default function (data) {
  let resp;

  /*
   * Model API - API CALLS
   */

  // Health check
  {
    group("Model API: __liveness check", () => {
      check(http.request("GET", `${apiHost}/v1alpha/__liveness`), {
        "GET /__liveness response status is 200": (r) => r.status === 200,
      });
      check(http.request("GET", `${apiHost}/v1alpha/__readiness`), {
        "GET /__readiness response status is 200": (r) => r.status === 200,
      });
    });
  }
  // Model Backend API: get model definition
  {
    group("Model Backend API: get model definition", function () {
      check(http.get(`${apiHost}/v1alpha/${model_def_name}`, {
        headers: genHeader(`application/json`),
      }), {
          [`GET /v1alpha/model-definitions/${model_def_name} response status`]: (r) =>
          r.status === 200, 
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.name`]: (r) =>
          r.json().model_definition.name === model_def_name,
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.id`]: (r) =>
          r.json().model_definition.id === "github",          
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.uid`]: (r) =>
          r.json().model_definition.uid === model_def_uid,
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.title`]: (r) =>
          r.json().model_definition.title === "GitHub",
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.documentation_url`]: (r) =>
          r.json().model_definition.documentation_url !== undefined,
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.model_spec`]: (r) =>
          r.json().model_definition.model_spec !== undefined,
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.model_instance_spec`]: (r) =>
          r.json().model_definition.model_instance_spec !== undefined,
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.create_time`]: (r) =>
          r.json().model_definition.create_time !== undefined,
          [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.update_time`]: (r) =>
          r.json().model_definition.update_time !== undefined,
      });
    });
  }

  // Model Backend API: get model definition list
  {
    group("Model Backend API: get model definition list", function () {
      check(http.get(`${apiHost}/v1alpha/model-definitions`, {
        headers: genHeader(`application/json`),
      }), {
          [`GET /v1alpha/model-definitions} response status`]: (r) =>
          r.status === 200, 
          [`GET /v1alpha/model-definitions response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
          [`GET /v1alpha/model-definitions response total_size`]: (r) =>
          r.json().total_size == 2,
          [`GET /v1alpha/model-definitions response model_definitions.length`]: (r) =>
          r.json().model_definitions.length === 2
      });
    });
  }

  // Model Backend API: upload model
  {
    group("Model Backend API: Upload a model", function () {
      let fd_cls = new FormData();
      let model_id_cls = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id_cls);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id_cls}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id_cls,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let fd_det = new FormData();
      let model_id_det = randomString(10)
      model_description = randomString(20)
      fd_det.append("name", "models/"+model_id_det);
      fd_det.append("description", model_description);
      fd_det.append("model_definition_name", model_def_name);
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
          "POST /v1alpha/models github task det response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name === `models/${model_id_det}`,
          "POST /v1alpha/models/upload (multipart) task det response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task det response model.id": (r) =>
          r.json().model.id === model_id_det,          
          "POST /v1alpha/models/upload (multipart) task det response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task det response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task det response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task det response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task det response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task det response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task det response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let fd_unspecified = new FormData();
      let model_id_unspecified = randomString(10)
      model_description = randomString(20)
      fd_unspecified.append("name", "models/"+model_id_unspecified);
      fd_unspecified.append("description", model_description);
      fd_unspecified.append("model_definition_name", model_def_name);
      fd_unspecified.append("content", http.file(unspecified_model, "dummy-unspecified-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
          "POST /v1alpha/models github task unspecified response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task unspecified response model.name": (r) =>
          r.json().model.name === `models/${model_id_unspecified}`,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.id": (r) =>
          r.json().model.id === model_id_unspecified,          
          "POST /v1alpha/models/upload (multipart) task unspecified response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task unspecified response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task unspecified response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /v1alpha/models/upload (multipart) already existed response status 409": (r) =>
        r.status === 409, 
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_cls}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_det}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id_unspecified}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
    });
  }

  // Model Backend API: load model online
  {
    group("Model Backend API: Load model online", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {} , {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:undeploy`, {} , {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_OFFLINE",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
    });
  }

  // Model Backend API: make inference
  {
    group("Model Backend API: Predict Model with classification model", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {} , {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "latest",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls response output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "inputs": [
          {"image_url": "https://artifacts.instill.tech/dog.jpg"},
          {"image_url": "https://artifacts.instill.tech/dog.jpg"}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images output.classification_outputs[1].category`]: (r) =>
          r.json().output.classification_outputs[1].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images output.classification_outputs[1].score`]: (r) =>
          r.json().output.classification_outputs[1].score === 1,          
      });      

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls response output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "inputs": [
          {"image_base64": base64_image,},
          {"image_base64": base64_image,}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images response output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images output.classification_outputs[1].category`]: (r) =>
          r.json().output.classification_outputs[1].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images response output.classification_outputs[1].score`]: (r) =>
          r.json().output.classification_outputs[1].score === 1,          
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest/upload:trigger`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest/upload:trigger`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls response output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls output.classification_outputs[1].category`]: (r) =>
          r.json().output.classification_outputs[1].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger form-data cls response output.classification_outputs[1].score`]: (r) =>
          r.json().output.classification_outputs[1].score === 1,          
      });      

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }

  // Model Backend API: make inference
  {
    group("Model Backend API: Predict Model with detection model", function () {
      let fd_det = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_det.append("name", "models/"+model_id);
      fd_det.append("description", model_description);
      fd_det.append("model_definition_name", model_def_name);
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
          "POST /v1alpha/models github task det response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task det response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task det response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task det response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task det response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task det response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task det response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task det response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task det response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task det response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {} , {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.id`]: (r) =>
          r.json().instance.id === "latest",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.task`]: (r) =>
          r.json().instance.task === "TASK_DETECTION",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task det response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}],
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "inputs": [
          {"image_url": "https://artifacts.instill.tech/dog.jpg"},
          {"image_url": "https://artifacts.instill.tech/dog.jpg"}
        ],
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images output.detection_outputs[1].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images output.detection_outputs[1].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det response output.detection_outputs[1].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].bounding_box !== undefined,          
      });      

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "inputs": [
          {"image_base64": base64_image,},
          {"image_base64": base64_image,}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images utput.detection_outputs[0].bounding_box_objects.length`]: (r) =>
          r.json().output.detection_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images output.detection_outputs[1].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images output.detection_outputs[1].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images output.detection_outputs[1].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].bounding_box !== undefined,          
      });      

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest/upload:trigger`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest/upload:trigger`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response output.detection_outputs[1].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].category === "test",
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response output.detection_outputs[1].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].score !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest/upload:trigger multiple-part det multiple images response output.detection_outputs[1].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].bounding_box !== undefined,          
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }

  // Model Backend API: make inference
  {
    group("Model Backend API: Predict Model with undefined task model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("name", "models/"+model_id);
      fd.append("description", model_description);
      fd.append("model_definition_name", model_def_name);
      fd.append("content", http.file(unspecified_model, "dummy-unspecified-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
          "POST /v1alpha/models github task unspecified response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task unspecified response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task unspecified response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task unspecified response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task unspecified response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task unspecified response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {} , {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.id`]: (r) =>
          r.json().instance.id === "latest",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.task`]: (r) =>
          r.json().instance.task === "TASK_UNSPECIFIED",
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task unspecified response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });
      sleep(5) // Triton loading models takes time        

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined outputs`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined raw_output_contents`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined raw_output_contents content`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "inputs": [
          {"image_url": "https://artifacts.instill.tech/dog.jpg"},
          {"image_url": "https://artifacts.instill.tech/dog.jpg"}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined multiple images outputs`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined multiple images output.outputs[0].shape[0]`]: (r) =>
          r.json().output.outputs[0].shape[0] === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined multiple images parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined multiple images raw_output_contents`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined multiple images output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });      

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "inputs": [
          {"image_base64": base64_image,},
          {"image_base64": base64_image,}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined multiple images output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined multiple images output.outputs[0].shape[0]`]: (r) =>
          r.json().output.outputs[0].shape[0] === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined multiple images output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined multiple images output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined multiple images output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,      
      });      

      // Predict with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest/upload:trigger`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest/upload:trigger`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined multiple images output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined multiple images output.outputs[0].shape[0]`]: (r) =>
          r.json().output.outputs[0].shape[0] === 2,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined multiple images output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined multiple images output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger multipart undefined multiple images output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });      

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }

  // Model Backend API: Get model info
  {
    group("Model Backend API: Get model info", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models/${model_id}`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models/${model_id} github task cls response status`]: (r) =>
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
        r.json().model.configuration !== undefined,
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
          r.status === 200 // TODO: update status to 204
      });
    });
  }

 // Model Backend API: Get model list
  {
    group("Model Backend API: Get model list", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.get(`${apiHost}/v1alpha/models`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/models github task cls response status`]: (r) =>
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
        r.json().models[0].description === model_description,
        [`GET /v1alpha/models task cls response models[0].model_definition`]: (r) =>
        r.json().models[0].model_definition === model_def_name,
        [`GET /v1alpha/models task cls response models[0].configuration`]: (r) =>
        r.json().models[0].configuration !== undefined,
        [`GET /v1alpha/models task cls response models[0].visibility`]: (r) =>
        r.json().models[0].visibility === "VISIBILITY_PRIVATE",
        [`GET /v1alpha/models task cls response models[0].owner`]: (r) =>
        r.json().models[0].user === 'users/local-user',
        [`GET /v1alpha/models task cls response models[0].create_time`]: (r) =>
        r.json().models[0].create_time !== undefined,
        [`GET /v1alpha/models task cls response models[0].update_time`]: (r) =>
        r.json().models[0].update_time !== undefined,           
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

    });
  }

  // Model Backend API: upload model by GitHub
  {
    group("Model Backend API: Upload a model by GitHub", function () {
      let model_id = randomString(10)
      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition":model_def_name,
        "configuration": {
          "repo": "https://github.com/Phelan164/test-repo.git",
          "tag": "v1.0",
          "html_url": ""
        }
      }), {
        headers: genHeader("application/json"),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models github task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models github task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models github task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models github task cls response model.description": (r) =>
          r.json().model.description !== undefined,
          "POST /v1alpha/models github task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models github task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models github task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models github task cls response model.configuration.repo": (r) =>
          r.json().model.configuration.repo === "https://github.com/Phelan164/test-repo.git",
          "POST /v1alpha/models github task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PUBLIC",
          "POST /v1alpha/models github task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models github task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models github task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0:deploy`, {} , {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/v1.0`,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.id`]: (r) =>
          r.json().instance.id === "v1.0",
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task cls response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });
      sleep(5) // Triton loading models takes time   

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [
          {"image_url": "https://artifacts.instill.tech/dog.jpg"}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "inputs": [
          {"image_url": "https://artifacts.instill.tech/dog.jpg"},
          {"image_url": "https://artifacts.instill.tech/dog.jpg"}
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls multiple images response status`]: (r) =>
          r.status === 200,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls multiple images output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 2,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls multiple images output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls multiple images output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls multiple images output.classification_outputs[1].category`]: (r) =>
          r.json().output.classification_outputs[1].category === "match",
          [`POST /v1alpha/models/${model_id}/instances/v1.0:trigger url cls multiple images output.classification_outputs[1].score`]: (r) =>
          r.json().output.classification_outputs[1].score === 1,          
      });      

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition":model_def_name,
        "configuration": {
            "repo": "https://github.com/Phelan164/test-repo.git",
            "tag": "invalid-tag",
            "html_url": ""
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github invalid url status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition":model_def_name,
        "configuration": {
          "repo": "https://github.com/Phelan164/non-exited.git",
          "tag": "v1.0",
          "html_url": ""
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github invalid url status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "model_definition":model_def_name,
        "configuration": {
          "repo": "https://github.com/Phelan164/test-repo.git",
          "tag": "v1.0",
          "html_url": ""
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models by github missing name status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": randomString(10),
        "model_definition":model_def_name,
        "configuration": {
          "tag": "v1.0",
          "html_url": ""
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
          r.status === 200 // TODO: update status to 204
      });

    });
  }

  // Model Backend API: Get model instance
  {
    group("Model Backend API: Get model instance", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
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
          r.status === 200 // TODO: update status to 204
      });
    });
  }

 // Model Backend API: Get model instance list
  {
    group("Model Backend API: Get model instance list", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
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
          r.json().instances[0].configuration !== undefined,              
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

    });
  }

  // Model Backend API: look up model
  {
    group("Model Backend API: Look up model", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      let res = http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(res, {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
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
        r.json().model.configuration !== undefined,
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
          r.status === 200 // TODO: update status to 204
      });
    });
  }

  // Model Backend API: Look up model instance
  {
    group("Model Backend API: Look up model instance", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      let resModel = http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      })
      check(resModel, {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
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
          r.status === 200 // TODO: update status to 204
      });
    });
  }

  // Model Backend API: Update model
  {
    group("Model Backend API: Update model", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      let new_description = randomString(20)
      let payload = JSON.stringify({
        "description": new_description
      })
      check(http.patch(`${apiHost}/v1alpha/models/${model_id}`, payload, {
        headers: genHeader(`application/json`)
      }), {
        [`PATCH /v1alpha/models/${model_id} task cls response status`]: (r) =>
        r.status === 200, 
        [`PATCH /v1alpha/models/${model_id} task cls response model.name`]: (r) =>
        r.json().model.name === `models/${model_id}`,
        [`PATCH /v1alpha/models/${model_id} task cls response model.uid`]: (r) =>
        r.json().model.uid !== undefined,
        [`PATCH /v1alpha/models/${model_id} task cls response model.id`]: (r) =>
        r.json().model.id === model_id,          
        [`PATCH /v1alpha/models/${model_id} task cls response model.description`]: (r) =>
        r.json().model.description === new_description,
        [`PATCH /v1alpha/models/${model_id} task cls response model.model_definition`]: (r) =>
        r.json().model.model_definition === model_def_name,
        [`PATCH /v1alpha/models/${model_id} task cls response model.configuration`]: (r) =>
        r.json().model.configuration !== undefined,
        [`PATCH /v1alpha/models/${model_id} task cls response model.visibility`]: (r) =>
        r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`PATCH /v1alpha/models/${model_id} task cls response model.owner`]: (r) =>
        r.json().model.user === 'users/local-user',
        [`PATCH /v1alpha/models/${model_id} task cls response model.create_time`]: (r) =>
        r.json().model.create_time !== undefined,
        [`PATCH /v1alpha/models/${model_id} task cls response model.update_time`]: (r) =>
        r.json().model.update_time !== undefined,           
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
    });
  }

  // Model Backend API: PublishModel
  {
    group("Model Backend API: PublishModel", function () {
      let fd_cls = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", "models/"+model_id);
      fd_cls.append("description", model_description);
      fd_cls.append("model_definition_name", model_def_name);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /v1alpha/models github task cls response status": (r) =>
          r.status === 201, 
          "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
          "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
          r.json().model.uid !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id === model_id,          
          "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
          "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
          "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
          "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}:publish`, null, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}:publish task cls response status`]: (r) =>
        r.status === 200, 
        [`POST /v1alpha/models/${model_id}:publish task cls response model.name`]: (r) =>
        r.json().model.name === `models/${model_id}`,
        [`POST /v1alpha/models/${model_id}:publish task cls response model.uid`]: (r) =>
        r.json().model.uid !== undefined,
        [`POST /v1alpha/models/${model_id}:publish task cls response model.id`]: (r) =>
        r.json().model.id === model_id,          
        [`POST /v1alpha/models/${model_id}:publish task cls response model.description`]: (r) =>
        r.json().model.description === model_description,
        [`POST /v1alpha/models/${model_id}:publish task cls response model.model_definition`]: (r) =>
        r.json().model.model_definition === model_def_name,
        [`POST /v1alpha/models/${model_id}:publish task cls response model.configuration`]: (r) =>
        r.json().model.configuration !== undefined,
        [`POST /v1alpha/models/${model_id}:publish task cls response model.visibility`]: (r) =>
        r.json().model.visibility === "VISIBILITY_PUBLIC",
        [`POST /v1alpha/models/${model_id}:publish task cls response model.owner`]: (r) =>
        r.json().model.user === 'users/local-user',
        [`POST /v1alpha/models/${model_id}:publish task cls response model.create_time`]: (r) =>
        r.json().model.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}:publish task cls response model.update_time`]: (r) =>
        r.json().model.update_time !== undefined,           
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}:unpublish`, null, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}:unpublish task cls response status`]: (r) =>
        r.status === 200, 
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.name`]: (r) =>
        r.json().model.name === `models/${model_id}`,
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.uid`]: (r) =>
        r.json().model.uid !== undefined,
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.id`]: (r) =>
        r.json().model.id === model_id,          
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.description`]: (r) =>
        r.json().model.description === model_description,
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.model_definition`]: (r) =>
        r.json().model.model_definition === model_def_name,
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.configuration`]: (r) =>
        r.json().model.configuration !== undefined,
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.visibility`]: (r) =>
        r.json().model.visibility === "VISIBILITY_PRIVATE",
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.owner`]: (r) =>
        r.json().model.user === 'users/local-user',
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.create_time`]: (r) =>
        r.json().model.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}:unpublish task cls response model.update_time`]: (r) =>
        r.json().model.update_time !== undefined,           
      });

      check(http.post(`${apiHost}/v1alpha/models/${randomString(10)}:publish`, null, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}:publish task cls response not found status`]: (r) => r.status === 404, 
      });

      check(http.post(`${apiHost}/v1alpha/models/${randomString(10)}:unpublish`, null, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}:unpublish task cls response not found status`]: (r) => r.status === 404, 
      });
      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
    });
  }   

}

export function teardown(data) {
  group("Model API: Delete all models created by this test", () => {
    for (const model of http
      .request("GET", `${apiHost}/v1alpha/models`, null, {
        headers: genHeader(
          "application/json"
        ),
      })
      .json("models")) {
      check(model, {
        "GET /clients response contents[*] id": (c) => c.id !== undefined,
      });
      check(
        http.request("DELETE", `${apiHost}/v1alpha/models/${model.id}`, null, {
          headers: genHeader("application/json"),
        }),
        {
          [`DELETE /v1alpha/models/${model.id} response status is 200`]: (r) =>
            r.status === 200,
        }
      );
    }
  });
}
