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
client.load(['proto/vdp/model/v1alpha'], 'model_definition.proto');
client.load(['proto/vdp/model/v1alpha'], 'model.proto');
client.load(['proto/vdp/model/v1alpha'], 'model_public_service.proto');

const model_def_name = "model-definitions/local"

export function ListModelOperations() {
    // GetModel check
    group("Model API: ListModelOperations", () => {
        client.connect(constant.gRPCPublicHost, {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
        let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(createClsModelRes, {
            "POST /v1alpha/models/multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models/multipart task cls response operation.name": (r) =>
                r.json().operation.name !== undefined,
        });

        sleep(1)

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/ListModelOperations', {}, {}), {
            "ListModelOperations response status": (r) => r.status === grpc.StatusOK,
            "ListModelOperations response totalSize": (r) => r.message.totalSize >= 1,
            "ListModelOperations response operations.length": (r) => r.message.operations.length > 0,
            "ListModelOperations response operations[0].name": (r) => r.message.operations[0].name != undefined,
            "ListModelOperations response operations[0].done": (r) => r.message.operations[0].done != undefined,
            "ListModelOperations response operations[0].response": (r) => r.message.operations[0].response != undefined,
            "ListModelOperations response operations[0].metadata": (r) => r.message.operations[0].metadata === null,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};


export function CancelModelOperation() {
    // ListModel check
    group("Model API: CancelModelOperation", () => {
        client.connect(constant.gRPCPublicHost, {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
        let createClsModelRes = http.request("POST", `${constant.apiPublicHost}/v1alpha/models/multipart`, fd_cls.body(), {
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

        let deployResp = client.invoke('vdp.model.v1alpha.ModelPublicService/DeployModel', {
            name: `models/${model_id}/instances/latest`
        }, {})
        check(deployResp, {
            'DeployModel status': (r) => r && r.status === grpc.StatusOK,
        });

        sleep(0.1) // make sure the deploy operation is started
        check(client.invoke('vdp.model.v1alpha.ModelPublicService/CancelModelOperation', {
            name: deployResp.message.operation.name
        }), {
            'CancelModelOperation status is OK': (r) => r && r.status === grpc.StatusOK,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};