import http from "k6/http";
import { check, group } from "k6";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
  base64_image,
} from "./helpers.js";

const apiHost = __ENV.HOSTNAME ? `http://${__ENV.HOSTNAME}:8083` : "http://model-backend:8083";

const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");
const dog_rgba_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog-rgba.png`, "b");
const cat_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/cat.jpg`, "b");
const bear_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/bear.jpg`, "b");

const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
const keypoint_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-keypoint-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const empty_response_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/empty-response-model.zip`, "b");
const model_def_name = "model-definitions/local"


export function InferModel() {
  // Model Backend API: Predict Model with classification model
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

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1,          
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images batch_outputs[1].task`]: (r) =>
          r.json().batch_outputs[1].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls multiple images batch_outputs[1].classification.category`]: (r) =>
          r.json().batch_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url cls response batch_outputs[1].classification.score`]: (r) =>
          r.json().batch_outputs[1].classification.score === 1,           
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{ "image_base64": base64_image, }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1,  
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images batch_outputs[1].task`]: (r) =>
          r.json().batch_outputs[1].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls multiple images batch_outputs[1].classification.category`]: (r) =>
          r.json().batch_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 cls response batch_outputs[1].classification.score`]: (r) =>
          r.json().batch_outputs[1].classification.score === 1,    
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1,  
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1, 
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(cat_img));
      fd.append("file", http.file(bear_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[1].task`]: (r) =>
          r.json().batch_outputs[1].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[1].classification.category`]: (r) =>
          r.json().batch_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls response batch_outputs[1].classification.score`]: (r) =>
          r.json().batch_outputs[1].classification.score === 1, 
          [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[2].task`]: (r) =>
          r.json().batch_outputs[2].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls multiple images batch_outputs[2].classification.category`]: (r) =>
          r.json().batch_outputs[2].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart cls response batch_outputs[2].classification.score`]: (r) =>
          r.json().batch_outputs[2].classification.score === 1,             
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[0].classification.category`]: (r) =>
          r.json().batch_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[0].classification.score`]: (r) =>
          r.json().batch_outputs[0].classification.score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[1].task`]: (r) =>
          r.json().batch_outputs[1].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[1].classification.category`]: (r) =>
          r.json().batch_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls response batch_outputs[1].classification.score`]: (r) =>
          r.json().batch_outputs[1].classification.score === 1, 
          [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[2].task`]: (r) =>
          r.json().batch_outputs[2].task === "CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls multiple images batch_outputs[2].classification.category`]: (r) =>
          r.json().batch_outputs[2].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart cls response batch_outputs[2].classification.score`]: (r) =>
          r.json().batch_outputs[2].classification.score === 1,   
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

  // Model Backend API: Predict Model with detection model
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

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }],
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,                    
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,          
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{ "image_base64": base64_image, }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0,         
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(cat_img));
      fd.append("file", http.file(bear_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.height === 0,             
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].category === "test",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].score === 1, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart multiple images det batch_outputs[2].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[2].detection.bounding_boxes[0].bounding_box.height === 0,   
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

  // Model Backend API: Predict Model with undefined task model
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

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined,                                                  
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].shape !== undefined,          
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{ "image_base64": base64_image, }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined,  
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 multiple images undefined batch_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].shape !== undefined,    
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined, 
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(cat_img));
      fd.append("file", http.file(bear_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].shape !== undefined,       
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].shape !== undefined,           
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs.length === 1,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[0].unspecified.raw_outputs[0].shape !== undefined, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[1].unspecified.raw_outputs[0].shape !== undefined,       
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].data !== undefined,   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].data_type === "FP32",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].name === "output",   
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart undefined batch_outputs[2].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().batch_outputs[2].unspecified.raw_outputs[0].shape !== undefined,  
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

  // Model Backend API: Predict Model with keypoint model
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

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "KEYPOINT",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint batch_outputs[0].keypoint.keypoint_groups.length`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group.length > 0,            
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url keypoint batch_outputs[0].keypoint.keypoint_groups[0].score`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].score === 1,          
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "KEYPOINT",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint batch_outputs[0].keypoint.keypoint_groups.length`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group.length > 0,            
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 keypoint batch_outputs[0].keypoint.keypoint_groups[0].score`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].score === 1,   
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
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart keypoint batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart keypoint batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "KEYPOINT",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart keypoint batch_outputs[0].keypoint.keypoint_groups.length`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart keypoint batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group.length > 0,            
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart keypoint batch_outputs[0].keypoint.keypoint_groups[0].score`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].score === 1, 
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart keypoint batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart keypoint batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "KEYPOINT",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart keypoint batch_outputs[0].keypoint.keypoint_groups.length`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart keypoint batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].keypoint_group.length > 0,            
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart keypoint batch_outputs[0].keypoint.keypoint_groups[0].score`]: (r) =>
          r.json().batch_outputs[0].keypoint.keypoint_groups[0].score === 1, 
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
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart form-data keypoint response status`]: (r) =>
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

  // Model Backend API: Predict Model with empty response
  {
    group("Model Backend API: Predict Model with empty response", function () {
      let fd_empty = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_empty.append("id", model_id);
      fd_empty.append("description", model_description);
      fd_empty.append("model_definition", model_def_name);
      fd_empty.append("content", http.file(empty_response_model, "empty-response-model.zip"));
      check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_empty.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_empty.boundary}`),
      }), {
        "POST /v1alpha/models:multipart response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models:multipart response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart response model.description": (r) =>
          r.json().model.description === model_description,
        "POST /v1alpha/models:multipart response model.model_definition": (r) =>
          r.json().model.model_definition === model_def_name,
        "POST /v1alpha/models:multipart response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /v1alpha/models:multipart response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });
      
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:deploy`, {}, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/latest`,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.id`]: (r) =>
          r.json().instance.id === "latest",
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.task`]: (r) =>
          r.json().instance.task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === model_def_name,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/latest:deploy online response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });

      // Predict with url
      let payload = JSON.stringify({
        "inputs": [{ "image_url": "https://artifacts.instill.tech/dog.jpg" }],
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,                   
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0,           
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{ "image_base64": base64_image, }]
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0,  
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
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger base64 det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger url det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0, 
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:test-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0, 
      });
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/latest:trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].category === "",
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].score === 0, 
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.top === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.left === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.width === 0,                    
        [`POST /v1alpha/models/${model_id}/instances/latest:trigger-multipart det multiple images batch_outputs[1].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[1].detection.bounding_boxes[0].bounding_box.height === 0, 
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

  // Model Backend API: Predict object detection model with 4 channel image
  {
    group("Model Backend API: Predict object detection model with 4 channel image", function () {
      let model_id = randomString(10)
      check(http.request("POST", `${apiHost}/v1alpha/models`, JSON.stringify({
        "id": model_id,
        "model_definition": "model-definitions/github",
        "configuration": {
          "repository": "instill-ai/model-yolov4"
        },
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /v1alpha/models:multipart task det response status": (r) =>
          r.status == 201,
        "POST /v1alpha/models:multipart task det response model.name": (r) =>
          r.json().model.name === `models/${model_id}`,
        "POST /v1alpha/models:multipart task det response model.uid": (r) =>
          r.json().model.uid !== undefined,
        "POST /v1alpha/models:multipart task det response model.id": (r) =>
          r.json().model.id === model_id,
        "POST /v1alpha/models:multipart task det response model.description": (r) =>
          r.json().model.description !== undefined,
        "POST /v1alpha/models:multipart task det response model.model_definition": (r) =>
          r.json().model.model_definition === "model-definitions/github",
        "POST /v1alpha/models:multipart task det response model.configuration": (r) =>
          r.json().model.configuration !== undefined,
        "POST /v1alpha/models:multipart task det response model.configuration.repository": (r) =>
          r.json().model.configuration.repository === "instill-ai/model-yolov4",
        "POST /v1alpha/models:multipart task det response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PUBLIC",
        "POST /v1alpha/models:multipart task det response model.owner": (r) =>
          r.json().model.user === 'users/local-user',
        "POST /v1alpha/models:multipart task det response model.create_time": (r) =>
          r.json().model.create_time !== undefined,
        "POST /v1alpha/models:multipart task det response model.update_time": (r) =>
          r.json().model.update_time !== undefined,
      });

      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0-cpu:deploy`, {}, {
        headers: genHeader(`application/json`),
        timeout: '600s'
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.name`]: (r) =>
          r.json().instance.name === `models/${model_id}/instances/v1.0-cpu`,
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.uid`]: (r) =>
          r.json().instance.uid !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.id`]: (r) =>
          r.json().instance.id === "v1.0-cpu",
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.state`]: (r) =>
          r.json().instance.state === "STATE_ONLINE",
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.task`]: (r) =>
          r.json().instance.task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.model_definition`]: (r) =>
          r.json().instance.model_definition === "model-definitions/github",
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.create_time`]: (r) =>
          r.json().instance.create_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.update_time`]: (r) =>
          r.json().instance.update_time !== undefined,
        [`POST /v1alpha/models/${model_id}/instances/v1.0:deploy online task det response instance.configuration`]: (r) =>
          r.json().instance.configuration !== undefined,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(dog_rgba_img));
      check(http.post(`${apiHost}/v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det batch_outputs.length`]: (r) =>
          r.json().batch_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det .batch_outputs[0].task`]: (r) =>
          r.json().batch_outputs[0].task === "DETECTION",
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det batch_outputs[0].detection.bounding_boxes.length`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes.length === 1,
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipartl det batch_outputs[0].detection.bounding_boxes[0].category`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det batch_outputs[0].detection.bounding_boxes[0].score`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].score !== undefined, 
        [`POST /v1alpha/models/${model_id}/instances/lv1.0-cpu:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.top`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.top !== 0,                    
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.left`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.left !== 0,                    
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.width`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.width !== 0,                    
        [`POST /v1alpha/models/${model_id}/instances/v1.0-cpu:test-multipart det batch_outputs[0].detection.bounding_boxes[0].bounding_box.height`]: (r) =>
          r.json().batch_outputs[0].detection.bounding_boxes[0].bounding_box.height !== 0,             
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
