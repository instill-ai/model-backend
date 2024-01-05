import grpc from 'k6/net/grpc';
import {
  check,
  group,
  sleep
} from 'k6';
import http from "k6/http";
import {
  FormData
} from "https://jslib.k6.io/formdata/0.0.2/index.js";
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import {
  URL
} from "https://jslib.k6.io/url/1.0.0/index.js";

import {
  genHeader,
  isValidOwner
} from "./helpers.js";

import * as constant from "./const.js"

const client = new grpc.Client();
client.load(['proto/model/model/v1alpha'], 'model_definition.proto');
client.load(['proto/model/model/v1alpha'], 'model.proto');
client.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

const model_def_name = "model-definitions/local"

export function PublishUnPublishUserModel(header) {
  // PublishModel/UnpublishModel check
  group("Model API: PublishModel/UnpublishUserModel", () => {
    client.connect(constant.gRPCPublicHost, {
      plaintext: true
    });

    let fd_cls = new FormData();
    let model_id = randomString(10)
    let model_description = randomString(20)
    fd_cls.append("id", model_id);
    fd_cls.append("description", model_description);
    fd_cls.append("model_definition", model_def_name);
    fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
    let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/multipart`, fd_cls.body(), {
      headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`, header.metadata.Authorization),
    })
    check(createClsModelRes, {
      "POST /v1alpha/users/admin/models/multipart task cls response status": (r) =>
        r.status === 201,
      "POST /v1alpha/users/admin/models/multipart task cls response operation.name": (r) =>
        r.json().operation.name !== undefined,
    });

    // Check model creation finished
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      let res = client.invoke('model.model.v1alpha.ModelPublicService/GetModelOperation', {
        name: createClsModelRes.json().operation.name
      }, header)
      if (res.message.operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    check(client.invoke('model.model.v1alpha.ModelPublicService/PublishUserModel', {
      name: `${constant.namespace}/models/${model_id}`
    }, header), {
      "PublishModel response status": (r) => r.status === grpc.StatusOK,
      "PublishModel response model.name": (r) => r.message.model.name === `${constant.namespace}/models/${model_id}`,
      "PublishModel response model.uid": (r) => r.message.model.uid !== undefined,
      "PublishModel response model.id": (r) => r.message.model.id === model_id,
      "PublishModel response model.description": (r) => r.message.model.description === model_description,
      "PublishModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
      "PublishModel response model.configuration": (r) => r.message.model.configuration !== undefined,
      "PublishModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PUBLIC",
      "PublishModel response model.ownerName": (r) => isValidOwner(r.message.model.ownerName),
      "PublishModel response model.create_time": (r) => r.message.model.createTime !== undefined,
      "PublishModel response model.update_time": (r) => r.message.model.updateTime !== undefined,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/UnpublishUserModel', {
      name: `${constant.namespace}/models/${model_id}`
    }, header), {
      "UnpublishModel response status": (r) => r.status === grpc.StatusOK,
      "UnpublishModel response model.name": (r) => r.message.model.name === `${constant.namespace}/models/${model_id}`,
      "UnpublishModel response model.uid": (r) => r.message.model.uid !== undefined,
      "UnpublishModel response model.id": (r) => r.message.model.id === model_id,
      "UnpublishModel response model.description": (r) => r.message.model.description === model_description,
      "UnpublishModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
      "UnpublishModel response model.configuration": (r) => r.message.model.configuration !== undefined,
      "UnpublishModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
      "UnpublishModel response model.owner": (r) => isValidOwner(r.message.model.user),
      "UnpublishModel response model.create_time": (r) => r.message.model.createTime !== undefined,
      "UnpublishModel response model.update_time": (r) => r.message.model.updateTime !== undefined,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/PublishUserModel', {
      name: "users/admin/models/" + randomString(10)
    }, header), {
      "PublishModel response not found status": (r) => r.status === grpc.StatusNotFound,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/UnpublishUserModel', {
      name: "users/admin/models/" + randomString(10)
    }, header), {
      "UnpublishModel response not found status": (r) => r.status === grpc.StatusNotFound,
    });
    currentTime = new Date().getTime();
    timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      let res = client.invoke('model.model.v1alpha.ModelPublicService/WatchUserModel', {
        name: `${constant.namespace}/models/${model_id}`
      }, header)
      if (res.message.state !== "STATE_UNSPECIFIED") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }
    check(client.invoke('model.model.v1alpha.ModelPublicService/DeleteUserModel', {
      name: `${constant.namespace}/models/${model_id}`
    }, header), {
      'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
    });
    client.close();
  });
};
