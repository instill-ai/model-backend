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
client.load(['proto'], 'definition.proto');
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
            'Missing stream body status': (r) => r && r.status == grpc.StatusInvalidArgument,
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
        let model_description = randomString(20)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", model_description);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === model_name_cls,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name === `local-user/${model_name_cls}`,
            "POST /models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /models/upload (multipart) task cls response model.source": (r) =>
            r.json().model.source === "SOURCE_LOCAL",
            "POST /models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
            r.json().model.owner.id !== undefined,
            "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
            r.json().model.owner.username === "local-user",
            "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
            r.json().model.owner.type === "user",
            "POST /models/upload (multipart) task cls response model.created_at": (r) =>
            r.json().model.created_at !== undefined,
            "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
            r.json().model.updated_at !== undefined,
            "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
            r.json().model.instances.length === 1,
        });
        check(client.invoke('instill.model.v1alpha.ModelService/ListModel', {}, {}), {
            'ListModel status': (r) => r && r.status === grpc.StatusOK,
            'ListModel models length': (r) => r && r.message.models.length > 0,
            'ListModel model name': (r) => r && r.message.models[0].name == model_name_cls,
            'ListModel model fullName': (r) => r && r.message.models[0].fullName === `local-user/${model_name_cls}`,
            'ListModel model source': (r) => r && r.message.models[0].source === "SOURCE_LOCAL",
            'ListModel model description': (r) => r && r.message.models[0].description === model_description,
            'ListModel model visibility': (r) => r && r.message.models[0].visibility === "VISIBILITY_PRIVATE",
            'ListModel model createdAt': (r) => r && r.message.models[0].createdAt !== undefined,
            'ListModel model updatedAt': (r) => r && r.message.models[0].updatedAt !== undefined,
            'ListModel model id': (r) => r && r.message.models[0].id !== undefined,
            'ListModel model owner id': (r) => r && r.message.models[0].owner.id !== undefined,
            'ListModel model owner username': (r) => r && r.message.models[0].owner.username === "local-user",
            'ListModel model owner type': (r) => r && r.message.models[0].owner.type === "user",
            'ListModel model instances length': (r) => r && r.message.models[0].instances.length > 0,
            'ListModel model instances status': (r) => r && r.message.models[0].instances[0].status === "STATUS_OFFLINE",
            'ListModel model instances name': (r) => r && r.message.models[0].instances[0].name === "latest",
            'ListModel model instances task': (r) => r && r.message.models[0].instances[0].task === "TASK_CLASSIFICATION",
            'ListModel model instances modelDefinitionId': (r) => r && r.message.models[0].instances[0].modelDefinitionId === r.message.models[0].id,
            'ListModel model instances modelDefinitionSource': (r) => r && r.message.models[0].instances[0].modelDefinitionSource === r.message.models[0].source,
            'ListModel model instances createdAt': (r) => r && r.message.models[0].instances[0].createdAt !== undefined,
            'ListModel model instances updatedAt': (r) => r && r.message.models[0].instances[0].updatedAt !== undefined,
            'ListModel model instances modelDefinitionName': (r) => r && r.message.models[0].instances[0].modelDefinitionName === model_name_cls,
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
        let model_description = randomString(20)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", model_description);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === model_name_cls,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name === `local-user/${model_name_cls}`,
            "POST /models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /models/upload (multipart) task cls response model.source": (r) =>
            r.json().model.source === "SOURCE_LOCAL",
            "POST /models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
            r.json().model.owner.id !== undefined,
            "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
            r.json().model.owner.username === "local-user",
            "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
            r.json().model.owner.type === "user",
            "POST /models/upload (multipart) task cls response model.created_at": (r) =>
            r.json().model.created_at !== undefined,
            "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
            r.json().model.updated_at !== undefined,
            "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
            r.json().model.instances.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel status': (r) => r && r.status === grpc.StatusOK,
            'GetModel model name': (r) => r && r.message.model.name == model_name_cls,
            'GetModel model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name_cls}`,
            'GetModel model source': (r) => r && r.message.model.source === "SOURCE_LOCAL",
            'GetModel model description': (r) => r && r.message.model.description === model_description,
            'GetModel model visibility': (r) => r && r.message.model.visibility === "VISIBILITY_PRIVATE",
            'GetModel model createdAt': (r) => r && r.message.model.createdAt !== undefined,
            'GetModel model updatedAt': (r) => r && r.message.model.updatedAt !== undefined,
            'GetModel model id': (r) => r && r.message.model.id !== undefined,
            'GetModel model owner id': (r) => r && r.message.model.owner.id !== undefined,
            'GetModel model owner username': (r) => r && r.message.model.owner.username === "local-user",
            'GetModel model owner type': (r) => r && r.message.model.owner.type === "user",
            'GetModel model instances length': (r) => r && r.message.model.instances.length > 0,
            'GetModel model instances status': (r) => r && r.message.model.instances[0].status === "STATUS_OFFLINE",
            'GetModel model instances name': (r) => r && r.message.model.instances[0].name === "latest",
            'GetModel model instances task': (r) => r && r.message.model.instances[0].task === "TASK_CLASSIFICATION",
            'GetModel model instances modelDefinitionId': (r) => r && r.message.model.instances[0].modelDefinitionId === r.message.model.id,
            'GetModel model instances modelDefinitionSource': (r) => r && r.message.model.instances[0].modelDefinitionSource === r.message.model.source,
            'GetModel model instances createdAt': (r) => r && r.message.model.instances[0].createdAt !== undefined,
            'GetModel model instances updatedAt': (r) => r && r.message.model.instances[0].updatedAt !== undefined,
            'GetModel model instances modelDefinitionName': (r) => r && r.message.model.instances[0].modelDefinitionName === model_name_cls,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: randomString(10)}, {}), {
            'GetModel non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model_name_cls}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // UpdateModelInstance check
    group("Model API: UpdateModelInstance", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let model_name_cls = randomString(10)
        let fd_cls = new FormData();
        let model_description = randomString(20)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", model_description);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === model_name_cls,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name === `local-user/${model_name_cls}`,
            "POST /models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /models/upload (multipart) task cls response model.source": (r) =>
            r.json().model.source === "SOURCE_LOCAL",
            "POST /models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
            r.json().model.owner.id !== undefined,
            "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
            r.json().model.owner.username === "local-user",
            "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
            r.json().model.owner.type === "user",
            "POST /models/upload (multipart) task cls response model.created_at": (r) =>
            r.json().model.created_at !== undefined,
            "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
            r.json().model.updated_at !== undefined,
            "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
            r.json().model.instances.length === 1,
        });

        let req = {model_name: model_name_cls, instance_name: "latest", status: 2}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelInstance', req, {}), {
            'UpdateModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelInstance instance status': (r) => r && r.message.instance.status === "STATUS_ONLINE",
            'UpdateModelInstance instance modelDefinitionName': (r) => r && r.message.instance.modelDefinitionName === model_name_cls,
            'UpdateModelInstance instance name': (r) => r && r.message.instance.name === "latest",
            'UpdateModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'UpdateModelInstance instance modelDefinitionSource': (r) => r && r.message.instance.modelDefinitionSource === "SOURCE_LOCAL",
            'UpdateModelInstance instance id': (r) => r && r.message.instance.id !== undefined,
            'UpdateModelInstance instance createdAt': (r) => r && r.message.instance.createdAt !== undefined,
            'UpdateModelInstance instance updatedAt': (r) => r && r.message.instance.updatedAt !== undefined,
        });
        sleep(5) // triton take time after update status

        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelInstance', {model_name: randomString(10), instance_name: "latest"}), {
            'UpdateModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelInstance', {model_name: model_name_cls, instance_name: "non-existed"}, {}), {
            'UpdateModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusNotFound,
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

        let model_name_cls = randomString(10)
        let fd_cls = new FormData();
        let model_description = randomString(20)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", model_description);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === model_name_cls,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name === `local-user/${model_name_cls}`,
            "POST /models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /models/upload (multipart) task cls response model.source": (r) =>
            r.json().model.source === "SOURCE_LOCAL",
            "POST /models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
            r.json().model.owner.id !== undefined,
            "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
            r.json().model.owner.username === "local-user",
            "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
            r.json().model.owner.type === "user",
            "POST /models/upload (multipart) task cls response model.created_at": (r) =>
            r.json().model.created_at !== undefined,
            "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
            r.json().model.updated_at !== undefined,
            "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
            r.json().model.instances.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel status': (r) => r && r.status === grpc.StatusOK,
            'GetModel model name': (r) => r && r.message.model.name == model_name_cls,
            'GetModel model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name_cls}`,
            'GetModel model source': (r) => r && r.message.model.source === "SOURCE_LOCAL",
            'GetModel model description': (r) => r && r.message.model.description === model_description,
            'GetModel model visibility': (r) => r && r.message.model.visibility === "VISIBILITY_PRIVATE",
            'GetModel model createdAt': (r) => r && r.message.model.createdAt !== undefined,
            'GetModel model updatedAt': (r) => r && r.message.model.updatedAt !== undefined,
            'GetModel model id': (r) => r && r.message.model.id !== undefined,
            'GetModel model owner id': (r) => r && r.message.model.owner.id !== undefined,
            'GetModel model owner username': (r) => r && r.message.model.owner.username === "local-user",
            'GetModel model owner type': (r) => r && r.message.model.owner.type === "user",
            'GetModel model instances length': (r) => r && r.message.model.instances.length > 0,
            'GetModel model instances status': (r) => r && r.message.model.instances[0].status === "STATUS_OFFLINE",
            'GetModel model instances name': (r) => r && r.message.model.instances[0].name === "latest",
            'GetModel model instances task': (r) => r && r.message.model.instances[0].task === "TASK_CLASSIFICATION",
            'GetModel model instances modelDefinitionId': (r) => r && r.message.model.instances[0].modelDefinitionId === r.message.model.id,
            'GetModel model instances modelDefinitionSource': (r) => r && r.message.model.instances[0].modelDefinitionSource === r.message.model.source,
            'GetModel model instances createdAt': (r) => r && r.message.model.instances[0].createdAt !== undefined,
            'GetModel model instances updatedAt': (r) => r && r.message.model.instances[0].updatedAt !== undefined,
            'GetModel model instances modelDefinitionName': (r) => r && r.message.model.instances[0].modelDefinitionName === model_name_cls,
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

    // DeleteModelInstance check
    group("Model API: DeleteModelInstance", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let model_name_cls = randomString(10)
        let fd_cls = new FormData();
        let model_description = randomString(20)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", model_description);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === model_name_cls,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name === `local-user/${model_name_cls}`,
            "POST /models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /models/upload (multipart) task cls response model.source": (r) =>
            r.json().model.source === "SOURCE_LOCAL",
            "POST /models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
            r.json().model.owner.id !== undefined,
            "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
            r.json().model.owner.username === "local-user",
            "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
            r.json().model.owner.type === "user",
            "POST /models/upload (multipart) task cls response model.created_at": (r) =>
            r.json().model.created_at !== undefined,
            "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
            r.json().model.updated_at !== undefined,
            "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
            r.json().model.instances.length === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: model_name_cls}, {}), {
            'GetModel status': (r) => r && r.status === grpc.StatusOK,
            'GetModel model name': (r) => r && r.message.model.name == model_name_cls,
            'GetModel model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name_cls}`,
            'GetModel model source': (r) => r && r.message.model.source === "SOURCE_LOCAL",
            'GetModel model description': (r) => r && r.message.model.description === model_description,
            'GetModel model visibility': (r) => r && r.message.model.visibility === "VISIBILITY_PRIVATE",
            'GetModel model createdAt': (r) => r && r.message.model.createdAt !== undefined,
            'GetModel model updatedAt': (r) => r && r.message.model.updatedAt !== undefined,
            'GetModel model id': (r) => r && r.message.model.id !== undefined,
            'GetModel model owner id': (r) => r && r.message.model.owner.id !== undefined,
            'GetModel model owner username': (r) => r && r.message.model.owner.username === "local-user",
            'GetModel model owner type': (r) => r && r.message.model.owner.type === "user",
            'GetModel model instances length': (r) => r && r.message.model.instances.length > 0,
            'GetModel model instances status': (r) => r && r.message.model.instances[0].status === "STATUS_OFFLINE",
            'GetModel model instances name': (r) => r && r.message.model.instances[0].name === "latest",
            'GetModel model instances task': (r) => r && r.message.model.instances[0].task === "TASK_CLASSIFICATION",
            'GetModel model instances modelDefinitionId': (r) => r && r.message.model.instances[0].modelDefinitionId === r.message.model.id,
            'GetModel model instances modelDefinitionSource': (r) => r && r.message.model.instances[0].modelDefinitionSource === r.message.model.source,
            'GetModel model instances createdAt': (r) => r && r.message.model.instances[0].createdAt !== undefined,
            'GetModel model instances updatedAt': (r) => r && r.message.model.instances[0].updatedAt !== undefined,
            'GetModel model instances modelDefinitionName': (r) => r && r.message.model.instances[0].modelDefinitionName === model_name_cls,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModelInstance', {model_name: randomString(10), instance_name: "latest"}, {}), {
            'DeleteModelInstance non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModelInstance', {model_name: model_name_cls, instance_name: "non-existed"}, {}), {
            'DeleteModelInstance non-existed model version status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModelInstance', {model_name: model_name_cls, instance_name: "latest"}, {}), {
            'DeleteModelInstance status OK': (r) => r && r.status === grpc.StatusOK,
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

        let model_name_cls = randomString(10)
        let fd_cls = new FormData();
        let model_description = randomString(20)
        fd_cls.append("name", model_name_cls);
        fd_cls.append("description", model_description);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /models/upload (multipart) task cls response status": (r) =>
            r.status === 200, // TODO: update status to 201
            "POST /models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === model_name_cls,
            "POST /models/upload (multipart) task cls response model.full_name": (r) =>
            r.json().model.full_name === `local-user/${model_name_cls}`,
            "POST /models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /models/upload (multipart) task cls response model.source": (r) =>
            r.json().model.source === "SOURCE_LOCAL",
            "POST /models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /models/upload (multipart) task cls response model.owner.id": (r) =>
            r.json().model.owner.id !== undefined,
            "POST /models/upload (multipart) task cls response model.owner.username": (r) =>
            r.json().model.owner.username === "local-user",
            "POST /models/upload (multipart) task cls response model.owner.type": (r) =>
            r.json().model.owner.type === "user",
            "POST /models/upload (multipart) task cls response model.created_at": (r) =>
            r.json().model.created_at !== undefined,
            "POST /models/upload (multipart) task cls response model.updated_at": (r) =>
            r.json().model.updated_at !== undefined,
            "POST /models/upload (multipart) task cls response model.instances.length": (r) =>
            r.json().model.instances.length === 1,
        });
        sleep(5)

        let req = {model_name: model_name_cls, instance_name: "latest", status: 2}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelInstance', req, {}), {
            'UpdateModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelInstance instance status': (r) => r && r.message.instance.status === "STATUS_ONLINE",
            'UpdateModelInstance instance modelDefinitionName': (r) => r && r.message.instance.modelDefinitionName === model_name_cls,
            'UpdateModelInstance instance name': (r) => r && r.message.instance.name === "latest",
            'UpdateModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'UpdateModelInstance instance modelDefinitionSource': (r) => r && r.message.instance.modelDefinitionSource === "SOURCE_LOCAL",
            'UpdateModelInstance instance id': (r) => r && r.message.instance.id !== undefined,
            'UpdateModelInstance instance createdAt': (r) => r && r.message.instance.createdAt !== undefined,
            'UpdateModelInstance instance updatedAt': (r) => r && r.message.instance.updatedAt !== undefined,
        });
        sleep(5) // triton take time after update status

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {model_name: model_name_cls, instance_name: "latest", inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModel output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModel output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModel output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {model_name: randomString(10), instance_name: "latest", inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {model_name: model_name_cls, instance_name: "non-existed", inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {model_name: model_name_cls, instance_name: "latest", inputs: [{image_url: "https://artifacts.instill.tech/non-existed.jpg"}]}, {}), {
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
        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "github": {
                "repo": "https://github.com/Phelan164/test-repo.git",
                "tag": "v1.0"
            }
        }), {
            'CreateModelByGitHub status': (r) => r && r.status === grpc.StatusOK,
            'CreateModelByGitHub model name': (r) => r && r.message.model.name == model_name,
            'CreateModelByGitHub model fullName': (r) => r && r.message.model.fullName === `local-user/${model_name}`,
            'CreateModelByGitHub model source': (r) => r && r.message.model.source === "SOURCE_GITHUB",
            'CreateModelByGitHub model description': (r) => r && r.message.model.description !== undefined,
            'CreateModelByGitHub model visibility': (r) => r && r.message.model.visibility === "VISIBILITY_PUBLIC",
            'CreateModelByGitHub model createdAt': (r) => r && r.message.model.createdAt !== undefined,
            'CreateModelByGitHub model updatedAt': (r) => r && r.message.model.updatedAt !== undefined,
            'CreateModelByGitHub model id': (r) => r && r.message.model.id !== undefined,
            'CreateModelByGitHub model configuration repo': (r) => r && r.message.model.configuration.repo === "https://github.com/Phelan164/test-repo.git",
            'CreateModelByGitHub model configuration htmlUrl': (r) => r && r.message.model.configuration.htmlUrl === "",
            'CreateModelByGitHub model owner id': (r) => r && r.message.model.owner.id !== undefined,
            'CreateModelByGitHub model owner username': (r) => r && r.message.model.owner.username === "local-user",
            'CreateModelByGitHub model owner type': (r) => r && r.message.model.owner.type === "user",
            'CreateModelByGitHub model instances length': (r) => r && r.message.model.instances.length > 0,
            'CreateModelByGitHub model instances status': (r) => r && r.message.model.instances[0].status === "STATUS_OFFLINE",
            'CreateModelByGitHub model instances name': (r) => r && r.message.model.instances[0].name === "v1.0",
            'CreateModelByGitHub model instances task': (r) => r && r.message.model.instances[0].task === "TASK_CLASSIFICATION",
            'CreateModelByGitHub model instances modelDefinitionId': (r) => r && r.message.model.instances[0].modelDefinitionId === r.message.model.id,
            'CreateModelByGitHub model instances modelDefinitionSource': (r) => r && r.message.model.instances[0].modelDefinitionSource === r.message.model.source,
            'CreateModelByGitHub model instances createdAt': (r) => r && r.message.model.instances[0].createdAt !== undefined,
            'CreateModelByGitHub model instances updatedAt': (r) => r && r.message.model.instances[0].updatedAt !== undefined,
            'CreateModelByGitHub model instances modelDefinitionName': (r) => r && r.message.model.instances[0].modelDefinitionName === model_name, 
            'CreateModelByGitHub model instances configuration repo': (r) => r && r.message.model.instances[0].configuration.repo === "https://github.com/Phelan164/test-repo.git",
            'CreateModelByGitHub model instances configuration tag': (r) => r && r.message.model.instances[0].configuration.tag === "v1.0",
            'CreateModelByGitHub model instances configuration htmlUrl': (r) => r && r.message.model.instances[0].configuration.htmlUrl === "",
        });
        sleep(5)

        let req = {model_name: model_name, instance_name: "v1.0", status: 2}
        check(client.invoke('instill.model.v1alpha.ModelService/UpdateModelInstance', req, {}), {
            'UpdateModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'UpdateModelInstance instance status': (r) => r && r.message.instance.status === "STATUS_ONLINE",
            'UpdateModelInstance instance modelDefinitionName': (r) => r && r.message.instance.modelDefinitionName === model_name,
            'UpdateModelInstance instance name': (r) => r && r.message.instance.name === "v1.0",
            'UpdateModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'UpdateModelInstance instance modelDefinitionSource': (r) => r && r.message.instance.modelDefinitionSource === "SOURCE_GITHUB",
            'UpdateModelInstance instance id': (r) => r && r.message.instance.id !== undefined,
            'UpdateModelInstance instance createdAt': (r) => r && r.message.instance.createdAt !== undefined,
            'UpdateModelInstance instance updatedAt': (r) => r && r.message.instance.updatedAt !== undefined,
        });
        sleep(5) // triton take time after update status

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModel', {model_name: model_name, instance_name: "v1.0", inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModel output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModel output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModel output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "github": {
                "repo": "https://github.com/Phelan164/test-repo.git",
                "tag": "non-existed"
            }
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
            "github": {
                "repo": "https://github.com/Phelan164/invalid-repo.git",
            }
        }), {
            'invalid github repo status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "github": {
                "repo": "https://github.com/Phelan164/test-repo.git",
                "tag": "v1.0"
            }
        }), {
            'missing name status': (r) => r && r.status == grpc.StatusFailedPrecondition,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelByGitHub', {
            "name": model_name,
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
