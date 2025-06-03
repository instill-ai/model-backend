import http from "k6/http";
import encoding from "k6/encoding";
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
  base64_image,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"


export function InferGitHubModel(header) {
  // Model Backend API: Predict Model with MobilenetV2 model
  {
    group("Model Backend API: Predict Model with MobilenetV2 model", function () {
      let model_id = randomString(10)
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id,
        "modelDefinition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-mobilenetv2",
          "tag": "v1.0-cpu"
        },
      }), header)

      check(createModelRes, {
        "POST /v1alpha/models task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task cls response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model instance state being updated in 1 hours. Some GitHub models is huge.
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Predict with url
      let payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/tiff-sample.tiff"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score > 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        },
        {
          "classification": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score > 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[2].classification.category`]: (r) =>
          r.json().task_outputs[2].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls response task_outputs[2].classification.score`]: (r) =>
          r.json().task_outputs[2].classification.score > 0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }

  // Model Backend API: Predict Model with MobilenetV2 model
  {
    group("Model Backend API: Predict Model with MobilenetV2 DVC model", function () {
      let model_id = randomString(10)
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id,
        "modelDefinition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-mobilenetv2-dvc"
        },
      }), header)

      check(createModelRes, {
        "POST /v1alpha/models task cls response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task cls response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task cls response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model instance state being updated in 1 hours. Some GitHub models is huge.
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Predict with url
      let payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/tiff-sample.tiff"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score > 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        },
        {
          "classification": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score > 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[2].classification.category`]: (r) =>
          r.json().task_outputs[2].classification.category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls response task_outputs[2].classification.score`]: (r) =>
          r.json().task_outputs[2].classification.score > 0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }

  // Model Backend API: Predict Model with YoloV4 model
  {
    group("Model Backend API: Predict Model with YoloV4 model", function () {
      let model_id = randomString(10)
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id,
        "modelDefinition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-yolov4"
        },
      }), header)

      check(createModelRes, {
        "POST /v1alpha/models task object detection response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task det response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task det response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model instance state being updated in 1 hour
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Predict with url
      let payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        }],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        },
        {
          "detection": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        }
        ],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        },
        {
          "detection": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height > 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects.length`]: (r) =>
          r.json().task_outputs[2].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.height > 0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }

  // Model Backend API: Predict Model with YoloV4 DVC model
  {
    group("Model Backend API: Predict Model with YoloV4 DVC model", function () {
      let model_id = randomString(10)
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id,
        "modelDefinition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-yolov4-dvc"
        },
      }), header)

      check(createModelRes, {
        "POST /v1alpha/models task object detection response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models task cls response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task det response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task det response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model instance state being updated in 1 hour
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Predict with url
      let payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        }],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        },
        {
          "detection": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
          }
        }
        ],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "detection": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        },
        {
          "detection": {
            "image_base64": encoding.b64encode(constant.dog_img, "b"),
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height > 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects.length`]: (r) =>
          r.json().task_outputs[2].detection.objects.length >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].category !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.height > 0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }

  // Model Backend API: Predict Model with keypoint model
  {
    group("Model Backend API: Predict Model with keypoint model", function () {
      let model_id = randomString(10)
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
        "id": model_id,
        "modelDefinition": "model-definitions/github",
        "configuration": {
          "repository": "admin/model-yolov7-pose-dvc"
        },
      }), header)

      check(createModelRes, {
        "POST /v1alpha/models task keypoint response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models task keypoint response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task keypoint response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task keypoint response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model instance state being updated in 1 hours. Some GitHub models is huge.
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Predict with url
      let payload = JSON.stringify({
        "task_inputs": [{
          "keypoint": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dance.jpg"
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length >= 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dance.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill-ai.com/imgs/dance.jpg"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length >= 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height > 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": encoding.b64encode(constant.dance_img, "b"),
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length >= 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": encoding.b64encode(constant.dance_img, "b"),
          }
        },
        {
          "classification": {
            "image_base64": encoding.b64encode(constant.dance_img, "b"),
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length >= 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height > 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height > 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dance_img));
      fd.append("file", http.file(constant.dance_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length >= 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left >= 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height > 0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }

  // // Model Backend API: Predict Model with semantic segmentation model
  // {
  //   group("Model Backend API: Predict Model with semantic segmentation model", function () {
  //     let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
  //       "id": model_id,
  //       "modelDefinition": "model-definitions/github",
  //       "configuration": {
  //         "repository": "admin/model-semantic-segmentation-dvc"
  //       },
  //     }), {
  //       headers: genHeader("application/json"),
  //     })

  //     check(createModelRes, {
  //       "POST /v1alpha/models task semantic response status": (r) =>
  //         r.status === 201,
  //       "POST /v1alpha/models task semantic response operation.name": (r) =>
  //         r.json().operation.name !== undefined,
  //     });

  //     // Check model creation finished
  //     let currentTime = new Date().getTime();
  //     let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
  //     while (timeoutTime > currentTime) {
  //       var res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, {
  //         headers: genHeader(`application/json`),
  //       })
  //       if (res.json().operation.done === true) {
  //         break
  //       }
  //       sleep(1)
  //       currentTime = new Date().getTime();
  //     }

  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/deploy online task semantic response status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.name`]: (r) =>
  //         r.json().operation.name !== undefined,
  //       [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.metadata`]: (r) =>
  //         r.json().operation.metadata === null,
  //       [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.done`]: (r) =>
  //         r.json().operation.done === false,
  //       [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.response`]: (r) =>
  //         r.json().operation.response !== undefined,
  //     });

  //     // Check the model instance state being updated in 1 hours. Some GitHub models is huge.
  //     currentTime = new Date().getTime();
  //     timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
  //     while (timeoutTime > currentTime) {
  //       var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, {
  //         headers: genHeader(`application/json`),
  //       })
  //       if (res.json().instance.state === "STATE_ONLINE") {
  //         break
  //       }
  //       sleep(1)
  //       currentTime = new Date().getTime();
  //     }

  //     // Predict with url
  //     let payload = JSON.stringify({
  //       "task_inputs": [{
  //         "classification": {
  //           "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
  //         }
  //       }]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task`]: (r) =>
  //         r.json().task === "TASK_SEMANTIC_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
  //     });

  //     // Predict multiple images with url
  //     payload = JSON.stringify({
  //       "task_inputs": [{
  //           "classification": {
  //             "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
  //           }
  //         },
  //         {
  //           "classification": {
  //             "image_url": "https://artifacts.instill-ai.com/imgs/tiff-sample.tiff"
  //           }
  //         }
  //       ]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic multiple images status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task`]: (r) =>
  //         r.json().task === "TASK_SEMANTIC_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 2,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
  //     });

  //     // Predict with base64
  //     payload = JSON.stringify({
  //       "task_inputs": [{
  //         "classification": {
  //           "image_base64": encoding.b64encode(constant.dog_img, "b"),
  //         }
  //       }]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task`]: (r) =>
  //         r.json().task === "TASK_SEMANTIC_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
  //     });

  //     // Predict multiple images with base64
  //     payload = JSON.stringify({
  //       "task_inputs": [{
  //           "classification": {
  //             "image_base64": encoding.b64encode(constant.dog_img, "b"),
  //           }
  //         },
  //         {
  //           "classification": {
  //             "image_base64": encoding.b64encode(constant.dog_img, "b"),
  //           }
  //         }
  //       ]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic multiple images status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task`]: (r) =>
  //         r.json().task === "TASK_SEMANTIC_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 2,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
  //     });

  //     // Predict with multiple-part
  //     fd = new FormData();
  //     fd.append("file", http.file(constant.dog_img));
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
  //       headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task`]: (r) =>
  //         r.json().task === "TASK_SEMANTIC_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
  //     });

  //     // Predict multiple images with multiple-part
  //     fd = new FormData();
  //     fd.append("file", http.file(constant.dog_img));
  //     fd.append("file", http.file(constant.cat_img));
  //     fd.append("file", http.file(constant.bear_img));
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
  //       headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task`]: (r) =>
  //         r.json().task === "TASK_SEMANTIC_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 3,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs`]: (r) =>
  //         r.json().task_outputs[2].semantic_segmentation.stuffs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs[0].category`]: (r) =>
  //         r.json().task_outputs[2].semantic_segmentation.stuffs[0].category === "tree",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs[0].rle`]: (r) =>
  //         r.json().task_outputs[2].semantic_segmentation.stuffs[0].rle !== undefined,
  //     });

  //     // clean up
  //     check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       "DELETE clean up response status": (r) =>
  //         r.status === 204
  //     });

  //   });
  // }

  // // Model Backend API: Predict Model with instance segmentation model
  // {
  //   group("Model Backend API: Predict Model with instance segmentation model", function () {
  //     let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`, JSON.stringify({
  //       "id": model_id,
  //       "modelDefinition": "model-definitions/github",
  //       "configuration": {
  //         "repository": "admin/model-instance-segmentation-dvc"
  //       },
  //     }), {
  //       headers: genHeader("application/json"),
  //     })

  //     check(createModelRes, {
  //       "POST /v1alpha/models task instance response status": (r) =>
  //         r.status === 201,
  //       "POST /v1alpha/models task instance response operation.name": (r) =>
  //         r.json().operation.name !== undefined,
  //     });

  //     // Check model creation finished
  //     let currentTime = new Date().getTime();
  //     let timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
  //     while (timeoutTime > currentTime) {
  //       var res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, {
  //         headers: genHeader(`application/json`),
  //       })
  //       if (res.json().operation.done === true) {
  //         break
  //       }
  //       sleep(1)
  //       currentTime = new Date().getTime();
  //     }

  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/deploy online task instance response status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/deploy online task instance response operation.name`]: (r) =>
  //         r.json().operation.name !== undefined,
  //       [`POST /v1alpha/models/${model_id}/deploy online task instance response operation.metadata`]: (r) =>
  //         r.json().operation.metadata === null,
  //       [`POST /v1alpha/models/${model_id}/deploy online task instance response operation.done`]: (r) =>
  //         r.json().operation.done === false,
  //       [`POST /v1alpha/models/${model_id}/deploy online task instance response operation.response`]: (r) =>
  //         r.json().operation.response !== undefined,
  //     });

  //     // Check the model instance state being updated in 1 hours. Some GitHub models is huge.
  //     currentTime = new Date().getTime();
  //     timeoutTime = new Date().getTime() + 1 * 60 * 60 * 1000;
  //     while (timeoutTime > currentTime) {
  //       var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, {
  //         headers: genHeader(`application/json`),
  //       })
  //       if (res.json().instance.state === "STATE_ONLINE") {
  //         break
  //       }
  //       sleep(1)
  //       currentTime = new Date().getTime();
  //     }

  //     // Predict with url
  //     let payload = JSON.stringify({
  //       "task_inputs": [{
  //         "classification": {
  //           "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
  //         }
  //       }]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger url instance status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task`]: (r) =>
  //         r.json().task === "TASK_INSTANCE_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].score > 0.0,
  //     });

  //     // Predict multiple images with url
  //     payload = JSON.stringify({
  //       "task_inputs": [{
  //           "classification": {
  //             "image_url": "https://artifacts.instill-ai.com/imgs/dog.jpg"
  //           }
  //         },
  //         {
  //           "classification": {
  //             "image_url": "https://artifacts.instill-ai.com/imgs/tiff-sample.tiff"
  //           }
  //         }
  //       ]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple images status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task`]: (r) =>
  //         r.json().task === "TASK_INSTANCE_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 2,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].score > 0.0,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].score > 0.0,
  //     });

  //     // Predict with base64
  //     payload = JSON.stringify({
  //       "task_inputs": [{
  //         "classification": {
  //           "image_base64": encoding.b64encode(constant.dog_img, "b"),
  //         }
  //       }]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task`]: (r) =>
  //         r.json().task === "TASK_INSTANCE_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].score > 0.0,
  //     });

  //     // Predict multiple images with base64
  //     payload = JSON.stringify({
  //       "task_inputs": [{
  //           "classification": {
  //             "image_base64": encoding.b64encode(constant.dog_img, "b"),
  //           }
  //         },
  //         {
  //           "classification": {
  //             "image_base64": encoding.b64encode(constant.dog_img, "b"),
  //           }
  //         }
  //       ]
  //     });
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple images status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task`]: (r) =>
  //         r.json().task === "TASK_INSTANCE_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 2,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].score > 0.0,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].score > 0.0,
  //     });

  //     // Predict with multiple-part
  //     fd = new FormData();
  //     fd.append("file", http.file(constant.dog_img));
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
  //       headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task`]: (r) =>
  //         r.json().task === "TASK_INSTANCE_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].score > 0.0,
  //     });

  //     // Predict multiple images with multiple-part
  //     fd = new FormData();
  //     fd.append("file", http.file(constant.dog_img));
  //     fd.append("file", http.file(constant.cat_img));
  //     fd.append("file", http.file(constant.bear_img));
  //     check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
  //       headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
  //     }), {
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images status`]: (r) =>
  //         r.status === 200,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task`]: (r) =>
  //         r.json().task === "TASK_INSTANCE_SEGMENTATION",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance task_outputs.length`]: (r) =>
  //         r.json().task_outputs.length === 3,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[0].instance_segmentation.objects[0].score > 0.0,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[1].instance_segmentation.objects[0].score > 0.0,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects.length === 1,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].rle`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].rle !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.top !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.left !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.height !== undefined,
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].category`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].category === "dog",
  //       [`POST /v1alpha/models/${model_id}/trigger-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].score`]: (r) =>
  //         r.json().task_outputs[2].instance_segmentation.objects[0].score > 0.0,
  //     });

  //     // clean up
  //     check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, {
  //       headers: genHeader(`application/json`),
  //     }), {
  //       "DELETE clean up response status": (r) =>
  //         r.status === 204
  //     });

  //   });
  // }
}
