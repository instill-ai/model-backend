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
  base64_image,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"


export function InferModel(header) {
  // Model Backend API: Predict Model with classification model
  {
    group("Model Backend API: Predict Model with classification model", function () {
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
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createClsModelRes.json().operation.name}`, header)
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

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
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
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger url cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/tiff-sample.tiff"
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
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/trigger url cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger url cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score === 1,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger base64 cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score === 1,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart cls status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart cls task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/test-multipart cls task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart cls task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/test-multipart cls task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
      });
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
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images task`]: (r) =>
          r.json().task === "TASK_CLASSIFICATION",
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images task_outputs[0].classification.category`]: (r) =>
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/test-multipart cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart cls multiple images task_outputs[2].classification.category`]: (r) =>
          r.json().task_outputs[2].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/test-multipart cls response task_outputs[2].classification.score`]: (r) =>
          r.json().task_outputs[2].classification.score === 1,
      });
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
          r.json().task_outputs[0].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[0].classification.score`]: (r) =>
          r.json().task_outputs[0].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[1].classification.category`]: (r) =>
          r.json().task_outputs[1].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls response task_outputs[1].classification.score`]: (r) =>
          r.json().task_outputs[1].classification.score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images task_outputs[2].classification.category`]: (r) =>
          r.json().task_outputs[2].classification.category === "match",
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls response task_outputs[2].classification.score`]: (r) =>
          r.json().task_outputs[2].classification.score === 1,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
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
      fd_det.append("content", http.file(constant.det_model, "dummy-det-model.zip"));

      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task det response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task det response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 20000;
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

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }
        ],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });
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
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects.length`]: (r) =>
          r.json().task_outputs[2].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.height === 0,
      });
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
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects.length`]: (r) =>
          r.json().task_outputs[1].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects.length`]: (r) =>
          r.json().task_outputs[2].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].category === "test",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images det task_outputs[2].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].detection.objects[0].bounding_box.height === 0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
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
      fd.append("content", http.file(constant.unspecified_model, "dummy-unspecified-model.zip"));

      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task unspecified response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task unspecified response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task unspecified response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task unspecified response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/trigger url undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger url undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger url undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/tiff-sample.tiff"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger url multiple images undefined task_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].shape !== undefined,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger base64 undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger base64 multiple images undefined task_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].shape !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/test-multipart undefined task_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].shape !== undefined,
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task`]: (r) =>
          r.json().task === "TASK_UNSPECIFIED",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs.length`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[0].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[0].unspecified.raw_outputs[0].shape !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[1].unspecified.raw_outputs[0].data`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[1].unspecified.raw_outputs[0].data_type`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].data_type === "FP32",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[1].unspecified.raw_outputs[0].name`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].name === "output",
        [`POST /v1alpha/models/${model_id}/trigger-multipart undefined task_outputs[1].unspecified.raw_outputs[0].shape`]: (r) =>
          r.json().task_outputs[1].unspecified.raw_outputs[0].shape !== undefined,
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
      let fd_keypoint = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd_keypoint.append("id", model_id);
      fd_keypoint.append("description", model_description);
      fd_keypoint.append("model_definition", model_def_name);
      fd_keypoint.append("content", http.file(constant.keypoint_model, "dummy-keypoint-model.zip"));

      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd_keypoint.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_keypoint.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task keypoint response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task keypoint response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
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

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
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
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger url keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": { "image_url": "https://artifacts.instill.tech/imgs/dog.jpg" }
        },
        {
          "classification": { "image_url": "https://artifacts.instill.tech/imgs/dog.jpg" }
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
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images url keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height === 1,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger multiple images base64 keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height === 1,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
      });
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
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task`]: (r) =>
          r.json().task === "TASK_KEYPOINT",
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height === 1,
      });
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
          r.json().task_outputs[0].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[0].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].keypoint.objects[0].bounding_box.height === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects.length`]: (r) =>
          r.json().task_outputs[1].keypoint.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].keypoints`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].keypoints.length > 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].score`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].score === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.top === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.left === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.width === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images keypoint task_outputs[1].keypoint.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].keypoint.objects[0].bounding_box.height === 1,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
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
      fd_empty.append("content", http.file(constant.empty_response_model, "empty-response-model.zip"));

      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd_empty.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_empty.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task det empty response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task det empty response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task det empty response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task det empty response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }
        ],
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
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
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger base64 det multiple images task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger url det multiple images task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
      });

      // Predict with multiple-part
      let fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart det status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart det task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });
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
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/test-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task`]: (r) =>
          r.json().task === "TASK_DETECTION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[0].detection.objects.length`]: (r) =>
          r.json().task_outputs[0].detection.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[0].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[0].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].detection.objects[0].bounding_box.height === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det task_outputs[0].detection.objects[0].category`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].category === "",
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[1].detection.objects[0].score`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].score === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.top === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.left === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.width`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.width === 0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart det multiple images task_outputs[1].detection.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].detection.objects[0].bounding_box.height === 0,
      });


      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }

  // Model Backend API: Predict Model with semantic segmentation model
  {
    group("Model Backend API: Predict Model with semantic segmentation model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.semantic_segmentation_model, "dummy-semantic_segmentation_model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task semantic response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task semantic response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task semantic response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url semantic status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/tiff-sample.tiff"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url semantic multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger url semantic task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger base64 semantic task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart semantic status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/test-multipart semantic task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/test-multipart semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[2].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[2].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/test-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[2].semantic_segmentation.stuffs[0].rle !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart cls multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task`]: (r) =>
          r.json().task === "TASK_SEMANTIC_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[0].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[0].semantic_segmentation.stuffs[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[1].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[1].semantic_segmentation.stuffs[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs`]: (r) =>
          r.json().task_outputs[2].semantic_segmentation.stuffs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs[0].category`]: (r) =>
          r.json().task_outputs[2].semantic_segmentation.stuffs[0].category === "tree",
        [`POST /v1alpha/models/${model_id}/trigger-multipart semantic multiple task_outputs[2].semantic_segmentation.stuffs[0].rle`]: (r) =>
          r.json().task_outputs[2].semantic_segmentation.stuffs[0].rle !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }

  // Model Backend API: Predict Model with instance segmentation model
  {
    group("Model Backend API: Predict Model with instance segmentation model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.instance_segmentation_model, "dummy-instance-segmentation-model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task semantic response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task semantic response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
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
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url instance status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url instance task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger url instance task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
      });

      // Predict multiple images with url
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        },
        {
          "classification": {
            "image_url": "https://artifacts.instill.tech/imgs/tiff-sample.tiff"
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger url instance multiple task_outputs[1].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].score === 1.0,
      });

      // Predict with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 instance status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger base64 instance task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
      });

      // Predict multiple images with base64
      payload = JSON.stringify({
        "task_inputs": [{
          "classification": {
            "image_base64": base64_image,
          }
        },
        {
          "classification": {
            "image_base64": base64_image,
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 2,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger base64 instance multiple task_outputs[1].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].score === 1.0,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart instance status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
      });

      // Predict multiple images with multiple-part
      fd = new FormData();
      fd.append("file", http.file(constant.dog_img));
      fd.append("file", http.file(constant.cat_img));
      fd.append("file", http.file(constant.bear_img));
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].score === 1.0,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/test-multipart instance multiple images task_outputs[2].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].score === 1.0,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task`]: (r) =>
          r.json().task === "TASK_INSTANCE_SEGMENTATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 3,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[0].instance_segmentation.objects[0].score === 1.0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[0].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[1].instance_segmentation.objects[0].score === 1.0,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].rle`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].rle !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.top`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.top !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.left`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.left !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].bounding_box.height`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].bounding_box.height !== undefined,
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].category`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].category === "dog",
        [`POST /v1alpha/models/${model_id}/trigger-multipart multiple images task_outputs[2].instance_segmentation.objects[0].score`]: (r) =>
          r.json().task_outputs[2].instance_segmentation.objects[0].score === 1.0,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    });
  }

  // Model Backend API: Predict Model with text to image model
  {
    group("Model Backend API: Predict Model with text to image model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.text_to_image_model, "dummy-text-to-image-model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task text to image response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task text to image response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task text to image response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task text to image response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Inference with only required input
      let payload = JSON.stringify({
        "task_inputs": [{
          "text_to_image": {
            "prompt": "hello this is a test"
          }
        }]
      })

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger text to image status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger text to image task`]: (r) =>
          r.json().task === "TASK_TEXT_TO_IMAGE",
        [`POST /v1alpha/models/${model_id}/trigger text to image task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger text to image task_outputs[0].text_to_image.images.length`]: (r) =>
          r.json().task_outputs[0].text_to_image.images.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger text to image task_outputs[0].text_to_image.images[0]`]: (r) =>
          r.json().task_outputs[0].text_to_image.images[0] !== undefined,
      });

      // Inference with multiple samples, samples = 2
      let num_samples = 2
      payload = JSON.stringify({
        "task_inputs": [{
          "text_to_image": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg",
            "steps": "1",
            "cfg_scale": "5.5",
            "seed": "1",
            "samples": `${num_samples}`
          }
        }]
      });

      let resp = http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header)

      check(resp, {
        [`POST /v1alpha/models/${model_id}/trigger text to image status [with multiple samples]`]: (r) =>
          r.status === 400,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart text to image status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart text to image task`]: (r) =>
          r.json().task === "TASK_TEXT_TO_IMAGE",
        [`POST /v1alpha/models/${model_id}/test-multipart text to image task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart text to image task_outputs[0].text_to_image.images`]: (r) =>
          r.json().task_outputs[0].text_to_image.images.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart text to image task_outputs[0].text_to_image.images[0]`]: (r) =>
          r.json().task_outputs[0].text_to_image.images[0] !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart text to image status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart text to image task`]: (r) =>
          r.json().task === "TASK_TEXT_TO_IMAGE",
        [`POST /v1alpha/models/${model_id}/trigger-multipart text to image task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart text to image task_outputs[0].text_to_image.images`]: (r) =>
          r.json().task_outputs[0].text_to_image.images.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart text to image task_outputs[0].text_to_image.images[0]`]: (r) =>
          r.json().task_outputs[0].text_to_image.images[0] !== undefined,
      });


      // Invalid cases: inference with multiple parameters
      payload = JSON.stringify({
        "task_inputs": [{
          "text_to_image": {
            "prompt": "hello this is a test",
            "steps": "1",
            "cfg_scale": "5.5",
            "seed": "1",
            "samples": `${num_samples}`
          }
        },
        {
          "text_to_image": {
            "prompt": "hello this is a test",
            "steps": "1",
            "cfg_scale": "5.5",
            "seed": "1",
            "samples": `${num_samples}`
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger text to image status [with multiple prompt]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("prompt", "hello this is a test");
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart text to image status [with multiple prompts]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("steps", 1);
      fd.append("steps", 1);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart text to image status [with multiple steps]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("samples", 1);
      fd.append("samples", 1);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart text to image status [with multiple samples]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("seed", 1);
      fd.append("seed", 1);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart text to image status [with multiple seed]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("cfg_scale", 1.0);
      fd.append("cfg_scale", 1.0);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart text to image status [with multiple cfg_scale]`]: (r) =>
          r.status === 400,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    })
  }
  // Model Backend API: Predict Model with image to image model
  {
    group("Model Backend API: Predict Model with image to image model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.image_to_image_model, "dummy-image-to-image-model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task text to image response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task text to image response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task image to image response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task image to image response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Inference with only required input
      let payload = JSON.stringify({
        "task_inputs": [{
          "image_to_image": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg"
          }
        }]
      })

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger image to image status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger image to image task`]: (r) =>
          r.json().task === "TASK_IMAGE_TO_IMAGE",
        [`POST /v1alpha/models/${model_id}/trigger image to image task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger image to image task_outputs[0].image_to_image.images.length`]: (r) =>
          r.json().task_outputs[0].image_to_image.images.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger image to image task_outputs[0].image_to_image.images[0]`]: (r) =>
          r.json().task_outputs[0].image_to_image.images[0] !== undefined,
      });

      // Inference with multiple samples, samples = 2
      let num_samples = 2
      payload = JSON.stringify({
        "task_inputs": [{
          "image_to_image": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg",
            "steps": "1",
            "cfg_scale": "5.5",
            "seed": "1",
            "samples": `${num_samples}`
          }
        }]
      });

      let resp = http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header)

      check(resp, {
        [`POST /v1alpha/models/${model_id}/trigger image to image status [with multiple samples]`]: (r) =>
          r.status === 400,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart image to image status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart image to image task`]: (r) =>
          r.json().task === "TASK_IMAGE_TO_IMAGE",
        [`POST /v1alpha/models/${model_id}/test-multipart image to image task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart image to image task_outputs[0].image_to_image.images`]: (r) =>
          r.json().task_outputs[0].image_to_image.images.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart image to image task_outputs[0].image_to_image.images[0]`]: (r) =>
          r.json().task_outputs[0].image_to_image.images[0] !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart image to image status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart image to image task`]: (r) =>
          r.json().task === "TASK_IMAGE_TO_IMAGE",
        [`POST /v1alpha/models/${model_id}/trigger-multipart image to image task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart image to image task_outputs[0].image_to_image.images`]: (r) =>
          r.json().task_outputs[0].image_to_image.images.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart image to image task_outputs[0].image_to_image.images[0]`]: (r) =>
          r.json().task_outputs[0].image_to_image.images[0] !== undefined,
      });


      // Invalid cases: inference with multiple parameters
      payload = JSON.stringify({
        "task_inputs": [{
          "imaga_to_image": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg",
            "steps": "1",
            "cfg_scale": "5.5",
            "seed": "1",
            "samples": `${num_samples}`
          }
        },
        {
          "image_to_image": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg",
            "steps": "1",
            "cfg_scale": "5.5",
            "seed": "1",
            "samples": `${num_samples}`
          }
        }
        ]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger image to image status [with multiple prompt]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart image to image status [with multiple prompts]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      fd.append("steps", 1);
      fd.append("steps", 1);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart image to image status [with multiple steps]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      fd.append("samples", 1);
      fd.append("samples", 1);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart image to image status [with multiple samples]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      fd.append("seed", 1);
      fd.append("seed", 1);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart image to image status [with multiple seed]`]: (r) =>
          r.status === 400,
      });

      fd = new FormData();
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      fd.append("cfg_scale", 1.0);
      fd.append("cfg_scale", 1.0);
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart image to image status [with multiple cfg_scale]`]: (r) =>
          r.status === 400,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });

    })
  }
  // Model Backend API: Predict Model with text generation model
  {
    group("Model Backend API: Predict Model with text generation model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.text_generation_model, "dummy-text-generation-model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task text generation response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task text generation response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task generation response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task generation response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Inference with only required input
      let payload = JSON.stringify({
        "task_inputs": [{
          "text_generation": {
            "prompt": "hello this is a test",
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url text generation status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url text generation task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION",
        [`POST /v1alpha/models/${model_id}/trigger url text generation task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url text generation task_outputs[0].text_generation.text`]: (r) =>
          r.json().task_outputs[0].text_generation.text !== undefined,
      });

      // Inference with multiple samples
      payload = JSON.stringify({
        "task_inputs": [{
          "text_generation": {
            "prompt": "hello this is a test",
            "max_new_tokens": "50",
            "temperature": "0.8",
            "top_k": "2",
            "seed": "0",
            "extra_params": [
              {
                "param_name": "test_param1",
                "param_value": "test_value_1"
              }, {
                "param_name": "test_param2",
                "param_value": "test_value_2"
              },
            ]
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url text generation input multiple params status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url text generation task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION",
        [`POST /v1alpha/models/${model_id}/trigger url text generation task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url text generation multiple params task_outputs[0].text_generation.text`]: (r) =>
          r.json().task_outputs[0].text_generation.text !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("max_new_tokens", "50");
      fd.append("temperature", "0.8");
      fd.append("top_k", "2");
      fd.append("seed", "0");

      // For extra_params, you need to append them as JSON string because FormData does not support object directly
      const extraParams = [
        {
          "param_name": "test_param1",
          "param_value": "test_value_1"
        },
        {
          "param_name": "test_param2",
          "param_value": "test_value_2"
        }
      ];
      fd.append("extra_params", JSON.stringify(extraParams));

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart instance status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION",
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].text_generation.text`]: (r) =>
          r.json().task_outputs[0].text_generation.text !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION",
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].text_generation.text`]: (r) =>
          r.json().task_outputs[0].text_generation.text !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
  // Model Backend API: Predict Model with text generation chat model
  {
    group("Model Backend API: Predict Model with text generation chat model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.text_generation_chat_model, "dummy-text-generation-chat-model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task text generation chat response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task text generation chat response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task generation chat response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task generation chat  response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Inference with only required input
      let payload = JSON.stringify({
        "task_inputs": [{
          "text_generation_chat": {
            "conversation": [
              {
                "role": "ADMIN",
                "content": "You are an integration test bot",
              }, {
                "role": "ASSIST",
                "content": "What can I help you?",
              }, {
                "role": "USER",
                "content": "Test it",
              }
            ]
          }
        }]
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION_CHAT",
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task_outputs[0].text_generation_chat.text`]: (r) =>
          r.json().task_outputs[0].text_generation_chat.text !== undefined,
      });

      // Inference with multiple samples
      payload = JSON.stringify({
        "task_inputs": [{
          "text_generation_chat": {
            "conversation": [
              {
                "role": "ADMIN",
                "content": "You are an integration test bot",
              }, {
                "role": "ASSIST",
                "content": "What can I help you?",
              }, {
                "role": "USER",
                "content": "Test it",
              }
            ],
            "max_new_tokens": "50",
            "temperature": "0.8",
            "top_k": "2",
            "seed": "0",
            "extra_params": [
              {
                "param_name": "test_param1",
                "param_value": "test_value_1"
              }, {
                "param_name": "test_param2",
                "param_value": "test_value_2"
              },
            ]
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat input multiple params status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION_CHAT",
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat multiple params task_outputs[0].text_generation_chat.text`]: (r) =>
          r.json().task_outputs[0].text_generation_chat.text !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      const conversation = [
        {
          "role": "ADMIN",
          "content": "You are an integration test bot",
        }, {
          "role": "ASSIST",
          "content": "What can I help you?",
        }, {
          "role": "USER",
          "content": "Test it",
        }
      ]
      fd.append("conversation", JSON.stringify(conversation));
      fd.append("max_new_tokens", "50");
      fd.append("temperature", "0.8");
      fd.append("top_k", "2");
      fd.append("seed", "0");

      // For extra_params, you need to append them as JSON string because FormData does not support object directly
      const extraParams = [
        {
          "param_name": "test_param1",
          "param_value": "test_value_1"
        },
        {
          "param_name": "test_param2",
          "param_value": "test_value_2"
        }
      ];
      fd.append("extra_params", JSON.stringify(extraParams));

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart instance status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION_CHAT",
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].text_generation_chat.text`]: (r) =>
          r.json().task_outputs[0].text_generation_chat.text !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task`]: (r) =>
          r.json().task === "TASK_TEXT_GENERATION_CHAT",
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].text_generation_chat.text`]: (r) =>
          r.json().task_outputs[0].text_generation_chat.text !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
  // Model Backend API: Predict Model with visual question answering model
  {
    group("Model Backend API: Predict Model with visual question answering model", function () {
      let fd = new FormData();
      let model_id = randomString(10)
      let model_description = randomString(20)
      fd.append("id", model_id);
      fd.append("description", model_description);
      fd.append("model_definition", model_def_name);
      fd.append("content", http.file(constant.visual_question_answering, "dummy-visual-question-answering-model.zip"));
      let createModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      })
      check(createModelRes, {
        "POST /v1alpha/models/multipart task visual question answering response status": (r) =>
          r.status === 201,
        "POST /v1alpha/models/multipart task visual question answering response operation.name": (r) =>
          r.json().operation.name !== undefined,
      });

      // Check model creation finished
      let currentTime = new Date().getTime();
      let timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        let res = http.get(`${constant.apiPublicHost}/v1alpha/${createModelRes.json().operation.name}`, header)
        if (res.json().operation.done === true) {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/deploy`, {}, header), {
        [`POST /v1alpha/models/${model_id}/deploy online task visual question answering response status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/deploy online task visual question answering response operation.name`]: (r) =>
          r.json().model_id === model_id
      });

      // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
      currentTime = new Date().getTime();
      timeoutTime = new Date().getTime() + 120000;
      while (timeoutTime > currentTime) {
        var res = http.get(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/watch`, header)
        if (res.json().state === "STATE_ONLINE") {
          break
        }
        sleep(1)
        currentTime = new Date().getTime();
      }

      // Inference with only required input
      let payload = JSON.stringify({
        "task_inputs": [{
          "visual_question_answering": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg",
          }
        }]
      });

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task`]: (r) =>
          r.json().task === "TASK_VISUAL_QUESTION_ANSWERING",
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url text generation chat task_outputs[0].visual_question_answering.text`]: (r) =>
          r.json().task_outputs[0].visual_question_answering.text !== undefined,
      });

      // Inference with multiple samples
      payload = JSON.stringify({
        "task_inputs": [{
          "visual_question_answering": {
            "prompt": "hello this is a test",
            "prompt_image_url": "https://artifacts.instill.tech/imgs/dog.jpg",
            "max_new_tokens": "50",
            "temperature": "0.8",
            "top_k": "2",
            "seed": "0",
            "extra_params": [
              {
                "param_name": "test_param1",
                "param_value": "test_value_1"
              }, {
                "param_name": "test_param2",
                "param_value": "test_value_2"
              },
            ]
          }
        }]
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger`, payload, header), {
        [`POST /v1alpha/models/${model_id}/trigger url visual question answering task input multiple params status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger url visual question answering task`]: (r) =>
          r.json().task === "TASK_VISUAL_QUESTION_ANSWERING",
        [`POST /v1alpha/models/${model_id}/trigger url visual question answering task task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger url visual question answering task multiple params task_outputs[0].visual_question_answering.text`]: (r) =>
          r.json().task_outputs[0].visual_question_answering.text !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("prompt", "hello this is a test");
      fd.append("prompt_image_url", "https://artifacts.instill.tech/imgs/dog.jpg");
      fd.append("max_new_tokens", "50");
      fd.append("temperature", "0.8");
      fd.append("top_k", "2");
      fd.append("seed", "0");

      // For extra_params, you need to append them as JSON string because FormData does not support object directly
      const extraParams = [
        {
          "param_name": "test_param1",
          "param_value": "test_value_1"
        },
        {
          "param_name": "test_param2",
          "param_value": "test_value_2"
        }
      ];
      fd.append("extra_params", JSON.stringify(extraParams));

      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/test-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/test-multipart instance status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task`]: (r) =>
          r.json().task === "TASK_VISUAL_QUESTION_ANSWERING",
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/test-multipart instance task_outputs[0].visual_question_answering.text`]: (r) =>
          r.json().task_outputs[0].visual_question_answering.text !== undefined,
      });
      check(http.post(`${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}/trigger-multipart`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`, header.headers.Authorization),
      }), {
        [`POST /v1alpha/models/${model_id}/trigger-multipart status`]: (r) =>
          r.status === 200,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task`]: (r) =>
          r.json().task === "TASK_VISUAL_QUESTION_ANSWERING",
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs.length`]: (r) =>
          r.json().task_outputs.length === 1,
        [`POST /v1alpha/models/${model_id}/trigger-multipart task_outputs[0].visual_question_answering.text`]: (r) =>
          r.json().task_outputs[0].visual_question_answering.text !== undefined,
      });

      // clean up
      check(http.request("DELETE", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${model_id}`, null, header), {
        "DELETE clean up response status": (r) =>
          r.status === 204
      });
    });
  }
}
