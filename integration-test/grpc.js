import grpc from 'k6/net/grpc';
import {
  check,
  group
} from 'k6';

import * as createModel from "./grpc-create-model.js"
import * as updateModel from "./grpc-update-model.js"
import * as queryModel from "./grpc-query-model.js"
import * as queryModelPrivate from "./grpc-query-model-private.js"
import * as deployModelPrivate from "./grpc-deploy-model-private.js"
import * as deployModel from "./grpc-deploy-model.js"
import * as triggerModel from "./grpc-infer-model.js"
import * as publishModel from "./grpc-publish-model.js"
import * as queryModelDefinition from "./grpc-query-model-definition.js"

import { getBasicAuthHeader } from "./helpers.js"
import * as constant from "./const.js"

export const options = {
  setupTimeout: '300s',
  insecureSkipTLSVerify: true,
  thresholds: {
    checks: ["rate == 1.0"],
  },
};

const client = new grpc.Client();
// Load protos from root 'proto' directory - let imports resolve naturally
client.load(['proto'], 'model/v1alpha/model_public_service.proto');
client.load(['proto'], 'model/v1alpha/model_private_service.proto');

export function setup() {
  // CE uses Basic Auth (JWT auth is only available in EE)
  const authHeader = getBasicAuthHeader(constant.defaultUserId, constant.defaultPassword);

  var metadata = {
    "metadata": {
      "Authorization": authHeader
    },
    "timeout": "600s",
  }

  return metadata
}

export default (header) => {
  // Skip gRPC tests in API Gateway mode - gRPC routing not fully configured for model service
  if (constant.apiGatewayMode) {
    console.log("Skipping gRPC tests in API Gateway mode. Use rest.js for API Gateway testing.");
    return;
  }

  // Liveness check
  {
    group("Model API: Liveness", () => {
      client.connect(constant.gRPCPrivateHost, {
        plaintext: true
      });
      const response = client.invoke('model.v1alpha.ModelPublicService/Liveness', {});
      check(response, {
        'Status is OK': (r) => r && r.status === grpc.StatusOK,
        'Response status is SERVING_STATUS_SERVING': (r) => r && r.message && r.message.healthCheckResponse && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
      });
      client.close()
    });
  }

  // Private API
  // if (!constant.apiGatewayMode) {
  // queryModelPrivate.ListModels(header)
  // queryModelPrivate.LookUpModel(header)
  // deployModelPrivate.CheckModel(header)
  // private deploy will be triggered by public deploy
  // deployModelPrivate.DeployUndeployModel()
  // }

  // Update model API
  // updateModel.UpdateUserModel(header)

  // Create model API
  // createModel.CreateUserModel(header)

  // Deploy Model API
  // deployModel.DeployUndeployUserModel(header)

  // Query Model API
  // queryModel.GetUserModel(header)
  // queryModel.ListUserModels(header)
  // queryModel.LookupModel(header)

  // Publish Model API
  // publishModel.PublishUnPublishUserModel(header)

  // Trigger Model API
  // triggerModel.TriggerUserModel(header)

  // Query Model Definition API
  // queryModelDefinition.GetModelDefinition(header)
  // queryModelDefinition.ListModelDefinitions(header)
};

export function teardown(header) {
  // client.connect(constant.gRPCPublicHost, {
  //   plaintext: true
  // });
  // group("Model API: Delete all models created by this test", () => {
  //   for (const model of client.invoke('model.v1alpha.ModelPublicService/ListModels', {}, header).message.models) {
  //     check(client.invoke('model.v1alpha.ModelPublicService/DeleteUserModel', {
  //       name: model.name
  //     }, header), {
  //       'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
  //     });
  //   }
  // });
  // client.close();
}
