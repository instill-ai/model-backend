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
} from "./helpers.js";

import * as constant from "./const.js"

const client = new grpc.Client();
// Load protos from root 'proto' directory - let imports resolve naturally
client.load(['proto'], 'model/v1alpha/model_public_service.proto');

const model_def_name = "model-definitions/local"

export function DeployUndeployUserModel(header) {
  // Deploy Model check
  group("Model API: Deploy User Model", () => {
    client.connect(constant.gRPCPublicHost, {
      plaintext: true
    });

    let fd_cls = new FormData();
    let model_id = randomString(10)
    let model_description = randomString(20)
    fd_cls.append("id", model_id);
    fd_cls.append("description", model_description);
    fd_cls.append("modelDefinition", model_def_name);
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
      let res = client.invoke('model.v1alpha.ModelPublicService/GetModelOperation', {
        name: createClsModelRes.json().operation.name
      }, header)
      if (res.message.operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    let req = {
      name: `${constant.namespace}/models/${model_id}`
    }
    check(client.invoke('model.v1alpha.ModelPublicService/DeployUserModel', req, header), {
      'DeployModel status': (r) => r && r.status === grpc.StatusOK,
      'DeployModel model name': (r) => r && r.message.modelId === model_id
    });

    // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
    currentTime = new Date().getTime();
    timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      var res = client.invoke('model.v1alpha.ModelPublicService/WatchUserModel', {
        name: `${constant.namespace}/models/${model_id}`
      }, header)
      if (res.message.state === "STATE_ONLINE") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    check(client.invoke('model.v1alpha.ModelPublicService/DeployUserModel', {
      name: `${constant.namespace}/models/non-existed`
    }, header), {
      'DeployModel non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
    });

    check(client.invoke('model.v1alpha.ModelPublicService/DeleteUserModel', {
      name: `${constant.namespace}/models/${model_id}`
    }, header), {
      'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
    });
    client.close();
  });
};
