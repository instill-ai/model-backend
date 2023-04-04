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


export function InferModel() {
    // TriggerModel check
    group("Model API: TriggerModel", () => {
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

        let req = {
            name: `models/${model_id}`
        }
        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeployModel', req, {}), {
            'DeployModel status': (r) => r && r.status === grpc.StatusOK,
            'DeployModel operation name': (r) => r && r.message.operation.name !== undefined,
            'DeployModel operation metadata': (r) => r && r.message.operation.metadata === null,
            'DeployModel operation done': (r) => r && r.message.operation.done === false,
        });

        // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.model.v1alpha.ModelPublicService/GetModel', {
                name: `models/${model_id}`
            }, {})
            if (res.message.model.state === "STATE_ONLINE") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }
        res = client.invoke('vdp.model.v1alpha.ModelPublicService/TriggerModel', {
            name: `models/${model_id}`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
            }]
        }, {})
        check(res, {
            'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TriggerModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TriggerModel', {
            name: `models/${model_id}`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff" }
            }]
        }, {}), {
            'TriggerModel status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
            'TriggerModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
        });


        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TriggerModel', {
            name: `models/non-existed`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
            }]
        }, {}), {
            'TriggerModel non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TriggerModel', {
            name: `models/${model_id}`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/non-existed.jpg" }
            }]
        }, {}), {
            'TriggerModel non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // TestModel check
    group("Model API: TestModel", () => {
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

        let req = {
            name: `models/${model_id}`
        }
        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeployModel', req, {}), {
            'DeployModel status': (r) => r && r.status === grpc.StatusOK,
            'DeployModel operation name': (r) => r && r.message.operation.name !== undefined,
            'DeployModel operation metadata': (r) => r && r.message.operation.metadata === null,
            'DeployModel operation done': (r) => r && r.message.operation.done === false,
        });

        // Check the model state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.model.v1alpha.ModelPublicService/GetModel', {
                name: `models/${model_id}`
            }, {})
            if (res.message.model.state === "STATE_ONLINE") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TestModel', {
            name: `models/${model_id}`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
            }]
        }, {}), {
            'TestModel status': (r) => r && r.status === grpc.StatusOK,
            'TestModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TestModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TestModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TestModel', {
            name: `models/${model_id}`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff" }
            }]
        }, {}), {
            'TestModel status': (r) => r && r.status === grpc.StatusOK,
            'TestModel output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TestModel output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
            'TestModel output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TestModel', {
            name: `models/non-existed`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
            }]
        }, {}), {
            'TestModel non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/TestModel', {
            name: `models/${model_id}`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/non-existed.jpg" }
            }]
        }, {}), {
            'TestModel non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelPublicService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
