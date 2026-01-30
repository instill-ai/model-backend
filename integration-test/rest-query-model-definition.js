import http from "k6/http";
import {
  check,
  group
} from "k6";

import {
  genHeader,
} from "./helpers.js";

import * as constant from "./const.js"

const model_def_name = "model-definitions/local"

export function ListModelDefinitions(header) {
  // Model Backend API: get model definition list
  {
    group("Model Backend API: get model definition list", function () {
      check(http.get(`${constant.apiPublicHost}/v1alpha/model-definitions?view=VIEW_BASIC`, header), {
        [`GET /v1alpha/model-definitions} response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/model-definitions response next_page_token`]: (r) =>
          r.json().next_page_token !== undefined,
        [`GET /v1alpha/model-definitions response total_size`]: (r) =>
          r.json().total_size == 3,
        [`GET /v1alpha/model-definitions response modelDefinitions.length`]: (r) =>
          r.json().modelDefinitions.length === 3,
        [`GET /v1alpha/model-definitions response modelDefinitions[0].name`]: (r) =>
          r.json().modelDefinitions[0].name === "model-definitions/container",
        [`GET /v1alpha/model-definitions response modelDefinitions[0].uid`]: (r) =>
          r.json().modelDefinitions[0].uid !== undefined,
        [`GET /v1alpha/model-definitions response modelDefinitions[0].id`]: (r) =>
          r.json().modelDefinitions[0].id === "container",
        [`GET /v1alpha/model-definitions response modelDefinitions[0].title`]: (r) =>
          r.json().modelDefinitions[0].title === "Container",
        [`GET /v1alpha/model-definitions response modelDefinitions[0].documentation_url`]: (r) =>
          r.json().modelDefinitions[0].documentation_url === "https://www.instill-ai.com/docs/import-models/local",
        [`GET /v1alpha/model-definitions response modelDefinitions[0].icon`]: (r) =>
          r.json().modelDefinitions[0].icon === "local.svg",
        [`GET /v1alpha/model-definitions response modelDefinitions[0].model_spec`]: (r) =>
          r.json().modelDefinitions[0].model_spec === null,
      });
    });

    check(http.get(`${constant.apiPublicHost}/v1alpha/model-definitions?view=VIEW_FULL`, header), {
      [`GET /v1alpha/model-definitions}?view=VIEW_FULL response status`]: (r) =>
        r.status === 200,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response next_page_token`]: (r) =>
        r.json().next_page_token !== undefined,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response total_size`]: (r) =>
        r.json().total_size == 3,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions.length`]: (r) =>
        r.json().modelDefinitions.length === 3,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].name`]: (r) =>
        r.json().modelDefinitions[0].name === "model-definitions/container",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].uid`]: (r) =>
        r.json().modelDefinitions[0].uid !== undefined,
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].id`]: (r) =>
        r.json().modelDefinitions[0].id === "container",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].title`]: (r) =>
        r.json().modelDefinitions[0].title === "Container",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].documentation_url`]: (r) =>
        r.json().modelDefinitions[0].documentation_url === "https://www.instill-ai.com/docs/import-models/local",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].icon`]: (r) =>
        r.json().modelDefinitions[0].icon === "local.svg",
      [`GET /v1alpha/model-definitions?view=VIEW_FULL response modelDefinitions[0].model_spec`]: (r) =>
        r.json().modelDefinitions[0].model_spec !== null,
    });
  }
}

export function GetModelDefinition(header) {
  // Model Backend API: get model definition
  {
    group("Model Backend API: get model definition", function () {
      check(http.get(`${constant.apiPublicHost}/v1alpha/${model_def_name}`, header), {
        [`GET /v1alpha/model-definitions/${model_def_name} response status`]: (r) =>
          r.status === 200,
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.name`]: (r) =>
          r.json().modelDefinition.name === model_def_name,
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.id`]: (r) =>
          r.json().modelDefinition.id === "local",
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.uid`]: (r) =>
          r.json().modelDefinition.uid !== undefined,
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.title`]: (r) =>
          r.json().modelDefinition.title === "Local",
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.documentation_url`]: (r) =>
          r.json().modelDefinition.documentation_url !== undefined,
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.model_spec`]: (r) =>
          r.json().modelDefinition.model_spec !== undefined,
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.createTime`]: (r) =>
          r.json().modelDefinition.createTime !== undefined,
        [`GET /v1alpha/model-definitions/${model_def_name} response modelDefinition.updateTime`]: (r) =>
          r.json().modelDefinition.updateTime !== undefined,
      });
    });
  }
}
