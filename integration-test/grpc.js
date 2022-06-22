import grpc from 'k6/net/grpc';
import { check, sleep, group } from 'k6';
import http from "k6/http";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import * as createModel from "./grpc_create_model.js"
import * as updateModel from "./grpc_update_model.js"
import * as queryModel from "./grpc_query_model.js"
import * as deployModel from "./grpc_deploy_model.js"
import * as inferModel from "./grpc_infer_model.js"
import * as publishModel from "./grpc_publish_model.js"
import * as queryModelInstance from "./grpc_query_model_instance.js"
import * as queryModelDefinition from "./grpc_query_model_definition.js"

import {
    genHeader,
    base64_image,
} from "./helpers.js";

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_service.proto');
client.load(['proto'], 'healthcheck.proto');

export function setup() {
}

export default () => {
    // Liveness check
    {
        group("Model API: Liveness", () => {
            client.connect('model-backend:8083', {
                plaintext: true
            });
            const response = client.invoke('vdp.model.v1alpha.ModelService/Liveness', {});
            check(response, {
                'Status is OK': (r) => r && r.status === grpc.StatusOK,
                'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
            });
        });
    }

    // Readiness check
    group("Model API: Readiness", () => {
        client.connect('model-backend:8083', {
            plaintext: true
        });
        const response = client.invoke('vdp.model.v1alpha.ModelService/Readiness', {});
        check(response, {
            'Status is OK': (r) => r && r.status === grpc.StatusOK,
            'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
        });
        client.close();
    });

    // Create model API
    createModel.CreateModel()

    // Update model API
    updateModel.UpdateModel()

    // Deploy Model API
    deployModel.DeployUndeployModel()

    // Query Model API
    queryModel.GetModel()
    queryModel.ListModel()
    queryModel.LookupModel()

    // Publish Model API
    publishModel.PublishUnPublishModel()

    // Infer Model API
    inferModel.InferModel()

    // Query Model Instance API
    queryModelInstance.GetModelInstance()
    queryModelInstance.ListModelInstance()
    queryModelInstance.LookupModelInstance()

    // Query Model Definition API
    queryModelDefinition.GetModelDefinition()
    queryModelDefinition.ListModelDefinition()
};

export function teardown() {
    client.connect('model-backend:8083', {
        plaintext: true
    });
    group("Model API: Delete all models created by this test", () => {
        for (const model of client.invoke('vdp.model.v1alpha.ModelService/ListModel', {}, {}).message.models) {
            check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: model.name }), {
                'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
            });
        }
    });
    client.close();
}
