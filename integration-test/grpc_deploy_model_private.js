import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
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

const privateClient = new grpc.Client();
privateClient.load(['proto/model/model/v1alpha'], 'model_definition.proto');
privateClient.load(['proto/model/model/v1alpha'], 'model.proto');
privateClient.load(['proto/model/model/v1alpha'], 'model_private_service.proto');

const publicClient = new grpc.Client();
publicClient.load(['proto/model/model/v1alpha'], 'model_definition.proto');
publicClient.load(['proto/model/model/v1alpha'], 'model.proto');
publicClient.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

const model_def_name = "model-definitions/local"


export function CheckModel() {
  // CheckModelAdmin check
  group("Model API: CheckModel by admin", () => {
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

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/CheckModelAdmin', {
      model_permalink: "models/" + res.message.model.uid
    }, {}), {
      "CheckModelAdmin response status": (r) => r.status === grpc.StatusOK,
      "CheckModelAdmin response state": (r) => r.message.state === "STATE_OFFLINE",
    });

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/CheckModelAdmin', {
      model_permalink: "models/" + randomString(10)
    }, {}), {
      'CheckModelAdmin uuid length is invalid': (r) => r && r.status === grpc.StatusInvalidArgument,
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

export function DeployUndeployModel() {
  // Deploy Model check
  group("Model API: Deploy and Undeploy Model by admin", () => {
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

    let getModelRes = publicClient.invoke('model.model.v1alpha.ModelPublicService/GetModel', {
      name: "models/" + model_id
    }, {})

    let req = {
      model_permalink: "models/" + getModelRes.message.model.uid
    }

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/DeployModelAdmin', req, {}), {
      'DeployModel status': (r) => r && r.status === grpc.StatusOK,
      'DeployModel operation name': (r) => r && r.message.operation.name !== undefined,
      'DeployModel operation metadata': (r) => r && r.message.operation.metadata === null,
      'DeployModel operation done': (r) => r && r.message.operation.done === false,
    });

    // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
    currentTime = new Date().getTime();
    timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      var res = publicClient.invoke('model.model.v1alpha.ModelPublicService/WatchModel', {
        name: `models/${model_id}`
      }, {})
      if (res.message.state === "STATE_ONLINE") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/DeployModelAdmin', {
      model_permalink: `models/non-existed`
    }), {
      'DeployModel uuid length is invalid': (r) => r && r.status === grpc.StatusInvalidArgument,
    });
    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/DeployModelAdmin', {
      model_permalink: `models/${uuidv4()}`
    }), {
      'DeployModel non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
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
