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

const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-det-model.zip`, "b");
const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-unspecified-model.zip`, "b");
const model_def_name = "model-definitions/local"
const model_def_uid = "909c3278-f7d1-461c-9352-87741bef11d3"

export function PublishUnpublishModel() {
  // Model Backend API: PublishModel
  {
    group("Model Backend API: PublishModel", function () {
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
          r.status === 204
      });
    });
  }

}
