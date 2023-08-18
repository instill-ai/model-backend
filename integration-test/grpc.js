import grpc from 'k6/net/grpc';
import {
  check,
  group
} from 'k6';

import * as createModel from "./grpc_create_model.js"
import * as updateModel from "./grpc_update_model.js"
import * as queryModel from "./grpc_query_model.js"
import * as queryModelPrivate from "./grpc_query_model_private.js"
import * as deployModelPrivate from "./grpc_deploy_model_private.js"
import * as deployModel from "./grpc_deploy_model.js"
import * as inferModel from "./grpc_infer_model.js"
import * as publishModel from "./grpc_publish_model.js"
import * as queryModelDefinition from "./grpc_query_model_definition.js"

import * as constant from "./const.js"

export const options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

const client = new grpc.Client();
client.load(['proto/common'], 'healthcheck.proto');
client.load(['proto/model/model/v1alpha'], 'model_definition.proto');
client.load(['proto/model/model/v1alpha'], 'model.proto');
client.load(['proto/model/model/v1alpha'], 'model_private_service.proto');
client.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

export function setup() { }

export default () => {
  // Liveness check
  {
    group("Model API: Liveness", () => {
      client.connect(constant.gRPCPublicHost, {
        plaintext: true
      });
      const response = client.invoke('model.model.v1alpha.ModelPublicService/Liveness', {});
      check(response, {
        'Status is OK': (r) => r && r.status === grpc.StatusOK,
        'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
      });
      client.close()
    });
  }

  // Private API
  if (!constant.apiGatewayMode) {
    queryModelPrivate.ListModels()
    queryModelPrivate.LookUpModel()
    deployModelPrivate.CheckModel()
    // private deploy will be triggered by public deploy
    // deployModelPrivate.DeployUndeployModel()
  }

  // Create model API
  createModel.CreateModel()

  // Update model API
  updateModel.UpdateModel()

  // Deploy Model API
  deployModel.DeployUndeployModel()

  // Query Model API
  queryModel.GetModel()
  queryModel.ListModels()
  queryModel.LookupModel()

  // Publish Model API
  publishModel.PublishUnPublishModel()

  // Infer Model API
  inferModel.InferModel()

  // Query Model Definition API
  queryModelDefinition.GetModelDefinition()
  queryModelDefinition.ListModelDefinitions()
};

export function teardown() {
  client.connect(constant.gRPCPublicHost, {
    plaintext: true
  });
  group("Model API: Delete all models created by this test", () => {
    for (const model of client.invoke('model.model.v1alpha.ModelPublicService/ListModels', {}, {}).message.models) {
      check(client.invoke('model.model.v1alpha.ModelPublicService/DeleteModel', {
        name: model.name
      }), {
        'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
      });
    }
  });
  client.close();
}
