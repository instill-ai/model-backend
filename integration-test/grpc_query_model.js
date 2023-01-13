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
client.load(['proto'], 'model_service.proto');

const model_def_name = "model-definitions/local"

export function GetModel() {
    // GetModel check
    group("Model API: GetModel", () => {
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
            let res = client.invoke('vdp.model.v1alpha.ModelService/GetModelOperation', {
                name: createClsModelRes.json().operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.model.v1alpha.ModelService/GetModel', {
            name: "models/" + model_id
        }, {}), {
            "GetModel response status": (r) => r.status === grpc.StatusOK,
            "GetModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "GetModel response model.uid": (r) => r.message.model.uid !== undefined,
            "GetModel response model.id": (r) => r.message.model.id === model_id,
            "GetModel response model.description": (r) => r.message.model.description === model_description,
            "GetModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "GetModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "GetModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "GetModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "GetModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "GetModel response model.update_time": (r) => r.message.model.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/GetModel', {
            name: "models/" + randomString(10)
        }, {}), {
            'GetModel non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });


        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};


export function ListModel() {
    // ListModel check
    group("Model API: ListModel", () => {
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
            let res = client.invoke('vdp.model.v1alpha.ModelService/GetModelOperation', {
                name: createClsModelRes.json().operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }
        check(client.invoke('vdp.model.v1alpha.ModelService/ListModel', {}, {}), {
            "ListModel response status": (r) => r.status === grpc.StatusOK,
            "ListModel response total_size": (r) => r.message.totalSize == 1,
            "ListModel response next_page_token": (r) => r.message.nextPageToken !== undefined,
            "ListModel response models.length": (r) => r.message.models.length === 1,
            "ListModel response models[0].name": (r) => r.message.models[0].name === `models/${model_id}`,
            "ListModel response models[0].uid": (r) => r.message.models[0].uid !== undefined,
            "ListModel response models[0].id": (r) => r.message.models[0].id === model_id,
            "ListModel response models[0].description": (r) => r.message.models[0].description === model_description,
            "ListModel response models[0].model_definition": (r) => r.message.models[0].modelDefinition === model_def_name,
            "ListModel response models[0].configuration": (r) => r.message.models[0].configuration !== undefined,
            "ListModel response models[0].visibility": (r) => r.message.models[0].visibility === "VISIBILITY_PRIVATE",
            "ListModel response models[0].owner": (r) => r.message.models[0].user === 'users/local-user',
            "ListModel response models[0].create_time": (r) => r.message.models[0].createTime !== undefined,
            "ListModel response models[0].update_time": (r) => r.message.models[0].updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};

export function LookupModel() {
    // LookUpModel check
    group("Model API: LookUpModel", () => {
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
            let res = client.invoke('vdp.model.v1alpha.ModelService/GetModelOperation', {
                name: createClsModelRes.json().operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        let res = client.invoke('vdp.model.v1alpha.ModelService/GetModelOperation', {
            name: createClsModelRes.json().operation.name
        }, {})
        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModel', {
            permalink: "models/" + res.message.operation.response.uid
        }, {}), {
            "LookUpModel response status": (r) => r.status === grpc.StatusOK,
            "LookUpModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "LookUpModel response model.uid": (r) => r.message.model.uid !== undefined,
            "LookUpModel response model.id": (r) => r.message.model.id === model_id,
            "LookUpModel response model.description": (r) => r.message.model.description === model_description,
            "LookUpModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "LookUpModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "LookUpModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "LookUpModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "LookUpModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "LookUpModel response model.update_time": (r) => r.message.model.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModel', {
            permalink: "models/" + randomString(10)
        }, {}), {
            'LookUpModel non-existed model status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
