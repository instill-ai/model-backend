import grpc from 'k6/net/grpc';
import { check, group } from 'k6';
import http from "k6/http";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import {
    genHeader,
} from "./helpers.js";

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_service.proto');

const apiHost = "model-backend:8083";
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const model_def_name = "model-definitions/local"


export function PublishUnPublishModel() {
    // PublishModel/UnpublishModel check
    group("Model API: PublishModel/UnpublishModel", () => {
        client.connect(apiHost, {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `http://${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models:multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models:multipart (multipart) task cls response model.name": (r) =>
                r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models:multipart (multipart) task cls response model.uid": (r) =>
                r.json().model.uid !== undefined,
            "POST /v1alpha/models:multipart (multipart) task cls response model.id": (r) =>
                r.json().model.id === model_id,
            "POST /v1alpha/models:multipart (multipart) task cls response model.description": (r) =>
                r.json().model.description === model_description,
            "POST /v1alpha/models:multipart (multipart) task cls response model.model_definition": (r) =>
                r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models:multipart (multipart) task cls response model.configuration": (r) =>
                r.json().model.configuration !== undefined,
            "POST /v1alpha/models:multipart (multipart) task cls response model.visibility": (r) =>
                r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models:multipart (multipart) task cls response model.owner": (r) =>
                r.json().model.user === 'users/local-user',
            "POST /v1alpha/models:multipart (multipart) task cls response model.create_time": (r) =>
                r.json().model.create_time !== undefined,
            "POST /v1alpha/models:multipart (multipart) task cls response model.update_time": (r) =>
                r.json().model.update_time !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/PublishModel', { name: "models/" + model_id }), {
            "PublishModel response status": (r) => r.status === grpc.StatusOK,
            "PublishModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "PublishModel response model.uid": (r) => r.message.model.uid !== undefined,
            "PublishModel response model.id": (r) => r.message.model.id === model_id,
            "PublishModel response model.description": (r) => r.message.model.description === model_description,
            "PublishModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "PublishModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "PublishModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PUBLIC",
            "PublishModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "PublishModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "PublishModel response model.update_time": (r) => r.message.model.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/UnpublishModel', { name: "models/" + model_id }), {
            "UnpublishModel response status": (r) => r.status === grpc.StatusOK,
            "UnpublishModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "UnpublishModel response model.uid": (r) => r.message.model.uid !== undefined,
            "UnpublishModel response model.id": (r) => r.message.model.id === model_id,
            "UnpublishModel response model.description": (r) => r.message.model.description === model_description,
            "UnpublishModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "UnpublishModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "UnpublishModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "UnpublishModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "UnpublishModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "UnpublishModel response model.update_time": (r) => r.message.model.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/PublishModel', { name: "models/" + randomString(10) }), {
            "PublishModel response not found status": (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/UnpublishModel', { name: "models/" + randomString(10) }), {
            "UnpublishModel response not found status": (r) => r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
