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
      check(http.request("GET", `${apiHost}/__liveness`), {
        "GET /__liveness response status is 200": (r) => r.status === 200,
      });
      check(http.request("GET", `${apiHost}/__readiness`), {
        "GET /__readiness response status is 200": (r) => r.status === 200,
      });
    });
  }

  // Model Backend API: upload model
  {
    group("Model Backend API: Upload a model", function () {
      let fd_cls = new FormData();
      let model_name_cls = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", model_name_cls);
      fd_cls.append("description", model_description);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
          headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /models github task cls response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === model_name_cls,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name_cls}`,
          "POST /models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });

      let fd_det = new FormData();
      let model_name_det = randomString(10)
      model_description = randomString(20)
      fd_det.append("name", model_name_det);
      fd_det.append("description", model_description);
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
          "POST /models/upload (multipart) task det response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task det response model.name": (r) =>
          r.json().model.name === model_name_det,
          "POST /models/upload (multipart) task det response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name_det}`,
          "POST /models/upload (multipart) task det response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task det response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task det response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task det response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task det response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task det response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task det response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task det response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task det response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task det response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task det response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task det response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task det response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task det response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_DETECTION", 
          "POST /models/upload (multipart) task det response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task det response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task det response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task det response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,                               
      });

      let fd_unspecified = new FormData();
      let model_name_unspecified = randomString(10)
      model_description = randomString(20)
      fd_unspecified.append("name", model_name_unspecified);
      fd_unspecified.append("description", model_description);
      fd_unspecified.append("content", http.file(unspecified_model, "dummy-unspecified-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /models/upload (multipart) task unspecified response status": (r) =>
        r.status === 200, 
        "POST /models/upload (multipart) task unspecified response model.name": (r) =>
        r.json().model.name === model_name_unspecified,
        "POST /models/upload (multipart) task unspecified response model.full_name": (r) =>
        r.json().model.full_name === `local-user/${model_name_unspecified}`,
        "POST /models/upload (multipart) task unspecified response model.id": (r) =>
        r.json().model.id !== undefined,
        "POST /models/upload (multipart) task unspecified response model.description": (r) =>
        r.json().model.description === model_description,
        "POST /models/upload (multipart) task unspecified response model.source": (r) =>
        r.json().model.source === "SOURCE_LOCAL",
        "POST /models/upload (multipart) task unspecified response model.visibility": (r) =>
        r.json().model.visibility === "VISIBILITY_PRIVATE",
        "POST /models/upload (multipart) task unspecified response model.owner.id": (r) =>
        r.json().model.owner.id !== undefined,
        "POST /models/upload (multipart) task unspecified response model.owner.username": (r) =>
        r.json().model.owner.username === "local-user",
        "POST /models/upload (multipart) task unspecified response model.owner.type": (r) =>
        r.json().model.owner.type === "user",
        "POST /models/upload (multipart) task unspecified response model.created_at": (r) =>
        r.json().model.created_at !== undefined,
        "POST /models/upload (multipart) task unspecified response model.updated_at": (r) =>
        r.json().model.updated_at !== undefined,
        "POST /models/upload (multipart) task unspecified response model.instances.length": (r) =>
        r.json().model.instances.length === 1,
        "POST /models/upload (multipart) task unspecified response model.instances[0].name": (r) =>
        r.json().model.instances[0].name === "latest",
        "POST /models/upload (multipart) task unspecified response model.instances[0].model_definition_name": (r) =>
        r.json().model.instances[0].model_definition_name === r.json().model.name,
        "POST /models/upload (multipart) task unspecified response model.instances[0].status": (r) =>
        r.json().model.instances[0].status === "STATUS_OFFLINE",          
        "POST /models/upload (multipart) task unspecified response model.instances[0].task": (r) =>
        r.json().model.instances[0].task === "TASK_UNSPECIFIED", 
        "POST /models/upload (multipart) task unspecified response model.instances[0].model_definition_id": (r) =>
        r.json().model.instances[0].model_definition_id === r.json().model.id,     
        "POST /models/upload (multipart) task unspecified response model.instances[0].model_definition_source": (r) =>
        r.json().model.instances[0].model_definition_source === r.json().model.source,    
        "POST /models/upload (multipart) task unspecified response model.instances[0].created_at": (r) =>
        r.json().model.instances[0].created_at !== undefined,  
        "POST /models/upload (multipart) task unspecified response model.instances[0].updated_at": (r) =>
        r.json().model.instances[0].updated_at !== undefined,   
      });

      check(http.request("POST", `${apiHost}/models/upload`, fd_unspecified.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_unspecified.boundary}`),
      }), {
        "POST /models/upload (multipart) already existed response status 409": (r) =>
        r.status === 409, 
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

  // Model Backend API: load model online
  {
    group("Model Backend API: Load model online", function () {
      let fd_cls = new FormData();
      let model_name = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", model_name);
      fd_cls.append("description", model_description);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
          headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /models/upload (multipart) task cls response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });
      sleep(5) // Triton loading models takes time

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/instances/latest`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/instances/latest online task cls response status`]: (r) =>
          r.status === 200,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.name`]: (r) =>
          r.json().instance.name === "latest",
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.model_definition_name`]: (r) =>
          r.json().instance.model_definition_name === model_name,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.created_at`]: (r) =>
          r.json().instance.created_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.updated_at`]: (r) =>
          r.json().instance.updated_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.status`]: (r) =>
          r.json().instance.status === "STATUS_ONLINE",
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.model_definition_source`]: (r) =>
          r.json().instance.model_definition_source === "SOURCE_LOCAL",    
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.model_definition_id`]: (r) =>
          r.json().instance.model_definition_id !== undefined,                
      });
      sleep(5) // Triton loading models takes time

      payload = JSON.stringify({
        "status": "STATUS_OFFLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/instances/latest`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/instances/latest offline task cls response status`]: (r) =>
          r.status === 200,
          [`PATCH /models/${model_name}/instances/latest offline task cls response instance.name`]: (r) =>
          r.json().instance.name === "latest",
          [`PATCH /models/${model_name}/instances/latest offline task cls response instance.model_definition_name`]: (r) =>
          r.json().instance.model_definition_name === model_name,
          [`PATCH /models/${model_name}/instances/latest offline task cls response instance.created_at`]: (r) =>
          r.json().instance.created_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest offline task cls response instance.updated_at`]: (r) =>
          r.json().instance.updated_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest offline task cls response model instance instance.status`]: (r) =>
          r.json().instance.status === "STATUS_OFFLINE",
          [`PATCH /models/${model_name}/instances/latest offline task cls response model instance instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`PATCH /models/${model_name}/instances/latest offline task cls response model instance instance.model_definition_source`]: (r) =>
          r.json().instance.model_definition_source === "SOURCE_LOCAL",    
          [`PATCH /models/${model_name}/instances/latest offline task cls response model instance instance.model_definition_id`]: (r) =>
          r.json().instance.model_definition_id !== undefined,      
      });
      sleep(6) // Triton unloading models takes time

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
      let model_description = randomString(20)
      fd_cls.append("name", model_name);
      fd_cls.append("description", model_description);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
          headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /models/upload (multipart) task cls response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/instances/latest`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/instances/latest online task cls response status`]: (r) =>
          r.status === 200,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.name`]: (r) =>
          r.json().instance.name === "latest",
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.model_definition_name`]: (r) =>
          r.json().instance.model_definition_name === model_name,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.created_at`]: (r) =>
          r.json().instance.created_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.updated_at`]: (r) =>
          r.json().instance.updated_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.status`]: (r) =>
          r.json().instance.status === "STATUS_ONLINE",
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.model_definition_source`]: (r) =>
          r.json().instance.model_definition_source === "SOURCE_LOCAL",    
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.model_definition_id`]: (r) =>
          r.json().instance.model_definition_id !== undefined,                
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs url cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs url cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs url cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/instances/latest/outputs url cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs base64 cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs base64 cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs base64 cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/instances/latest/outputs base64 cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      // Predict with multiple-part
      const fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/upload/outputs`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /models/${model_name}/instances/latest/upload/outputs form-data cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/upload/outputs form-data cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/upload/outputs form-data cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/instances/latest/upload/outputs form-data cls response contents.score`]: (r) =>
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
      let model_description = randomString(20)
      let fd_det = new FormData();
      let model_name = randomString(10)
      fd_det.append("name", model_name);
      fd_det.append("description", model_description);
      fd_det.append("content", http.file(det_model, "dummy-det-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_det.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd_det.boundary}`),
      }), {
          "POST /models/upload (multipart) task cls response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_DETECTION", 
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/instances/latest`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/instances/latest online task cls response status`]: (r) =>
          r.status === 200,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.name`]: (r) =>
          r.json().instance.name === "latest",
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.model_definition_name`]: (r) =>
          r.json().instance.model_definition_name === model_name,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.created_at`]: (r) =>
          r.json().instance.created_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task cls response instance.updated_at`]: (r) =>
          r.json().instance.updated_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.status`]: (r) =>
          r.json().instance.status === "STATUS_ONLINE",
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.task`]: (r) =>
          r.json().instance.task === "TASK_DETECTION",
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.model_definition_source`]: (r) =>
          r.json().instance.model_definition_source === "SOURCE_LOCAL",    
          [`PATCH /models/${model_name}/instances/latest online task cls response model instance instance.model_definition_id`]: (r) =>
          r.json().instance.model_definition_id !== undefined,                
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}],
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs url det response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs url det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs url det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /models/${model_name}/instances/latest/outputs url det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /models/${model_name}/instances/latest/outputs url det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs base64 det response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs base64 det output.detection_outputs.length`]: (r) =>
          r.json().output.detection_outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs base64 det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /models/${model_name}/instances/latest/outputs base64 det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /models/${model_name}/instances/latest/outputs base64 det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].bounding_box !== undefined,
      });

      // Predict with multiple-part
      const fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/upload/outputs`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /models/${model_name}/instances/latest/upload/outputs multiple-part det response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/upload/outputs multiple-part det output.detection_outputs[0].bounding_box_objects.length`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects.length === 1,
          [`POST /models/${model_name}/instances/latest/upload/outputs multiple-part det response output.detection_outputs[0].bounding_box_objects[0].category`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].category === "test",
          [`POST /models/${model_name}/instances/latest/upload/outputs multiple-part det response output.detection_outputs[0].bounding_box_objects[0].score`]: (r) =>
          r.json().output.detection_outputs[0].bounding_box_objects[0].score !== undefined,
          [`POST /models/${model_name}/instances/latest/upload/outputs multiple-part det response output.detection_outputs[0].bounding_box_objects[0].bounding_box`]: (r) =>
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
      let model_description = randomString(20)
      let fd = new FormData();
      let model_name = randomString(10)
      fd.append("name", model_name);
      fd.append("description", model_description);
      fd.append("content", http.file(unspecified_model, "dummy-unspecified-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
          "POST /models/upload (multipart) task unspecified response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task unspecified response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models/upload (multipart) task unspecified response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models/upload (multipart) task unspecified response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task unspecified response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task unspecified response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task unspecified response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task unspecified response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task unspecified response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task unspecified response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task unspecified response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task unspecified response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task unspecified response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task unspecified response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task unspecified response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task unspecified response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_UNSPECIFIED", 
          "POST /models/upload (multipart) task unspecified response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task unspecified response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task unspecified response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task unspecified response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/instances/latest`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/instances/latest online task unspecified response status`]: (r) =>
          r.status === 200,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.name`]: (r) =>
          r.json().instance.name === "latest",
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.model_definition_name`]: (r) =>
          r.json().instance.model_definition_name === model_name,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.created_at`]: (r) =>
          r.json().instance.created_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.updated_at`]: (r) =>
          r.json().instance.updated_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.status`]: (r) =>
          r.json().instance.status === "STATUS_ONLINE",
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.task`]: (r) =>
          r.json().instance.task === "TASK_UNSPECIFIED",
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.model_definition_source`]: (r) =>
          r.json().instance.model_definition_source === "SOURCE_LOCAL",    
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.model_definition_id`]: (r) =>
          r.json().instance.model_definition_id !== undefined,                
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs url undefined response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs url undefined outputs`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs url undefined parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /models/${model_name}/instances/latest/outputs url undefined raw_output_contents`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs url undefined raw_output_contents content`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict with base64
      payload = JSON.stringify({
        "inputs": [{"image_base64": base64_image,}]
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs base64 undefined response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs base64 undefined output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs base64 undefined output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /models/${model_name}/instances/latest/outputs base64 undefined output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs base64 undefined output.raw_output_contents[0]`]: (r) =>
          r.json().output.raw_output_contents[0] !== undefined,
      });

      // Predict with multiple-part
      fd = new FormData();
      fd.append("file", http.file(dog_img));
      check(http.post(`${apiHost}/models/${model_name}/instances/latest/upload/outputs`, fd.body(), {
        headers: genHeader(`multipart/form-data; boundary=${fd.boundary}`),
      }), {
        [`POST /models/${model_name}/instances/latest/outputs multipart undefined response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/latest/outputs multipart undefined output.outputs.length`]: (r) =>
          r.json().output.outputs.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs multipart undefined output.parameters`]: (r) =>
          r.json().output.parameters !== undefined,
          [`POST /models/${model_name}/instances/latest/outputs multipart undefined output.raw_output_contents.length`]: (r) =>
          r.json().output.raw_output_contents.length === 1,
          [`POST /models/${model_name}/instances/latest/outputs multipart undefined output.raw_output_contents[0]`]: (r) =>
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
      let model_description = randomString(20)
      fd_cls.append("name", model_name);
      fd_cls.append("description", model_description);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
          headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /models/upload (multipart) task cls response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });

      check(http.get(`${apiHost}/models/${model_name}`, {
        headers: genHeader(`application/json`),
      }), {
          [`GET /models/${model_name} response status`]: (r) =>
          r.status === 200, 
          [`GET /models/${model_name} response model.name`]: (r) =>
          r.json().model.name === model_name,
          [`GET /models/${model_name} response model.full_name`]: (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          [`GET /models/${model_name} response model.id`]: (r) =>
          r.json().model.id !== undefined,
          [`GET /models/${model_name} response model.description`]: (r) =>
          r.json().model.description === model_description,
          [`GET /models/${model_name} response model.source`]: (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          [`GET /models/${model_name} response model.visibility`]: (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          [`GET /models/${model_name} response model.owner.id`]: (r) =>
          r.json().model.owner.id !== undefined,
          [`GET /models/${model_name} response model.owner.username`]: (r) =>
          r.json().model.owner.username === "local-user",
          [`GET /models/${model_name} response model.owner.type`]: (r) =>
          r.json().model.owner.type === "user",
          [`GET /models/${model_name} response model.created_at`]: (r) =>
          r.json().model.created_at !== undefined,
          [`GET /models/${model_name} response model.updated_at`]: (r) =>
          r.json().model.updated_at !== undefined,
          [`GET /models/${model_name} response model.instances.length`]: (r) =>
          r.json().model.instances.length === 1,
          [`GET /models/${model_name} response model.instances[0].name`]: (r) =>
          r.json().model.instances[0].name === "latest",
          [`GET /models/${model_name} response model.instances[0].model_definition_name`]: (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          [`GET /models/${model_name} response model.instances[0].status`]: (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          [`GET /models/${model_name} response model.instances[0].task`]: (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          [`GET /models/${model_name} response model.instances[0].model_definition_id`]: (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          [`GET /models/${model_name} response model.instances[0].model_definition_source`]: (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          [`GET /models/${model_name} response model.instances[0].created_at`]: (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          [`GET /models/${model_name} response model.instances[0].updated_at`]: (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
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

 // Model Backend API: Get model list
  {
    group("Model Backend API: Get model list", function () {
      let fd_cls = new FormData();
      let model_name = randomString(10)
      let model_description = randomString(20)
      fd_cls.append("name", model_name);
      fd_cls.append("description", model_description);
      fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
      check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
          headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
      }), {
          "POST /models/upload (multipart) task cls response status": (r) =>
          r.status === 200, 
          "POST /models/upload (multipart) task cls response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models/upload (multipart) task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models/upload (multipart) task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models/upload (multipart) task cls response model.description": (r) =>
          r.json().model.description === model_description,
          "POST /models/upload (multipart) task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_LOCAL",
          "POST /models/upload (multipart) task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PRIVATE",
          "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models/upload (multipart) task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models/upload (multipart) task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "latest",
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models/upload (multipart) task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models/upload (multipart) task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models/upload (multipart) task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models/upload (multipart) task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models/upload (multipart) task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,              
      });

      check(http.get(`${apiHost}/models`, {
        headers: genHeader(`application/json`),
      }), {
          "GET /models response status": (r) =>
          r.status === 200, 
          "GET /models response model.name": (r) =>
          r.json().models[0].name === model_name,
          "GET /models response model.full_name": (r) =>
          r.json().models[0].full_name === `local-user/${model_name}`,
          "GET /models response model.id": (r) =>
          r.json().models[0].id !== undefined,
          "GET /models response model.description": (r) =>
          r.json().models[0].description === model_description,
          "GET /models response model.source": (r) =>
          r.json().models[0].source === "SOURCE_LOCAL",
          "GET /models response model.visibility": (r) =>
          r.json().models[0].visibility === "VISIBILITY_PRIVATE",
          "GET /models response model.owner.id": (r) =>
          r.json().models[0].owner.id !== undefined,
          "GET /models response model.owner.username": (r) =>
          r.json().models[0].owner.username === "local-user",
          "GET /models response model.owner.type": (r) =>
          r.json().models[0].owner.type === "user",
          "GET /models response model.created_at": (r) =>
          r.json().models[0].created_at !== undefined,
          "GET /models response model.updated_at": (r) =>
          r.json().models[0].updated_at !== undefined,
          "GET /models response model.instances.length": (r) =>
          r.json().models[0].instances.length === 1,
          "GET /models response model.instances[0].name": (r) =>
          r.json().models[0].instances[0].name === "latest",
          "GET /models response model.instances[0].model_definition_name": (r) =>
          r.json().models[0].instances[0].model_definition_name === r.json().models[0].name,
          "GET /models response model.instances[0].status": (r) =>
          r.json().models[0].instances[0].status === "STATUS_OFFLINE",          
          "GET /models response model.instances[0].task": (r) =>
          r.json().models[0].instances[0].task === "TASK_CLASSIFICATION", 
          "GET /models response model.instances[0].model_definition_id": (r) =>
          r.json().models[0].instances[0].model_definition_id === r.json().models[0].id,     
          "GET /models response model.instances[0].model_definition_source": (r) =>
          r.json().models[0].instances[0].model_definition_source === r.json().models[0].source,    
          "GET /models response model.instances[0].created_at": (r) =>
          r.json().models[0].instances[0].created_at !== undefined,  
          "GET /models response model.instances[0].updated_at": (r) =>
          r.json().models[0].instances[0].updated_at !== undefined,              
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

  // Model Backend API: upload model by GitHub
  {
    group("Model Backend API: Upload a model by GitHub", function () {
      let model_name = randomString(10)
      check(http.request("POST", `${apiHost}/models`, JSON.stringify({
        "name": model_name,
        "github": {
          "repo": "https://github.com/Phelan164/test-repo.git",
          "tag": "v1.0",
        }
      }), {
        headers: genHeader("application/json"),
      }), {
          "POST /models github task cls response status": (r) =>
          r.status === 200, 
          "POST /models github task cls response model.name": (r) =>
          r.json().model.name === model_name,
          "POST /models github task cls response model.full_name": (r) =>
          r.json().model.full_name === `local-user/${model_name}`,
          "POST /models github task cls response model.id": (r) =>
          r.json().model.id !== undefined,
          "POST /models github task cls response model.source": (r) =>
          r.json().model.source === "SOURCE_GITHUB",
          "POST /models github task cls response model.configuration.repo": (r) =>
          r.json().model.configuration.repo === "https://github.com/Phelan164/test-repo.git",
          "POST /models github task cls response model.visibility": (r) =>
          r.json().model.visibility === "VISIBILITY_PUBLIC",
          "POST /models github task cls response model.owner.id": (r) =>
          r.json().model.owner.id !== undefined,
          "POST /models github task cls response model.owner.username": (r) =>
          r.json().model.owner.username === "local-user",
          "POST /models github task cls response model.owner.type": (r) =>
          r.json().model.owner.type === "user",
          "POST /models github task cls response model.created_at": (r) =>
          r.json().model.created_at !== undefined,
          "POST /models github task cls response model.updated_at": (r) =>
          r.json().model.updated_at !== undefined,
          "POST /models github task cls response model.instances.length": (r) =>
          r.json().model.instances.length === 1,
          "POST /models github task cls response model.instances[0].name": (r) =>
          r.json().model.instances[0].name === "v1.0",
          "POST /models github task cls response model.instances[0].model_definition_name": (r) =>
          r.json().model.instances[0].model_definition_name === r.json().model.name,
          "POST /models github task cls response model.instances[0].status": (r) =>
          r.json().model.instances[0].status === "STATUS_OFFLINE",          
          "POST /models github task cls response model.instances[0].task": (r) =>
          r.json().model.instances[0].task === "TASK_CLASSIFICATION", 
          "POST /models github task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id === r.json().model.id,     
          "POST /models github task cls response model.instances[0].model_definition_source": (r) =>
          r.json().model.instances[0].model_definition_source === r.json().model.source,    
          "POST /models github task cls response model.instances[0].created_at": (r) =>
          r.json().model.instances[0].created_at !== undefined,  
          "POST /models github task cls response model.instances[0].updated_at": (r) =>
          r.json().model.instances[0].updated_at !== undefined,    
          "POST /models github task cls response model.instances[0].configuration.repo": (r) =>
          r.json().model.instances[0].configuration.repo === r.json().model.configuration.repo,   
          "POST /models github task cls response model.instances[0].configuration.tag": (r) =>
          r.json().model.instances[0].configuration.tag === "v1.0",     
          "POST /models github task cls response model.instances[0].configuration.html_url": (r) =>
          r.json().model.instances[0].configuration.html_url === "",     
          "POST /models github task cls response model.instances[0].model_definition_id": (r) =>
          r.json().model.instances[0].model_definition_id !== undefined,  
      });

      let payload = JSON.stringify({
        "status": "STATUS_ONLINE",
      });
      check(http.patch(`${apiHost}/models/${model_name}/instances/v1.0`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`PATCH /models/${model_name}/instances/latest online task unspecified response status`]: (r) =>
          r.status === 200,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.name`]: (r) =>
          r.json().instance.name === "v1.0",
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.model_definition_name`]: (r) =>
          r.json().instance.model_definition_name === model_name,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.created_at`]: (r) =>
          r.json().instance.created_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response instance.updated_at`]: (r) =>
          r.json().instance.updated_at !== undefined,
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.status`]: (r) =>
          r.json().instance.status === "STATUS_ONLINE",
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.task`]: (r) =>
          r.json().instance.task === "TASK_CLASSIFICATION",
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.model_definition_source`]: (r) =>
          r.json().instance.model_definition_source === "SOURCE_GITHUB",    
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance.model_definition_id`]: (r) =>
          r.json().instance.model_definition_id !== undefined,  
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance..configuration.repo`]: (r) =>
          r.json().instance.configuration.repo === "https://github.com/Phelan164/test-repo.git",         
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance..configuration.tag`]: (r) =>
          r.json().instance.configuration.tag === "v1.0",         
          [`PATCH /models/${model_name}/instances/latest online task unspecified response model instance instance..configuration.html_url`]: (r) =>
          r.json().instance.configuration.html_url === "",                
      });
      sleep(5) // Triton loading models takes time

      // Predict with url
      payload = JSON.stringify({
        "inputs": [{"image_url": "https://artifacts.instill.tech/dog.jpg"}]
      });
      check(http.post(`${apiHost}/models/${model_name}/instances/v1.0/outputs`, payload, {
        headers: genHeader(`application/json`),
      }), {
        [`POST /models/${model_name}/instances/v1.0/outputs url cls response status`]: (r) =>
          r.status === 200,
          [`POST /models/${model_name}/instances/v1.0/outputs url cls contents`]: (r) =>
          r.json().output.classification_outputs.length === 1,
          [`POST /models/${model_name}/instances/v1.0/outputs url cls contents.category`]: (r) =>
          r.json().output.classification_outputs[0].category === "match",
          [`POST /models/${model_name}/instances/v1.0/outputs url cls response contents.score`]: (r) =>
          r.json().output.classification_outputs[0].score === 1,
      });

      check(http.request("POST", `${apiHost}/models`, JSON.stringify({
        "name": model_name,
        "github": {
          "repo_url": "https://github.com/Phelan164/test-repo.git",
          "tag": "non-existed"
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /models by github invalid url status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/models`, JSON.stringify({
        "name": model_name,
        "github": {
          "repo": "https://github.com/Phelan164/non-existed-repo.git",
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /models by github invalid url status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/models`, JSON.stringify({
        "github": {
          "repo": "https://github.com/Phelan164/test-repo.git"
        }
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /models by github missing name status": (r) =>
          r.status === 400,
      });

      check(http.request("POST", `${apiHost}/models`, JSON.stringify({
        "name": model_name,
      }), {
        headers: genHeader("application/json"),
      }), {
        "POST /models by github missing github_url status": (r) =>
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
