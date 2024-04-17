import grpc from 'k6/net/grpc';
import {
  check,
  group
} from 'k6';
import * as constant from "./const.js"

const client = new grpc.Client();
client.load(['proto/model/model/v1alpha'], 'model_definition.proto');
client.load(['proto/model/model/v1alpha'], 'model.proto');
client.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

const model_def_name = "model-definitions/local"


export function TriggerUserModel(header) {
  // TriggerModel check
  group("Model API: TriggerUserModel", () => {
    client.connect(constant.gRPCPublicHost, {
      plaintext: true
    });

    res = client.invoke('model.model.v1alpha.ModelPublicService/TriggerUserModel', {
      name: `${constant.namespace}/models/${constant.cls_model}`,
      version: "test",
      task_inputs: [{
        classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
      }]
    }, header)
    check(res, {
      'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
      'TriggerModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
      'TriggerModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
      'TriggerModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/TriggerUserModel', {
      name: `${constant.namespace}/models/${constant.cls_model}`,
      version: "test",
      task_inputs: [{
        classification: { image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff" }
      }]
    }, header), {
      'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
      'TriggerModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
      'TriggerModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
      'TriggerModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
    });


    check(client.invoke('model.model.v1alpha.ModelPublicService/TriggerUserModel', {
      name: `${constant.namespace}/models/non-existed`,
      task_inputs: [{
        classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
      }]
    }, header), {
      'TriggerModel non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/TriggerUserModel', {
      name: `${constant.namespace}/models/${constant.cls_model}`,
      task_inputs: [{
        classification: { image_url: "https://artifacts.instill.tech/non-existed.jpg" }
      }]
    }, header), {
      'TriggerModel non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
    });

    client.close();
  });
};
