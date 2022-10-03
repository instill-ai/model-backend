import http from "k6/http";
import { check, group } from "k6";

import {
  genHeader,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"

export function ListModelDefinition() {
  // Model Backend API: get model definition list
  {
    group("Model Backend API: get model definition list", function () {
      check(http.get(`${constant.apiHost}/v1alpha/model-definitions`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/model-definitions} response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/model-definitions response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/model-definitions response total_size`]: (r) =>
          r.json().total_size == 4,
        [`GET /v1alpha/model-definitions response model_definitions.length`]: (r) =>
          r.json().model_definitions.length === 4,
        [`GET /v1alpha/model-definitions response model_definitions[0].name`]: (r) =>
        r.json().model_definitions[2].name === "model-definitions/local",
        [`GET /v1alpha/model-definitions response model_definitions[0].uid`]: (r) =>
        r.json().model_definitions[2].uid !== undefined,
        [`GET /v1alpha/model-definitions response model_definitions[0].id`]: (r) =>
        r.json().model_definitions[2].id === "local",
        [`GET /v1alpha/model-definitions response model_definitions[0].title`]: (r) =>
        r.json().model_definitions[2].title === "Local",
        [`GET /v1alpha/model-definitions response model_definitions[0].documentation_url`]: (r) =>
        r.json().model_definitions[2].documentation_url === "https://www.instill.tech/docs/import-models/local",
        [`GET /v1alpha/model-definitions response model_definitions[0].icon`]: (r) =>
        r.json().model_definitions[2].icon === "local.svg",
        [`GET /v1alpha/model-definitions response model_definitions[0].model_spec`]: (r) =>
        r.json().model_definitions[2].model_spec === null,
        [`GET /v1alpha/model-definitions response model_definitions[0].model_instance_spec`]: (r) =>
        r.json().model_definitions[2].model_instance_spec === null,
      });
    });

    check(http.get(`${constant.apiHost}/v1alpha/model-definitions?view=VIEW_FULL`, {
      headers: genHeader(`application/json`),
    }), {
      [`GET /v1alpha/model-definitions}?view=VIEW_FULL response status`]: (r) =>
        r.status === 200,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response next_page_token`]: (r) =>
        r.json().next_page_token !== undefined,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response total_size`]: (r) =>
        r.json().total_size == 4,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions.length`]: (r) =>
        r.json().model_definitions.length === 4,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].name`]: (r) =>
      r.json().model_definitions[2].name === "model-definitions/local",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].uid`]: (r) =>
      r.json().model_definitions[2].uid !== undefined,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].id`]: (r) =>
      r.json().model_definitions[2].id === "local",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].title`]: (r) =>
      r.json().model_definitions[2].title === "Local",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].documentation_url`]: (r) =>
      r.json().model_definitions[2].documentation_url === "https://www.instill.tech/docs/import-models/local",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].icon`]: (r) =>
      r.json().model_definitions[2].icon === "local.svg",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response model_definitions[0].model_spec`]: (r) =>
      r.json().model_definitions[2].model_spec !== null,
      [`GET /v1alpha/model-definitions response model_definitions[0].model_instance_spec`]: (r) =>
      r.json().model_definitions[2].model_instance_spec !== null,
    });
  }
}

export function GetModelDefinition() {
  // Model Backend API: get model definition
  {
    group("Model Backend API: get model definition", function () {
      check(http.get(`${constant.apiHost}/v1alpha/${model_def_name}`, {
        headers: genHeader(`application/json`),
      }), {
        [`GET /v1alpha/model-definitions/${model_def_name} response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.name`]: (r) =>
          r.json().model_definition.name === model_def_name,
        [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.id`]: (r) =>
          r.json().model_definition.id === "local",
        [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.uid`]: (r) =>
          r.json().model_definition.uid !== undefined,
        [`GET /v1alpha/model-definitions/${model_def_name} response model_definition.title`]: (r) =>
          r.json().model_definition.title === "Local",
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
