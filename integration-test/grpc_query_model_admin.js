import grpc from 'k6/net/grpc';
import {
    check,
    group,
    sleep
} from 'k6';
import http from "k6/http";
import {
    FormData
} from "https://jslib.k6.io/formdata/0.0.2/index.js";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import {
    genHeader,
} from "./helpers.js";

import * as constant from "./const.js"

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_public_service.proto');
client.load(['proto'], 'model_private_service.proto');

const model_def_name = "model-definitions/local"

export function GetModelAdmin() {
    // GetModelAdmin check
    group("Model API: GetModelAdmin", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
        let createClsModelRes = http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(createClsModelRes, {
            "POST /v1alpha/models/multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
                r.json().operation.name !== undefined,
        });

        // Check model creation finished
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            let res = client.invoke('vdp.model.v1alpha.ModelPublicService/GetModelOperation', {
                name: createClsModelRes.json().operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.model.v1alpha.ModelPrivateService/GetModelAdmin', {
            name: "models/" + model_id
        }, {}), {
            "GetModelAdmin response status": (r) => r.status === grpc.StatusOK,
            "GetModelAdmin response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "GetModelAdmin response model.uid": (r) => r.message.model.uid !== undefined,
            "GetModelAdmin response model.id": (r) => r.message.model.id === model_id,
            "GetModelAdmin response model.description": (r) => r.message.model.description === model_description,
            "GetModelAdmin response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "GetModelAdmin response model.configuration": (r) => r.message.model.configuration !== undefined,
            "GetModelAdmin response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "GetModelAdmin response model.owner": (r) => r.message.model.user === 'users/local-user',
            "GetModelAdmin response model.create_time": (r) => r.message.model.createTime !== undefined,
            "GetModelAdmin response model.update_time": (r) => r.message.model.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPrivateService/GetModelAdmin', {
            name: "models/" + randomString(10)
        }, {}), {
            'GetModel non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};


export function ListModelsAdmin() {
    // ListModelsAdmin check
    group("Model API: ListModelsAdmin", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
        let createClsModelRes = http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(createClsModelRes, {
            "POST /v1alpha/models/multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
                r.json().operation.name !== undefined,
        });

        // Check model creation finished
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            let res = client.invoke('vdp.model.v1alpha.ModelPublicService/GetModelOperation', {
                name: createClsModelRes.json().operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }
        check(client.invoke('vdp.model.v1alpha.ModelPrivateService/ListModelsAdmin', {}, {}), {
            "ListModelsAdmin response status": (r) => r.status === grpc.StatusOK,
            "ListModelsAdmin response total_size": (r) => r.message.totalSize >= 1,
            "ListModelsAdmin response next_page_token": (r) => r.message.nextPageToken !== undefined,
            "ListModelsAdmin response models.length": (r) => r.message.models.length >= 1,
            "ListModelsAdmin response models[0].name": (r) => r.message.models[0].name === `models/${model_id}`,
            "ListModelsAdmin response models[0].uid": (r) => r.message.models[0].uid !== undefined,
            "ListModelsAdmin response models[0].id": (r) => r.message.models[0].id === model_id,
            "ListModelsAdmin response models[0].description": (r) => r.message.models[0].description !== undefined,
            "ListModelsAdmin response models[0].model_definition": (r) => r.message.models[0].modelDefinition === model_def_name,
            "ListModelsAdmin response models[0].configuration": (r) => r.message.models[0].configuration !== undefined,
            "ListModelsAdmin response models[0].visibility": (r) => r.message.models[0].visibility === "VISIBILITY_PRIVATE",
            "ListModelsAdmin response models[0].owner": (r) => r.message.models[0].user === 'users/local-user',
            "ListModelsAdmin response models[0].create_time": (r) => r.message.models[0].createTime !== undefined,
            "ListModelsAdmin response models[0].update_time": (r) => r.message.models[0].updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};

export function LookUpModelAdmin() {
    // LookUpModelAdmin check
    group("Model API: LookUpModelAdmin", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
        let createClsModelRes = http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(createClsModelRes, {
            "POST /v1alpha/models/multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
                r.json().operation.name !== undefined,
        });

        // Check model creation finished
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            let res = client.invoke('vdp.model.v1alpha.ModelPublicService/GetModelOperation', {
                name: createClsModelRes.json().operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        let res = client.invoke('vdp.model.v1alpha.ModelPublicService/GetModelOperation', {
            name: createClsModelRes.json().operation.name
        }, {})
        check(client.invoke('vdp.model.v1alpha.ModelPrivateService/LookUpModelAdmin', {
            permalink: "models/" + res.message.operation.response.uid
        }, {}), {
            "LookUpModelAdmin response status": (r) => r.status === grpc.StatusOK,
            "LookUpModelAdmin response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "LookUpModelAdmin response model.uid": (r) => r.message.model.uid !== undefined,
            "LookUpModelAdmin response model.id": (r) => r.message.model.id === model_id,
            "LookUpModelAdmin response model.description": (r) => r.message.model.description === model_description,
            "LookUpModelAdmin response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "LookUpModelAdmin response model.configuration": (r) => r.message.model.configuration !== undefined,
            "LookUpModelAdmin response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "LookUpModelAdmin response model.owner": (r) => r.message.model.user === 'users/local-user',
            "LookUpModelAdmin response model.create_time": (r) => r.message.model.createTime !== undefined,
            "LookUpModelAdmin response model.update_time": (r) => r.message.model.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPrivateService/LookUpModelAdmin', {
            permalink: "models/" + randomString(10)
        }, {}), {
            'LookUpModelAdmin non-existed model status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });
        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
