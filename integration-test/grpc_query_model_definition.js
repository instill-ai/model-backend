import grpc from 'k6/net/grpc';
import {
    check,
    group
} from 'k6';

import * as constant from "./const.js"

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_public_service.proto');

const model_def_name = "model-definitions/github"

export function GetModelDefinition() {
    group("Model API: GetModelDefinition", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });
        check(client.invoke('vdp.model.v1alpha.ModelPublicService/GetModelDefinition', { name: model_def_name }, {}), {
            "GetModelDefinition response status": (r) => r.status === grpc.StatusOK,
            "GetModelDefinition response modelDefinition.name": (r) => r.message.modelDefinition.name === model_def_name,
            "GetModelDefinition response modelDefinition.uid": (r) => r.message.modelDefinition.uid !== undefined,
            "GetModelDefinition response modelDefinition.id": (r) => r.message.modelDefinition.id === "github",
            "GetModelDefinition response modelDefinition.title": (r) => r.message.modelDefinition.title === "GitHub",
            "GetModelDefinition response modelDefinition.icon": (r) => r.message.modelDefinition.icon !== undefined,
            "GetModelDefinition response modelDefinition.documentationUrl": (r) => r.message.modelDefinition.documentationUrl !== undefined,
            "GetModelDefinition response modelDefinition.modelSpec": (r) => r.message.modelDefinition.modelSpec !== undefined,
            "GetModelDefinition response modelDefinition.modelInstanceSpec": (r) => r.message.modelDefinition.modelInstanceSpec !== undefined,
            "GetModelDefinition response modelDefinition.create_time": (r) => r.message.modelDefinition.createTime !== undefined,
            "GetModelDefinition response modelDefinition.update_time": (r) => r.message.modelDefinition.updateTime !== undefined,
        });
        client.close();
    });
};

export function ListModelDefinitions() {
    group("Model API: ListModelDefinitions", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });
        check(client.invoke('vdp.model.v1alpha.ModelPublicService/ListModelDefinitions', {}, {}), {
            "ListModelDefinitions response status": (r) => r.status === grpc.StatusOK,
            "ListModelDefinitions response modelDefinitions[2].name": (r) => r.message.modelDefinitions[2].name === "model-definitions/local",
            "ListModelDefinitions response modelDefinitions[2].uid": (r) => r.message.modelDefinitions[2].uid !== undefined,
            "ListModelDefinitions response modelDefinitions[2].id": (r) => r.message.modelDefinitions[2].id === "local",
            "ListModelDefinitions response modelDefinitions[2].title": (r) => r.message.modelDefinitions[2].title === "Local",
            "ListModelDefinitions response modelDefinitions[2].icon": (r) => r.message.modelDefinitions[2].icon !== undefined,
            "ListModelDefinitions response modelDefinitions[2].documentationUrl": (r) => r.message.modelDefinitions[2].documentationUrl !== undefined,
            "ListModelDefinitions response modelDefinitions[2].modelSpec": (r) => r.message.modelDefinitions[2].modelSpec !== undefined,
            "ListModelDefinitions response modelDefinitions[2].modelInstanceSpec": (r) => r.message.modelDefinitions[2].modelInstanceSpec !== undefined,
            "ListModelDefinitions response modelDefinitions[2].create_time": (r) => r.message.modelDefinitions[2].createTime !== undefined,
            "ListModelDefinitions response modelDefinitions[2].update_time": (r) => r.message.modelDefinitions[2].updateTime !== undefined,
        });
        client.close();
    });
};
