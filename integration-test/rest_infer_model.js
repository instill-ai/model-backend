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
const keypoint_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-keypoint-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const model_def_name = "model-definitions/local"
const model_def_uid = "909c3278-f7d1-461c-9352-87741bef11d3"


export function InferModel() {
  // Model Backend API: make inference
  {
    group("Model Backend API: Predict Model with classification model", function () {
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
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {}, {
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
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }]
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
          { "image_url": "https://artifacts.instill.tech/dog.jpg" },
          { "image_url": "https://artifacts.instill.tech/dog.jpg" }
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
        "inputs": [{ "image_base64": base64_image, }]
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
          { "image_base64": base64_image, },
          { "image_base64": base64_image, }
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
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls output.classification_outputs.length`]: (r) =>
          r.json().output.classification_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls output.classification_outputs[0].category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls response output.classification_outputs[0].score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls output.classification_outputs[1].category`]: (r) =>
          r.json().output.classification_outputs[1].category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data cls response output.classification_outputs[1].score`]: (r) =>
          r.json().output.classification_outputs[1].score === 1,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
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
      fd_det.append("id", model_id);
      fd_det.append("description", model_description);
      fd_det.append("model_definition", model_def_name);
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
        "POST /v1alpha/models:multipart task det response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task det response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task det response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task det response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task det response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart task det response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task det response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task det response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task det response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task det response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task det response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {}, {
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
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }],
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
          { "image_url": "https://artifacts.instill.tech/dog.jpg" },
          { "image_url": "https://artifacts.instill.tech/dog.jpg" }
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
        "inputs": [{ "image_base64": base64_image, }]
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
          { "image_base64": base64_image, },
          { "image_base64": base64_image, }
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
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response output.detection_outputs[1].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response output.detection_outputs[1].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].score !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple-part det multiple images response output.detection_outputs[1].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[1].bounding_box_objects[0].bounding_box !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
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
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(unspecified_model, "dummy-unspecified-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        "POST /v1alpha/models:multipart task unspecified response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task unspecified response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task unspecified response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task unspecified response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task unspecified response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart task unspecified response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task unspecified response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task unspecified response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task unspecified response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task unspecified response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task unspecified response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {}, {
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
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }]
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
          { "image_url": "https://artifacts.instill.tech/dog.jpg" },
          { "image_url": "https://artifacts.instill.tech/dog.jpg" }
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
        "inputs": [{ "image_base64": base64_image, }]
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
          { "image_base64": base64_image, },
          { "image_base64": base64_image, }
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
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
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
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
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
          r.status === 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }

  // Model Backend API: make inference
  {
    group("Model Backend API: Predict Model with keypoint model", function () {
      let fd_keypoint = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_keypoint.append("id", model_id);
      fd_keypoint.append("description", model_description);
      fd_keypoint.append("model_definition", model_def_name);
      fd_keypoint.append("content", http.file(keypoint_model, "dummy-keypoint-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_keypoint.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_keypoint.boundary}`),
      }), {
        "POST /v1alpha/models:multipart task keypoint response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart task keypoint response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task keypoint response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task keypoint response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task keypoint response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart task keypoint response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart task keypoint response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task keypoint response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart task keypoint response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task keypoint response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task keypoint response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      sleep(5) // Triton loading models takes time

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.task`]: (r) =>
          r.json().instance.task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online task keypoint response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint output.keypoint_outputs.length`]: (r) =>
          r.json().output.keypoint_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint output.keypoint_outputs[0].keypoints.length`]: (r) =>
          r.json().output.keypoint_outputs[0].keypoints.length === 17,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint response output.keypoint_outputs[0].score`]: (r) =>
          r.json().output.keypoint_outputs[0].score === 1,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "inputs": [
          { "image_url": "https://artifacts.instill.tech/dog.jpg" },
          { "image_url": "https://artifacts.instill.tech/dog.jpg" }
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint multiple images status`]: (r) =>
          r.status === 400,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{ "image_base64": base64_image, }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint output.keypoint_outputs.length`]: (r) =>
          r.json().output.keypoint_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint output.keypoint_outputs[0].keypoints.length`]: (r) =>
          r.json().output.keypoint_outputs[0].keypoints.length === 17,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint response output.keypoint_outputs[0].score`]: (r) =>
          r.json().output.keypoint_outputs[0].score === 1,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "inputs": [
          { "image_base64": base64_image, },
          { "image_base64": base64_image, }
        ]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint multiple images status`]: (r) =>
          r.status === 400,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger multiple-part keypoint response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger multiple-part keypoint output.keypoint_outputs.length`]: (r) =>
          r.json().output.keypoint_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger multiple-part keypoint output.keypoint_outputs[0].keypoints.length`]: (r) =>
          r.json().output.keypoint_outputs[0].keypoints.length === 17,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger multiple-part keypoint response output.keypoint_outputs[0].score`]: (r) =>
          r.json().output.keypoint_outputs[0].score === 1,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart form-data keypoint response status`]: (r) =>
          r.status === 400,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/v1alpha/models/${model_id}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }    
}
