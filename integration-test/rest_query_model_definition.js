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

export function ListModelDefinition() {
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
          r.json().model_definitions.length === 2,
        [`GET /v1alpha/model-definitions response model_definitions[0].name`]: (r) =>
        r.json().model_definitions[0].name === "model-definitions/local",
        [`GET /v1alpha/model-definitions response model_definitions[0].uid`]: (r) =>
        r.json().model_definitions[0].uid !== undefined,
        [`GET /v1alpha/model-definitions response model_definitions[0].id`]: (r) =>
        r.json().model_definitions[0].id === "local",      
        [`GET /v1alpha/model-definitions response model_definitions[0].title`]: (r) =>
        r.json().model_definitions[0].title === "Local",                
        [`GET /v1alpha/model-definitions response model_definitions[0].documentation_url`]: (r) =>
        r.json().model_definitions[0].documentation_url === "https://docs.instill.tech/integrations/models/local",          
        [`GET /v1alpha/model-definitions response model_definitions[0].icon`]: (r) =>
        r.json().model_definitions[0].icon === "local.svg",
        [`GET /v1alpha/model-definitions response model_definitions[0].model_spec`]: (r) =>
        r.json().model_definitions[0].model_spec === null,        
        [`GET /v1alpha/model-definitions response model_definitions[0].model_instance_spec`]: (r) =>
        r.json().model_definitions[0].model_instance_spec === null,         
      });
    });

    check(http.get(`${apiHost}/v1alpha/model-definitions?view=VIEW_FULL`, {
      headers: genHeader(`application/json`),
    }), {
      [`GET /v1alpha/model-definitions}?view=VIEW_FULL response status`]: (r) =>
        r.status === 200,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response next_page_token`]: (r) =>
        r.json().next_page_token !== undefined,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response total_size`]: (r) =>
        r.json().total_size == 2,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions.length`]: (r) =>
        r.json().model_definitions.length === 2,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].name`]: (r) =>
      r.json().model_definitions[0].name === "model-definitions/local",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].uid`]: (r) =>
      r.json().model_definitions[0].uid !== undefined,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].id`]: (r) =>
      r.json().model_definitions[0].id === "local",      
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].title`]: (r) =>
      r.json().model_definitions[0].title === "Local",                
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].documentation_url`]: (r) =>
      r.json().model_definitions[0].documentation_url === "https://docs.instill.tech/integrations/models/local",          
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].icon`]: (r) =>
      r.json().model_definitions[0].icon === "local.svg",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].model_spec`]: (r) =>
      r.json().model_definitions[0].model_spec !== null,        
      [`GET /v1alpha/model-definitions response model_definitions[0].model_instance_spec`]: (r) =>
      r.json().model_definitions[0].model_instance_spec !== null,         
    });
  }
}

export function GetModelDefinition() {
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
}