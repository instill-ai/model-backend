import grpc from 'k6/net/grpc';
import {
    check,
    group
} from 'k6';

import * as createModel from "./grpc_create_model.js"
import * as updateModel from "./grpc_update_model.js"
import * as queryModel from "./grpc_query_model.js"
import * as queryModelPrivate from "./grpc_query_model_private.js"
import * as deployModel from "./grpc_deploy_model.js"
import * as inferModel from "./grpc_infer_model.js"
import * as publishModel from "./grpc_publish_model.js"
import * as queryModelInstance from "./grpc_query_model_instance.js"
import * as queryModelDefinition from "./grpc_query_model_definition.js"
import * as modelOperation from "./grpc_model_operation.js"

import * as constant from "./const.js"

export const options = {
    setupTimeout: '300s',
    insecureSkipTLSVerify: true,
    thresholds: {
        checks: ["rate == 1.0"],
    },
};

const client = new grpc.Client();
client.load(['proto/vdp/model/v1alpha'], 'model_definition.proto');
client.load(['proto/vdp/model/v1alpha'], 'model.proto');
client.load(['proto/vdp/model/v1alpha'], 'model_private_service.proto');
client.load(['proto/vdp/model/v1alpha'], 'model_public_service.proto');
client.load(['proto/vdp/model/v1alpha'], 'healthcheck.proto');

export function setup() { }

export default () => {
    // Liveness check
    {
        group("Model API: Liveness", () => {
            client.connect(constant.gRPCPublicHost, {
                plaintext: true
            });
            const response = client.invoke('vdp.model.v1alpha.ModelPublicService/Liveness', {});
            console.log(response.message);
            check(response, {
                'Status is OK': (r) => r && r.status === grpc.StatusOK,
                'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
            });
            client.close()
        });
    }

    // Private API
    if (__ENV.MODE != "api-gateway" && __ENV.MODE != "localhost") {
        queryModelPrivate.GetModel()
        queryModelPrivate.ListModels()
        queryModelPrivate.LookUpModel()
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

    // Query Model Instance API
    queryModelInstance.GetModelInstance()
    queryModelInstance.ListModelInstances()
    queryModelInstance.LookupModelInstance()

    // Query Model Definition API
    queryModelDefinition.GetModelDefinition()
    queryModelDefinition.ListModelDefinitions()

    // Operation API
    modelOperation.ListModelOperations()
    modelOperation.CancelModelOperation()
};

export function teardown() {
    client.connect(constant.gRPCPublicHost, {
        plaintext: true
    });
    group("Model API: Delete all models created by this test", () => {
        for (const model of client.invoke('vdp.model.v1alpha.ModelPublicService/ListModels', {}, {}).message.models) {
            check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
                name: model.name
            }), {
                'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
            });
        }
    });
    client.close();
}
