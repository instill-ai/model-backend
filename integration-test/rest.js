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
    group("Model API: Health check", () => {
      check(http.request("GET", `${apiHost}/health/model`), {
        "GET /health/model response status is 200": (r) => r.status === 200,
      });
    });
  }

  // Model Backend API: upload model
  {
    group("Model Backend API: Upload a model", function () {
      let fd_cls = new FormData();
      let model_name_cls = randomString(10)
      fd_cls.append("name", model_name_cls);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_CLASSIFICATION");
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) task cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      let fd_det = new FormData();
      let model_name_det = randomString(10)
      fd_det.append("name", model_name_det);
      fd_det.append("description", randomString(20));
      fd_det.append("task", "TASK_DETECTION");
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
        "POST /models/upload (multipart) task det response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task det response model.task": (r) =>
          r.json().model.task === "TASK_DETECTION",
          "POST /models/upload (multipart) task det response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      let fd_undefined = new FormData();
      let model_name_unspecified = randomString(10)
      fd_undefined.append("name", model_name_unspecified);
      fd_undefined.append("description", randomString(20));
      fd_undefined.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_undefined.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_undefined.boundary}`),
      }), {
        "POST /models/upload (multipart) task undefined response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task undefined response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task undefined response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task undefined response model.task": (r) =>
          r.json().model.task === "TASK_UNSPECIFIED",
          "POST /models/upload (multipart) task undefined response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name_cls}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
      check(http.request("DELETE", `${apiHost}/models/${model_name_det}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
      check(http.request("DELETE", `${apiHost}/models/${model_name_unspecified}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });
    });
  }

  // Model Backend API: auto increase model version
  {
    group("Model Backend API: Auto increase model version", function () {
      let fd_cls = new FormData();
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_CLASSIFICATION");
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      let fd_det = new FormData();
      fd_det.append("name", model_name);
      fd_det.append("description", randomString(20));
      fd_det.append("task", "TASK_CLASSIFICATION");
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
        "POST /models/upload (multipart) det response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task det response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task det response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 2,
      });

      let fd_undefined = new FormData();
      fd_undefined.append("name", model_name);
      fd_undefined.append("description", randomString(20));
      fd_det.append("task", "TASK_DETECTION");
      fd_undefined.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_undefined.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_undefined.boundary}`),
      }), {
        "POST /models/upload (multipart) wrong task response status": (r) =>
          r.status === 400,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
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
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_CLASSIFICATION");
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 2,
      });

      // Triton loading models takes time
      sleep(6)

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/1`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/1 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_ONLINE",
      });

      // Triton loading models takes time
      sleep(6)

      payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/2`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/2 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_ONLINE",
      });

      // Triton loading models takes time
      sleep(6)

      payload = JSON.stringify({
        "status": "STATUS_OFFLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/1`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/1 offline task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/1 offline task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/1 offline task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/1 offline task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/1 offline task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 offline task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 offline task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_OFFLINE",
      });

      // Triton unloading models takes time
      sleep(6)

      payload = JSON.stringify({
        "status": "STATUS_OFFLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/2`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/2 offline task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/2 offline task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/2 offline task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/2 offline task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/2 offline task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/2 offline task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/2 offline task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_OFFLINE",
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
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
    group("Model Backend API: Predict Model with classification model", function () {
      let fd_cls = new FormData();
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_CLASSIFICATION");
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE"
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/1`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/1 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_ONLINE",
      });

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/models/${model_name}/versions/1/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs url cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs url cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs url cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/versions/1/outputs url cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/models/${model_name}/versions/1/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs base64 cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs base64 cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs base64 cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/versions/1/outputs base64 cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict with multiple-part
      const fd = new FormData();
      fd.append("inputs", http.file(dog_img));
      check(http.post(`${apiHost}/models/${model_name}/versions/1/upload/outputs`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /models/${model_name}/versions/1/upload/outputs form-data cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/upload/outputs form-data cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/versions/1/upload/outputs form-data cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/versions/1/upload/outputs form-data cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
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
      let fd_cls = new FormData();
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_DETECTION");
      fd_cls.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) det response status": (r) =>
          r.status === 200,
          "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task det response model.task": (r) =>
          r.json().model.task === "TASK_DETECTION",
          "POST /models/upload (multipart) task det response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      // Triton loading models takes time
      sleep(6)

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/1`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/1 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_ONLINE",
      });

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}],
      });
      check(http.post(`${apiHost}/models/${model_name}/versions/1/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs url det response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs url det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs url det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /models/${model_name}/versions/1/outputs url det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /models/${model_name}/versions/1/outputs url det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/models/${model_name}/versions/1/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs base64 det response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs base64 det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs base64 det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /models/${model_name}/versions/1/outputs base64 det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /models/${model_name}/versions/1/outputs base64 det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict with multiple-part
      const fd = new FormData();
      fd.append("inputs", http.file(dog_img));
      check(http.post(`${apiHost}/models/${model_name}/versions/1/upload/outputs`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /models/${model_name}/versions/1/upload/outputs multiple-part det response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/upload/outputs multiple-part det output.detection_outputs[0].bounding_box_objects.length`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects.length === 1,
          [`POST /models/${model_name}/versions/1/upload/outputs multiple-part det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /models/${model_name}/versions/1/upload/outputs multiple-part det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /models/${model_name}/versions/1/upload/outputs multiple-part det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
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
      let model_name = randomString(10)
      fd.append("name", model_name);
      fd.append("description", randomString(20));
      fd.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        "POST /models/upload (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_UNSPECIFIED",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      // Triton loading models takes time
      sleep(6)

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/1`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/1 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status === "STATUS_ONLINE",
      });

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/models/${model_name}/versions/1/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs url undefined response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs url undefined outputs`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs url undefined parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /models/${model_name}/versions/1/outputs url undefined raw_output_contents`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /models/${model_name}/versions/1/outputs url undefined raw_output_contents content`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/models/${model_name}/versions/1/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs base64 undefined response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs base64 undefined output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs base64 undefined output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /models/${model_name}/versions/1/outputs base64 undefined output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /models/${model_name}/versions/1/outputs base64 undefined output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("inputs", http.file(dog_img));
      check(http.post(`${apiHost}/models/${model_name}/versions/1/upload/outputs`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /models/${model_name}/versions/1/outputs multipart undefined response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/versions/1/outputs multipart undefined output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /models/${model_name}/versions/1/outputs multipart undefined output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /models/${model_name}/versions/1/outputs multipart undefined output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /models/${model_name}/versions/1/outputs multipart undefined output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
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
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_DETECTION");
      fd_cls.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) det response status": (r) =>
          r.status === 200,
          "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task det response model.task": (r) =>
          r.json().model.task === "TASK_DETECTION",
          "POST /models/upload (multipart) task det response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      // Triton loading models takes time
      sleep(6)

      check(http.get(`${apiHost}/models/${model_name}`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /models/${model_name} response status`]: (r) =>
          r.status === 200,
          [`GET /models/${model_name} task`]: (r) =>
          r.json().model.task === "TASK_DETECTION",
          [`GET /models/${model_name} model.model_versions.length`]: (r) =>
          r.json().model.model_versions.length === 1,
          [`GET /models/${model_name} version model.model_versions[0].created_at`]: (r) =>
          r.json().model.model_versions[0].created_at !== undefined,
          [`GET /models/${model_name} version model.model_versions[0].updated_at`]: (r) =>
          r.json().model.model_versions[0].updated_at !== undefined,
          [`GET /models/${model_name} version model.model_versions[0].status`]: (r) =>
          r.json().model.model_versions[0].status === "STATUS_OFFLINE",
          [`GET /models/${model_name} model.id`]: (r) =>
          r.json().model.id !== undefined,
          [`GET /models/${model_name} model.full_name`]: (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }

 // Model Backend API: Get model list
  {
    group("Model Backend API: Get model list", function () {
      let fd_cls = new FormData();
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_DETECTION");
      fd_cls.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) det response status": (r) =>
          r.status === 200,
          "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task det response model.task": (r) =>
          r.json().model.task === "TASK_DETECTION",
          "POST /models/upload (multipart) task det response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      // Triton loading models takes time
      sleep(6)

      check(http.get(`${apiHost}/models`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /models response status`]: (r) =>
          r.status === 200,
          [`GET /models task`]: (r) =>
          r.json().models[0].task !== undefined,
          [`GET /models models[0].model_versions.length`]: (r) =>
          r.json().models[0].model_versions.length > 0,
          [`GET /models models[0].model_versions[0].created_at`]: (r) =>
          r.json().models[0].model_versions[0].created_at !== undefined,
          [`GET /models models[0].model_versions[0].updated_at`]: (r) =>
          r.json().models[0].model_versions[0].updated_at !== undefined,
          [`GET /models models[0].model_versions[0].status`]: (r) =>
          r.json().models[0].model_versions[0].status !== undefined,
          [`GET /models models[0].id`]: (r) =>
          r.json().models[0].id !== undefined,
          [`GET /models models[0].full_name`]: (r) =>
          r.json().models[0].full_name !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }

  // Model Backend API: update model version description
  {
    group("Model Backend API: Update model version description", function () {
      let fd_cls = new FormData();
      let model_name = randomString(10)
      fd_cls.append("name", model_name);
      fd_cls.append("description", randomString(20));
      fd_cls.append("task", "TASK_CLASSIFICATION");
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models/upload (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 1,
      });

      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
        "POST /models (multipart) cls response status": (r) =>
          r.status === 200, // TODO: update status to 201
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name !== undefined,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name !== undefined,
          "POST /models/upload (multipart) task cls response model.task": (r) =>
          r.json().model.task === "TASK_CLASSIFICATION",
          "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
          r.json().model.model_versions.length === 2,
      });

      let new_description = randomString(20)
      let payload = JSON.stringify({
        "description": new_description,
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/1`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/1 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description === new_description,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/1 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status !== undefined,
      });

      let new_description2 = randomString(20)
      payload = JSON.stringify({
        "description": new_description2,
      });
      check(http.patch(`${apiHost}/models/${model_name}/versions/2`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/versions/2 online task cls response status`]: (r) =>
          r.status === 200, // TODO: update status to 201
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.version`]: (r) =>
          r.json().model_version.version !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.model_id`]: (r) =>
          r.json().model_version.model_id !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.description`]: (r) =>
          r.json().model_version.description === new_description2,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.created_at`]: (r) =>
          r.json().model_version.created_at !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model_version.updated_at`]: (r) =>
          r.json().model_version.updated_at !== undefined,
          [`PATCH /models/${model_name}/versions/2 online task cls response model version model_version.status`]: (r) =>
          r.json().model_version.status !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${apiHost}/models/${model_name}`, null, {
        headers: genHeader(`application/json`),
      }), {
        "DELETE clean up response status": (r) =>
          r.status === 200 // TODO: update status to 204
      });

      // Triton unloading models takes time
      sleep(6)
    });
  }
}

export function teardown(data) {
  group("Model API: Delete all models created by this test", () => {
    let res = http
    .request("GET", `${apiHost}/models`, null, {
      headers: genHeader(
        "application/json"
      ),
    })
    for (const model of http
      .request("GET", `${apiHost}/models`, null, {
        headers: genHeader(
          "application/json"
        ),
      })
      .json("models")) {
      check(model, {
        "GET /clients response contents[*] id": (c) => c.id !== undefined,
      });
      check(
        http.request("DELETE", `${apiHost}/models/${model.name}`, null, {
          headers: genHeader("application/json"),
        }),
        {
          [`DELETE /models/${model.name} response status is 204`]: (r) =>
            r.status === 200, //TODO: update to 204
        }
      );
    }
  });
}
