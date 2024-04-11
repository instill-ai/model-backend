import grpc from 'k6/net/grpc';
import {
  check,
  group,
  sleep
} from 'k6';
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

const publicClient = new grpc.Client();
publicClient.load(['proto/model/model/v1alpha'], 'model_definition.proto');
publicClient.load(['proto/model/model/v1alpha'], 'model.proto');
publicClient.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

const privateClient = new grpc.Client();
privateClient.load(['proto/model/model/v1alpha'], 'model_definition.proto');
privateClient.load(['proto/model/model/v1alpha'], 'model.proto');
privateClient.load(['proto/model/model/v1alpha'], 'model_private_service.proto');


import * as constant from "./const.js"

export function CreateUserModel(header) {
  // CreateModel check
  group("Model API: CreateUserModel", () => {
    publicClient.connect(constant.gRPCPublicHost, {
      plaintext: true
    });
    privateClient.connect(constant.gRPCPrivateHost, {
      plaintext: true
    });
    let createRes = publicClient.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        id: constant.cls_model,
        model_definition: constant.model_def_name,
        visibility: "VISIBILITY_PUBLIC",
        region: "REGION_GCP_EUROPE_WEST_4",
        hardware: "CPU",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
      parent: constant.namespace,
    }, header)
    check(createRes, {
      'CreateUserModel status': (r) => r && r.status === grpc.StatusOK,
      'CreateUserModel model': (r) => r && r.message.model !== undefined,
    });

    check(privateClient.invoke('model.model.v1alpha.ModelPrivateService/DeployModelAdmin', {
      name: `${constant.namespace}/models/${constant.cls_model}`,
      version: "test"
    }, header), {
      'DeployUserModel status': (r) => r && r.status === grpc.StatusOK,
    });

    // Check the model state being updated in 360 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 360000;
    while (timeoutTime > currentTime) {
      var res = publicClient.invoke('model.model.v1alpha.ModelPublicService/WatchUserModel', {
        name: `${constant.namespace}/models/${constant.cls_model}`,
        version: "test"
      }, header)
      console.log(res.message.state)
      console.log(res.message.message)
      if (res.message.state === "STATE_ACTIVE") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }
    check(publicClient.invoke('model.model.v1alpha.ModelPublicService/TriggerUserModel', {
      name: `${constant.namespace}/models/${constant.cls_model}`,
      version: "test",
      task_inputs: [{
        classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
      }]
    }, header), {
      'TriggerUserModel status': (r) => r && r.status === grpc.StatusOK,
      'TriggerUserModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
      'TriggerUserModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
      'TriggerUserModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
    });

    // check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
    //   model: {
    //     id: randomString(10),
    //     model_definition: randomString(10),
    //     configuration: {
    //       repository: "admin/model-dummy-cls",
    //       tag: "v1.0-cpu"
    //     }
    //   },
    //   parent: constant.namespace,
    // }, header), {
    //   'status': (r) => r && r.status == grpc.StatusInvalidArgument,
    // });

    // check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
    //   model: {
    //     model_definition: model_def_name,
    //     configuration: {
    //       repository: "admin/model-dummy-cls",
    //       tag: "v1.0-cpu"
    //     }
    //   },
    //   parent: constant.namespace,
    // }, header), {
    //   'missing name status': (r) => r && r.status == grpc.StatusInvalidArgument,
    // });

    // check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
    //   model: {
    //     model_definition: model_def_name,
    //     configuration: {
    //       repository: "admin/model-dummy-cls",
    //       tag: "v1.0-cpu"
    //     }
    //   }
    // }, header), {
    //   'missing namespace': (r) => r && r.status == grpc.StatusInvalidArgument,
    // });

    // check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
    //   model: {
    //     id: randomString(10),
    //     model_definition: model_def_name,
    //   },
    //   parent: constant.namespace,
    // }, header), {
    //   'missing github url status': (r) => r && r.status == grpc.StatusInvalidArgument,
    // });

    check(publicClient.invoke('model.model.v1alpha.ModelPublicService/DeleteUserModel', {
      name: `${constant.namespace}/models/${constant.cls_model}`
    }, header), {
      'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
    });

    publicClient.close();
    privateClient.close();
  });
};
