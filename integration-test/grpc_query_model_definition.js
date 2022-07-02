import grpc from 'k6/net/grpc';
import { check, sleep, group } from 'k6';
import http from "k6/http";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import {
    genHeader,
    base64_image,
} from "./helpers.js";

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_service.proto');

const apiHost = "http://model-backend:8083";
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const model_def_name = "model-definitions/github"

export function GetModelDefinition() {
    group("Model API: GetModelDefinition", () => {
        client.connect('model-backend:8083', {
            plaintext: true
        });
        check(client.invoke('vdp.model.v1alpha.ModelService/GetModelDefinition', { name: model_def_name }, {}), {
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

export function ListModelDefinition() {
    group("Model API: ListModelDefinition", () => {
        client.connect('model-backend:8083', {
            plaintext: true
        });
        check(client.invoke('vdp.model.v1alpha.ModelService/ListModelDefinition', {}, {}), {
            "ListModelDefinition response status": (r) => r.status === grpc.StatusOK,
            "ListModelDefinition response modelDefinitions[2].name": (r) => r.message.modelDefinitions[2].name === "model-definitions/local",
            "ListModelDefinition response modelDefinitions[2].uid": (r) => r.message.modelDefinitions[2].uid !== undefined,
            "ListModelDefinition response modelDefinitions[2].id": (r) => r.message.modelDefinitions[2].id === "local",
            "ListModelDefinition response modelDefinitions[2].title": (r) => r.message.modelDefinitions[2].title === "Local",
            "ListModelDefinition response modelDefinitions[2].icon": (r) => r.message.modelDefinitions[2].icon !== undefined,
            "ListModelDefinition response modelDefinitions[2].documentationUrl": (r) => r.message.modelDefinitions[2].documentationUrl !== undefined,
            "ListModelDefinition response modelDefinitions[2].modelSpec": (r) => r.message.modelDefinitions[2].modelSpec !== undefined,
            "ListModelDefinition response modelDefinitions[2].modelInstanceSpec": (r) => r.message.modelDefinitions[2].modelInstanceSpec !== undefined,
            "ListModelDefinition response modelDefinitions[2].create_time": (r) => r.message.modelDefinitions[2].createTime !== undefined,
            "ListModelDefinition response modelDefinitions[2].update_time": (r) => r.message.modelDefinitions[2].updateTime !== undefined,
        });
        client.close();
    });
};
