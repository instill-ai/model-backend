import http from "k6/http";
import { check, group } from "k6";

import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
  genHeader,
} from "./helpers.js";

const apiHost = "http://model-backend:8083";

const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const cls_no_readme_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-no-readme.zip`, "b");
const model_def_name = "model-definitions/local"

export function GetModelCard() {
  // Model Backend API: Get model card
  {
    group("Model Backend API: Get model card", function () {
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

      check(http.get(`${apiHost}/v1alpha/models/${model_id}/instances/latest/readme`), {
        [`GET /v1alpha/models/${model_id}/instances/latest/readme response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/models/${model_id}/instances/latest/readme response readme.name`]: (r) =>
          r.json().readme.name === `models/${model_id}/instances/latest/readme`,
        [`GET /v1alpha/models/${model_id}/instances/latest/readme response readme.size`]: (r) =>
          r.json().readme.size !== undefined,
        [`GET /v1alpha/models/${model_id}/instances/latest/readme response readme.type`]: (r) =>
          r.json().readme.type === "file",  
        [`GET /v1alpha/models/${model_id}/instances/latest/readme response readme.encoding`]: (r) =>
          r.json().readme.encoding === "base64",                    
        [`GET /v1alpha/models/${model_id}/instances/latest/readme response readme.content`]: (r) =>
          r.json().readme.content !== undefined,      
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
 // Model Backend API: Get model card without readme
 {
  group("Model Backend API: Get model card without readme", function () {
    let fd_cls = new FormData();
    let model_id = randomString(10)
    let model_description = randomString(20)
    fd_cls.append("id", model_id);
    fd_cls.append("description", model_description);
    fd_cls.append("model_definition", model_def_name);
    fd_cls.append("content", http.file(cls_no_readme_model, "dummy-cls-no-readme.zip"));
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

    check(http.get(`${apiHost}/v1alpha/models/${model_id}/instances/latest/readme`), {
      [`GET /v1alpha/models/${model_id}/instances/latest/readme response status`]: (r) =>
        r.status === 200,
      [`GET /v1alpha/models/${model_id}/instances/latest/readme no readme response readme.name`]: (r) =>
        r.json().readme.name === `models/${model_id}/instances/latest/readme`,
      [`GET /v1alpha/models/${model_id}/instances/latest/readme no readme response readme.size`]: (r) =>
        r.json().readme.size === 0,
      [`GET /v1alpha/models/${model_id}/instances/latest/readme no readme response readme.type`]: (r) =>
        r.json().readme.type === "file",  
      [`GET /v1alpha/models/${model_id}/instances/latest/readme no readme response readme.encoding`]: (r) =>
        r.json().readme.encoding === "base64",                    
      [`GET /v1alpha/models/${model_id}/instances/latest/readme no readme response readme.content`]: (r) =>
        r.json().readme.content === "",      
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
