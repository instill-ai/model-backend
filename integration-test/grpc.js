import grpc from 'k6/net/grpc';
import http from 'k6/http';
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
import * as triggerModel from "./grpc_infer_model.js"
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
client.load(['proto/model/model/v1alpha'], 'model_definition.proto');
client.load(['proto/model/model/v1alpha'], 'model.proto');
client.load(['proto/model/model/v1alpha'], 'model_private_service.proto');
client.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

export function setup() {
  var loginResp = http.request("POST", `${constant.mgmtPublicHost}/v1beta/auth/login`, JSON.stringify({
    "username": constant.defaultUserId,
    "password": constant.defaultPassword,
  }))

  check(loginResp, {
    [`POST ${constant.mgmtPublicHost}/v1beta/auth/login response status is 200`]: (
      r
    ) => r.status === 200,
  });

  var metadata = {
    "metadata": {
      "Authorization": `Bearer ${loginResp.json().access_token}`
    },
    "timeout": "600s",
  }

  return metadata
}

export default (header) => {
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
    queryModelPrivate.ListModels(header)
    queryModelPrivate.LookUpModel(header)
    deployModelPrivate.CheckModel(header)
    // private deploy will be triggered by public deploy
    // deployModelPrivate.DeployUndeployModel()
  }

  // Update model API
  updateModel.UpdateUserModel(header)

  // Create model API
  createModel.CreateUserModel(header)

  // Deploy Model API
  deployModel.DeployUndeployUserModel(header)

  // Query Model API
  queryModel.GetUserModel(header)
  queryModel.ListUserModels(header)
  queryModel.LookupModel(header)

  // Publish Model API
  publishModel.PublishUnPublishUserModel(header)

  // Trigger Model API
  triggerModel.TriggerUserModel(header)

  // Query Model Definition API
  queryModelDefinition.GetModelDefinition(header)
  queryModelDefinition.ListModelDefinitions(header)
};

export function teardown(header) {
  client.connect(constant.gRPCPublicHost, {
    plaintext: true
  });
  group("Model API: Delete all models created by this test", () => {
    for (const model of client.invoke('model.model.v1alpha.ModelPublicService/ListModels', {}, header).message.models) {
      check(client.invoke('model.model.v1alpha.ModelPublicService/DeleteUserModel', {
        name: model.name
      }, header), {
        'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
      });
    }
  });
  client.close();
}
