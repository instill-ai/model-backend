import grpc from 'k6/net/grpc';
import {
  check,
  group,
  sleep
} from 'k6';
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

const client = new grpc.Client();
client.load(['proto/model/model/v1alpha'], 'model_definition.proto');
client.load(['proto/model/model/v1alpha'], 'model.proto');
client.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

import * as constant from "./const.js"

const model_def_name = "model-definitions/github"

export function CreateModel() {
  // CreateModelBinaryFileUpload check
  group("Model API: CreateModelBinaryFileUpload", () => {
    client.connect(constant.gRPCPublicHost, {
      plaintext: true
    });
    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateModelBinaryFileUpload', {}), {
      'Missing stream body status': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    client.close();
  });


  // CreateModel check
  group("Model API: CreateModel with GitHub", () => {
    client.connect(constant.gRPCPublicHost, {
      plaintext: true
    });
    let model_id = randomString(10)
    let createOperationRes = client.invoke('model.model.v1alpha.ModelPublicService/CreateModel', {
      model: {
        id: model_id,
        model_definition: model_def_name,
        configuration: {
          repository: "instill-ai/model-dummy-cls",
          tag: "v1.0-cpu"
        }
      }
    })
    check(createOperationRes, {
      'CreateModel status': (r) => r && r.status === grpc.StatusOK,
      'CreateModel operation name': (r) => r && r.message.operation.name !== undefined,
    });

    // Check model creation finished
    let currentTime = new Date().getTime();
    let timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      let res = client.invoke('model.model.v1alpha.ModelPublicService/GetModelOperation', {
        name: createOperationRes.message.operation.name
      }, {})
      if (res.message.operation.done === true) {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    let req = {
      name: `models/${model_id}`
    }
    check(client.invoke('model.model.v1alpha.ModelPublicService/DeployModel', req, {}), {
      'DeployModel status': (r) => r && r.status === grpc.StatusOK,
      'DeployModel model name': (r) => r && r.message.modelId === model_id
    });

    // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
    currentTime = new Date().getTime();
    timeoutTime = new Date().getTime() + 120000;
    while (timeoutTime > currentTime) {
      var res = client.invoke('model.model.v1alpha.ModelPublicService/WatchModel', {
        name: `models/${model_id}`
      }, {})
      if (res.message.state === "STATE_ONLINE") {
        break
      }
      sleep(1)
      currentTime = new Date().getTime();
    }

    check(client.invoke('model.model.v1alpha.ModelPublicService/TriggerModel', {
      name: `models/${model_id}`,
      task_inputs: [{
        classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
      }]
    }, {}), {
      'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
      'TriggerModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
      'TriggerModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
      'TriggerModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateModel', {
      model: {
        id: randomString(10),
        model_definition: randomString(10),
        configuration: {
          repository: "instill-ai/model-dummy-cls",
          tag: "v1.0-cpu"
        }
      }
    }), {
      'status': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateModel', {
      model: {
        model_definition: model_def_name,
        configuration: {
          repository: "instill-ai/model-dummy-cls",
          tag: "v1.0-cpu"
        }
      }
    }), {
      'missing name status': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    let createModel = client.invoke('model.model.v1alpha.ModelPublicService/CreateModel', {
      model: {
        id: randomString(10),
        model_definition: model_def_name,
      }
    })
    check(createModel, {
      'missing github url status': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/DeleteModel', {
      name: "models/" + model_id
    }), {
      'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
    });

    client.close();
  });
};
