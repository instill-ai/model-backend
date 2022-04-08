import grpc from 'k6/net/grpc';
import grpc from 'k6/net/grpc';
import { check, sleep, group } from 'k6';
import http from "k6/http";
import {FormData} from "https://jslib.k6.io/formdata/0.0.2/index.js";
import {randomString} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import {URL} from "https://jslib.k6.io/url/1.0.0/index.js";

import {
    genHeader,
    base64_image,
} from "./helpers.js";

const client = new grpc.Client();
client.load(['proto'], 'model.proto');

const apiHost = "http://localhost:8080";
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");

export function setup() {
}

export default () => {
    // Liveness check
    {
        group("Model API: Liveness", () => {
            client.connect('localhost:8080', {
                plaintext: true
            });
            const response = client.invoke('instill.model.v1alpha.ModelService/Liveness', {});
            check(response, {
                'Status is OK': (r) => r && r.status === grpc.StatusOK,
                'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.status === "SERVING_STATUS_SERVING",
            });
        });
    }

    // Readiness check
    group("Model API: Readiness", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        const response = client.invoke('instill.model.v1alpha.ModelService/Readiness', {});
        check(response, {
            'Status is OK': (r) => r && r.status === grpc.StatusOK,
            'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.status === "SERVING_STATUS_SERVING",
        });
        client.close();
    });

    // CreateModelBinaryFileUpload check
    group("Model API: CreateModelBinaryFileUpload", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        let response = client.invoke('instill.model.v1alpha.ModelService/CreateModelBinaryFileUpload', {});
        check(response, {
            'Missing stream body status': (r) => r && r.status == grpc.StatusInvalidArgument,  //TODO: need update to grpc code
        });

        client.close();
    });

    // ListModel check
    group("Model API: ListModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_name_cls = randomString(10)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", randomString(20));
        fd_cls.append("task", "TASK_CLASSIFICATION");
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name !== undefined,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name !== undefined,
            "POST /models/upload (multipart) task cls response model.task": (r) =>
            r.json().model.task === "TASK_CLASSIFICATION",
            "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
            r.json().model.model_versions.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/ListModel', {}, {}), {
            'ListModel status': (r) => r && r.status === grpc.StatusOK,
            'ListModel models length': (r) => r && r.message.models.length > 0,
            'ListModel model name': (r) => r && r.message.models[0].name == model_name_cls,
            'ListModel model fullName': (r) => r && r.message.models[0].fullName === `local-user/${model_name_cls}`,
            'ListModel model task': (r) => r && r.message.models[0].task === "TASK_CLASSIFICATION",
            'ListModel model modelVersions length': (r) => r && r.message.models[0].modelVersions.length > 0,
            'ListModel model modelVersions status': (r) => r && r.message.models[0].modelVersions[0].status === "STATUS_OFFLINE",
            'ListModel model modelVersions version': (r) => r && r.message.models[0].modelVersions[0].version == 1, //response is string ?
            'ListModel model modelVersions modelId': (r) => r && r.message.models[0].modelVersions[0].modelId !== undefined,
            'ListModel model modelVersions description': (r) => r && r.message.models[0].modelVersions[0].description !== undefined,
            'ListModel model modelVersions createdAt': (r) => r && r.message.models[0].modelVersions[0].createdAt !== undefined,
            'ListModel model modelVersions updatedAt': (r) => r && r.message.models[0].modelVersions[0].updatedAt !== undefined,
            'ListModel model modelVersions modelId': (r) => r && r.message.models[0].modelVersions[0].modelId !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // GetModel check
    group("Model API: GetModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_name_cls = randomString(10)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", randomString(20));
        fd_cls.append("task", "TASK_CLASSIFICATION");
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name !== undefined,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name !== undefined,
            "POST /models/upload (multipart) task cls response model.task": (r) =>
            r.json().model.task === "TASK_CLASSIFICATION",
            "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
            r.json().model.model_versions.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel status': (r) => r && r.status === grpc.StatusOK,
            'GetModel model name': (r) => r && r.message.model.name == model_name_cls,
            'GetModel model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name_cls}`,
            'GetModel model task': (r) => r && r.message.model.task === "TASK_CLASSIFICATION",
            'GetModel model modelVersions length': (r) => r && r.message.model.modelVersions.length > 0,
            'GetModel model modelVersions status': (r) => r && r.message.model.modelVersions[0].status === "STATUS_OFFLINE",
            'GetModel model modelVersions version': (r) => r && r.message.model.modelVersions[0].version == 1, //response is string ?
            'GetModel model modelVersions modelId': (r) => r && r.message.model.modelVersions[0].modelId !== undefined,
            'GetModel model modelVersions description': (r) => r && r.message.model.modelVersions[0].description !== undefined,
            'GetModel model modelVersions createdAt': (r) => r && r.message.model.modelVersions[0].createdAt !== undefined,
            'GetModel model modelVersions updatedAt': (r) => r && r.message.model.modelVersions[0].updatedAt !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: randomString(10)}, {}), {
            'GetModel non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // UpdateModelVersion check
    group("Model API: UpdateModelVersion", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_name_cls = randomString(10)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", randomString(20));
        fd_cls.append("task", "TASK_CLASSIFICATION");
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name !== undefined,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name !== undefined,
            "POST /models/upload (multipart) task cls response model.task": (r) =>
            r.json().model.task === "TASK_CLASSIFICATION",
            "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
            r.json().model.model_versions.length === 1,
        });

        let description = randomString(10)
        let req = {name: model_name_cls, version: 1, version_patch: {description: description}, field_mask: "description"}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', req, {}), {
            'UpdateModelVersion 1st status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelVersion 1st modelVersion status': (r) => r && r.message.modelVersion.status === "STATUS_OFFLINE",
            'UpdateModelVersion 1st modelVersion version': (r) => r && r.message.modelVersion.version == 1, //response is string ?
            'UpdateModelVersion 1st modelVersion modelId': (r) => r && r.message.modelVersion.modelId !== undefined,
            'UpdateModelVersion 1st modelVersion description': (r) => r && r.message.modelVersion.description == description,
            'UpdateModelVersion 1st modelVersion createdAt': (r) => r && r.message.modelVersion.createdAt !== undefined,
            'UpdateModelVersion 1st modelVersion updatedAt': (r) => r && r.message.modelVersion.updatedAt !== undefined,
        });

        req = {name: model_name_cls, version: 1, version_patch: {status: "STATUS_ONLINE"}, field_mask: "status"}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', req, {}), {
            'UpdateModelVersion 2nd status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelVersion 2nd modelVersion status': (r) => r && r.message.modelVersion.status === "STATUS_ONLINE",
            'UpdateModelVersion 2nd modelVersion version': (r) => r && r.message.modelVersion.version == 1, //response is string ?
            'UpdateModelVersion 2nd modelVersion modelId': (r) => r && r.message.modelVersion.modelId !== undefined,
            'UpdateModelVersion 2nd modelVersion description': (r) => r && r.message.modelVersion.description !== undefined,
            'UpdateModelVersion 2nd modelVersion createdAt': (r) => r && r.message.modelVersion.createdAt !== undefined,
            'UpdateModelVersion 2nd modelVersion updatedAt': (r) => r && r.message.modelVersion.updatedAt !== undefined,
        });
        sleep(5) // triton take time after update status

        let new_description = randomString(10)
        req = {name: model_name_cls, version: 1, version_patch: {status: "STATUS_OFFLINE",description: new_description}, field_mask: "status,description"}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', req, {}), {
            'UpdateModelVersion 3rd status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelVersion 3rd modelVersion status': (r) => r && r.message.modelVersion.status === "STATUS_OFFLINE",
            'UpdateModelVersion 3rd modelVersion version': (r) => r && r.message.modelVersion.version == 1, //response is string ?
            'UpdateModelVersion 3rd modelVersion modelId': (r) => r && r.message.modelVersion.modelId !== undefined,
            'UpdateModelVersion 3rd modelVersion description': (r) => r && r.message.modelVersion.description == new_description,
            'UpdateModelVersion 3rd modelVersion createdAt': (r) => r && r.message.modelVersion.createdAt !== undefined,
            'UpdateModelVersion 3rd modelVersion updatedAt': (r) => r && r.message.modelVersion.updatedAt !== undefined,
        });

        sleep(5) // triton take time after update status

        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', {name: randomString(10), version: 1}), {
            'UpdateModelVersion non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', {name: model_name_cls, version: 999}, {}), {
            'UpdateModelVersion non-existed version status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // DeleteModel check
    group("Model API: DeleteModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: randomString(10)}, {}), {
            'DeleteModel non-exist model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        let fd_cls = new FormData();
        let model_name_cls = randomString(10)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", randomString(20));
        fd_cls.append("task", "TASK_CLASSIFICATION");
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name !== undefined,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name !== undefined,
            "POST /models/upload (multipart) task cls response model.task": (r) =>
            r.json().model.task === "TASK_CLASSIFICATION",
            "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
            r.json().model.model_versions.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel status': (r) => r && r.status === grpc.StatusOK,
            'GetModel model name': (r) => r && r.message.model.name == model_name_cls,
            'GetModel model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name_cls}`,
            'GetModel model task': (r) => r && r.message.model.task === "TASK_CLASSIFICATION",
            'GetModel model modelVersions length': (r) => r && r.message.model.modelVersions.length > 0,
            'GetModel model modelVersions status': (r) => r && r.message.model.modelVersions[0].status === "STATUS_OFFLINE",
            'GetModel model modelVersions version': (r) => r && r.message.model.modelVersions[0].version == 1, //response is string ?
            'GetModel model modelVersions modelId': (r) => r && r.message.model.modelVersions[0].modelId !== undefined,
            'GetModel model modelVersions description': (r) => r && r.message.model.modelVersions[0].description !== undefined,
            'GetModel model modelVersions createdAt': (r) => r && r.message.model.modelVersions[0].createdAt !== undefined,
            'GetModel model modelVersions updatedAt': (r) => r && r.message.model.modelVersions[0].updatedAt !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}, {}), {
            'DeleteModel status OK': (r) => r && r.status === grpc.StatusOK,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel after delete version status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusNotFound,
        });
        client.close();
    });

    // DeleteModelVersion check
    group("Model API: DeleteModelVersion", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_name_cls = randomString(10)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", randomString(20));
        fd_cls.append("task", "TASK_CLASSIFICATION");
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name !== undefined,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name !== undefined,
            "POST /models/upload (multipart) task cls response model.task": (r) =>
            r.json().model.task === "TASK_CLASSIFICATION",
            "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
            r.json().model.model_versions.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel status': (r) => r && r.status === grpc.StatusOK,
            'GetModel model name': (r) => r && r.message.model.name == model_name_cls,
            'GetModel model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name_cls}`,
            'GetModel model task': (r) => r && r.message.model.task === "TASK_CLASSIFICATION",
            'GetModel model modelVersions length': (r) => r && r.message.model.modelVersions.length > 0,
            'GetModel model modelVersions status': (r) => r && r.message.model.modelVersions[0].status === "STATUS_OFFLINE",
            'GetModel model modelVersions version': (r) => r && r.message.model.modelVersions[0].version == 1, //response is string ?
            'GetModel model modelVersions modelId': (r) => r && r.message.model.modelVersions[0].modelId !== undefined,
            'GetModel model modelVersions description': (r) => r && r.message.model.modelVersions[0].description !== undefined,
            'GetModel model modelVersions createdAt': (r) => r && r.message.model.modelVersions[0].createdAt !== undefined,
            'GetModel model modelVersions updatedAt': (r) => r && r.message.model.modelVersions[0].updatedAt !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModelVersion', {name: randomString(10), version:1}, {}), {
            'DeleteModelVersion non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModelVersion', {name: model_name_cls, version:999}, {}), {
            'DeleteModelVersion non-existed model version status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModelVersion', {name: model_name_cls, version:1}, {}), {
            'DeleteModelVersion status OK': (r) => r && r.status === grpc.StatusOK,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel after delete version status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusNotFound,
        });
        client.close();
    });

    // TriggerModel check
    group("Model API: TriggerModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_name_cls = randomString(10)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", randomString(20));
        fd_cls.append("task", "TASK_CLASSIFICATION");
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name !== undefined,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name !== undefined,
            "POST /models/upload (multipart) task cls response model.task": (r) =>
            r.json().model.task === "TASK_CLASSIFICATION",
            "POST /models/upload (multipart) task cls response model.model_versions.length": (r) =>
            r.json().model.model_versions.length === 1,
        });

        let req = {name: model_name_cls, version: 1, version_patch: {status: "STATUS_ONLINE"}, field_mask: "status"}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', req, {}), {
            'UpdateModelVersion status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelVersion modelVersion status': (r) => r && r.message.modelVersion.status === "STATUS_ONLINE",
            'UpdateModelVersion modelVersion version': (r) => r && r.message.modelVersion.version == 1, //response is string ?
            'UpdateModelVersion modelVersion modelId': (r) => r && r.message.modelVersion.modelId !== undefined,
            'UpdateModelVersion modelVersion description': (r) => r && r.message.modelVersion.description !== undefined,
            'UpdateModelVersion modelVersion createdAt': (r) => r && r.message.modelVersion.createdAt !== undefined,
            'UpdateModelVersion modelVersion updatedAt': (r) => r && r.message.modelVersion.updatedAt !== undefined,
        });
        sleep(5) // triton take time after change status

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {name: model_name_cls, version: 1, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModel output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModel output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModel output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {name: randomString(10), version: 1, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {name: model_name_cls, version: 999, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {name: model_name_cls, version: 1, inputs: [{image_url: "https://artifacts.instill.tech/non-existed.jpg"}]}, {}), {
            'TriggerModel non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // CreateModelByGitHub check
    group("Model API: CreateModelByGitHub", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        let model_name = randomString(10)
        let model_description = randomString(20)
        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
            }         
        }), {
            'status': (r) => r && r.status == grpc.StatusOK, 
            'model_name': (r) => r && r.message.model.name == model_name,
            'task': (r) => r && r.message.model.task == "TASK_CLASSIFICATION",
            'modelVersions': (r) => r && r.message.model.modelVersions.length == 1, 
            'modelVersions status': (r) => r && r.message.model.modelVersions[0].status == "STATUS_OFFLINE",
            'modelVersions version': (r) => r && r.message.model.modelVersions[0].version == 1, 
            'modelVersions modelId': (r) => r && r.message.model.modelVersions[0].modelId != undefined,
            'modelVersions description': (r) => r && r.message.model.modelVersions[0].description == model_description,
            'modelVersions createdAt': (r) => r && r.message.model.modelVersions[0].createdAt != undefined,
            'modelVersions updatedAt': (r) => r && r.message.model.modelVersions[0].updatedAt != undefined,
            'modelVersions repoUrl': (r) => r && r.message.model.modelVersions[0].github.repoUrl == "https://github.com/Phelan164/test-repo.git",
        });

        sleep(5)
        let req = {name: model_name, version: 1, version_patch: {status: "STATUS_ONLINE"}, field_mask: "status"}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelVersion', req, {}), {
            'UpdateModelVersion status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelVersion modelVersion status': (r) => r && r.message.modelVersion.status === "STATUS_ONLINE",
            'UpdateModelVersion modelVersion version': (r) => r && r.message.modelVersion.version == 1, //response is string ?
            'UpdateModelVersion modelVersion modelId': (r) => r && r.message.modelVersion.modelId !== undefined,
            'UpdateModelVersion modelVersion description': (r) => r && r.message.modelVersion.description !== undefined,
            'UpdateModelVersion modelVersion createdAt': (r) => r && r.message.modelVersion.createdAt !== undefined,
            'UpdateModelVersion modelVersion updatedAt': (r) => r && r.message.modelVersion.updatedAt !== undefined,
        });
        sleep(5) // triton take time after change status

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {name: model_name, version: 1, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModel output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModel output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModel output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });

        let model_description_version2 = randomString(20)
        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "commit": "641c76de930003ac9f8dfc4d6b7430a9a98e305b"
                }
            }          
        }), {
            '2nd status': (r) => r && r.status == grpc.StatusOK, 
            '2nd model_name': (r) => r && r.message.model.name == model_name,
            '2nd task': (r) => r && r.message.model.task == "TASK_CLASSIFICATION",
            'modelVersions 2nd': (r) => r && r.message.model.modelVersions.length == 2, 
            'modelVersions 2nd status': (r) => r && r.message.model.modelVersions[1].status == "STATUS_OFFLINE",
            'modelVersions 2nd version': (r) => r && r.message.model.modelVersions[1].version == 2, 
            'modelVersions 2nd modelId': (r) => r && r.message.model.modelVersions[1].modelId != undefined,
            'modelVersions 2nd description': (r) => r && r.message.model.modelVersions[1].description == model_description_version2,
            'modelVersions 2nd createdAt': (r) => r && r.message.model.modelVersions[1].createdAt != undefined,
            'modelVersions 2nd updatedAt': (r) => r && r.message.model.modelVersions[1].updatedAt != undefined,
            'modelVersions 2nd github repoUrl': (r) => r && r.message.model.modelVersions[1].github.repoUrl == "https://github.com/Phelan164/test-repo.git",
            'modelVersions 2nd github gitRef commit': (r) => r && r.message.model.modelVersions[1].github.gitRef.commit == "641c76de930003ac9f8dfc4d6b7430a9a98e305b",
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "tag": "v1.0"
                }
            }          
        }), {
            '3rd status': (r) => r && r.status == grpc.StatusOK, 
            '3rd model_name': (r) => r && r.message.model.name == model_name,
            '3rd task': (r) => r && r.message.model.task == "TASK_CLASSIFICATION",
            'modelVersions 3rd': (r) => r && r.message.model.modelVersions.length == 3, 
            'modelVersions 3rd status': (r) => r && r.message.model.modelVersions[2].status == "STATUS_OFFLINE",
            'modelVersions 3rd version': (r) => r && r.message.model.modelVersions[2].version == 3, 
            'modelVersions 3rd modelId': (r) => r && r.message.model.modelVersions[2].modelId != undefined,
            'modelVersions 3rd description': (r) => r && r.message.model.modelVersions[2].description == model_description_version2,
            'modelVersions 3rd createdAt': (r) => r && r.message.model.modelVersions[2].createdAt != undefined,
            'modelVersions 3rd updatedAt': (r) => r && r.message.model.modelVersions[2].updatedAt != undefined,
            'modelVersions 3rd github repoUrl': (r) => r && r.message.model.modelVersions[2].github.repoUrl == "https://github.com/Phelan164/test-repo.git",
            'modelVersions 3rd github gitRef tag': (r) => r && r.message.model.modelVersions[2].github.gitRef.tag == "v1.0",
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "branch": "feat-a"
                }
            }          
        }), {
            'status': (r) => r && r.status == grpc.StatusOK, 
            'model_name': (r) => r && r.message.model.name == model_name,
            'task': (r) => r && r.message.model.task == "TASK_CLASSIFICATION",
            'modelVersions 3rd': (r) => r && r.message.model.modelVersions.length == 4, 
            'modelVersions 3rd status': (r) => r && r.message.model.modelVersions[3].status == "STATUS_OFFLINE",
            'modelVersions 3rd version': (r) => r && r.message.model.modelVersions[3].version == 4, 
            'modelVersions 3rd modelId': (r) => r && r.message.model.modelVersions[3].modelId != undefined,
            'modelVersions 3rd description': (r) => r && r.message.model.modelVersions[3].description == model_description_version2,
            'modelVersions 3rd createdAt': (r) => r && r.message.model.modelVersions[3].createdAt != undefined,
            'modelVersions 3rd updatedAt': (r) => r && r.message.model.modelVersions[3].updatedAt != undefined,
            'modelVersions 3rd github repoUrl': (r) => r && r.message.model.modelVersions[3].github.repoUrl == "https://github.com/Phelan164/test-repo.git",
            'modelVersions 3rd github gitRef branch': (r) => r && r.message.model.modelVersions[3].github.gitRef.branch == "feat-a",
        });        

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "branch": "non-existed"
                }
            }          
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument, 
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "tag": "non-existed"
                }
            }          
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument, 
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "commit": "non-existed"
                }
            }          
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument, 
        });        

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description_version2,
            "github": {
                "repo_url": "https://github.com/Phelan164/invalid-repo.git",
            }  
        }), {
            'invalid github repo status': (r) => r && r.status == grpc.StatusInvalidArgument, 
        });        

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "github": {
                "repo_url": "https://github.com/Phelan164/test-repo.git",
                "git_ref": {
                    "tag": "v1.0"
                }
            }         
        }), {
            'missing name status': (r) => r && r.status == grpc.StatusFailedPrecondition, 
        });     
        
        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "description": model_description,
        }), {
            'missing github url status': (r) => r && r.status == grpc.StatusFailedPrecondition, 
        });     

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });             

        client.close();
    });    

    sleep(1);
};

export function teardown() {
    client.connect('localhost:8080', {
        plaintext: true
    });
    group("Model API: Delete all models created by this test", () => {
        for (const model of client.invoke('instill.model.v1alpha.ModelService/ListModel', {}, {}).message.models) {
            check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model.name}), {
                'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
            });
        }
    });
    client.close();
}
