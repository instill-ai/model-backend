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
  genHeader,
  isValidOwner,
} from "./helpers.js";

import * as constant from "./const.js"

const privateClient = new grpc.Client();
privateClient.load(['proto/model/model/v1alpha'], 'model_definition.proto');
privateClient.load(['proto/model/model/v1alpha'], 'model.proto');
privateClient.load(['proto/model/model/v1alpha'], 'model_private_service.proto');

const publicClient = new grpc.Client();
publicClient.load(['proto/model/model/v1alpha'], 'model_definition.proto');
publicClient.load(['proto/model/model/v1alpha'], 'model.proto');
publicClient.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

const model_def_name = "model-definitions/local"


export function ListModels() {
  // ListModelsAdmin check
  group("Model API: ListModels by admin", () => {
    privateClient.connect(constant.gRPCPrivateHost, {
      plaintext: true
    });
    publicClient.connect(constant.gRPCPublicHost, {
      plaintext: true
    });

    let fd_cls = new FormData();
    let model_id = randomString(10)
    let model_description = randomString(20)
    fd_cls.append("id", model_id);
    fd_cls.append("description", model_description);
    fd_cls.append("model_definition", model_def_name);
    fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
    let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
      headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
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
      let res = publicClient.invoke('model.model.v1alpha.ModelPublicService/GetModelOperation', {
        name: createClsModelRes.json().operation.name
      }, {})
      if (res.message.operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }
    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/ListModelsAdmin', {}, {}), {
      "ListModelsAdmin response status": (r) => r.status === grpc.StatusOK,
      "ListModelsAdmin response total_size": (r) => r.message.totalSize >= 1,
      "ListModelsAdmin response next_page_token": (r) => r.message.nextPageToken !== undefined,
      "ListModelsAdmin response models.length": (r) => r.message.models.length >= 1,
      "ListModelsAdmin response models[0].name": (r) => r.message.models[0].name === `models/${model_id}`,
      "ListModelsAdmin response models[0].uid": (r) => r.message.models[0].uid !== undefined,
      "ListModelsAdmin response models[0].id": (r) => r.message.models[0].id === model_id,
      "ListModelsAdmin response models[0].description": (r) => r.message.models[0].description !== undefined,
      "ListModelsAdmin response models[0].model_definition": (r) => r.message.models[0].modelDefinition === model_def_name,
      "ListModelsAdmin response models[0].configuration": (r) => r.message.models[0].configuration !== undefined,
      "ListModelsAdmin response models[0].visibility": (r) => r.message.models[0].visibility === "VISIBILITY_PRIVATE",
      "ListModelsAdmin response models[0].owner": (r) => isValidOwner(r.message.models[0].user),
      "ListModelsAdmin response models[0].create_time": (r) => r.message.models[0].createTime !== undefined,
      "ListModelsAdmin response models[0].update_time": (r) => r.message.models[0].updateTime !== undefined,
    });

    check(publicClient.invoke('model.model.v1alpha.ModelPublicService/DeleteModel', {
      name: "models/" + model_id
    }), {
      'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
    });

    privateClient.close();
    publicClient.close();
  });
};

export function LookUpModel() {
  // LookUpModelAdmin check
  group("Model API: LookUpModel by admin", () => {
    privateClient.connect(constant.gRPCPrivateHost, {
      plaintext: true
    });

    publicClient.connect(constant.gRPCPublicHost, {
      plaintext: true
    });

    let fd_cls = new FormData();
    let model_id = randomString(10)
    let model_description = randomString(20)
    fd_cls.append("id", model_id);
    fd_cls.append("description", model_description);
    fd_cls.append("model_definition", model_def_name);
    fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
    let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
      headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
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
      let res = publicClient.invoke('model.model.v1alpha.ModelPublicService/GetModelOperation', {
        name: createClsModelRes.json().operation.name
      }, {})
      if (res.message.operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    let res = publicClient.invoke('model.model.v1alpha.ModelPublicService/GetModel', {
      name: "models/" + model_id
    }, {})

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/LookUpModelAdmin', {
      permalink: "models/" + res.message.model.uid
    }, {}), {
      "LookUpModelAdmin response status": (r) => r.status === grpc.StatusOK,
      "LookUpModelAdmin response model.name": (r) => r.message.model.name === `models/${model_id}`,
      "LookUpModelAdmin response model.uid": (r) => r.message.model.uid !== undefined,
      "LookUpModelAdmin response model.id": (r) => r.message.model.id === model_id,
      "LookUpModelAdmin response model.description": (r) => r.message.model.description === model_description,
      "LookUpModelAdmin response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
      "LookUpModelAdmin response model.configuration": (r) => r.message.model.configuration !== undefined,
      "LookUpModelAdmin response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
      "LookUpModelAdmin response model.owner": (r) => isValidOwner(r.message.model.user),
      "LookUpModelAdmin response model.create_time": (r) => r.message.model.createTime !== undefined,
      "LookUpModelAdmin response model.update_time": (r) => r.message.model.updateTime !== undefined,
    });

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/LookUpModelAdmin', {
      permalink: "models/" + randomString(10)
    }, {}), {
      'LookUpModelAdmin non-existed model status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
    });
    check(publicClient.invoke('model.model.v1alpha.ModelPublicService/DeleteModel', {
      name: "models/" + model_id
    }), {
      'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
    });

    publicClient.close();
    privateClient.close();
  });
};
