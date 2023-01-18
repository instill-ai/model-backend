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


export function InferModel() {
    // TriggerModelInstance check
    group("Model API: TriggerModelInstance", () => {
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

        let req = {
            name: `models/${model_id}/instances/latest`
        }
        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
            'DeployModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'DeployModelInstance operation name': (r) => r && r.message.operation.name !== undefined,
            'DeployModelInstance operation metadata': (r) => r && r.message.operation.metadata === null,
            'DeployModelInstance operation done': (r) => r && r.message.operation.done === false,
        });

        // Check the model instance state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', {
                name: `models/${model_id}/instances/latest`
            }, {})
            if (res.message.instance.state === "STATE_ONLINE") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }
        res = client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', {
            name: `models/${model_id}/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/dog.jpg"}
            }]
        }, {})
        check(res, {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', {
            name: `models/${model_id}/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff"}
            }]
        }, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
        });


        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', {
            name: `models/non-existed/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/dog.jpg"}
            }]
        }, {}), {
            'TriggerModelInstance non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', {
            name: `models/${model_id}/instances/non-existed`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/dog.jpg"}
            }]
        }, {}), {
            'TriggerModelInstance non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', {
            name: `models/${model_id}/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/non-existed.jpg"}
            }]
        }, {}), {
            'TriggerModelInstance non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // TestModelInstance check
    group("Model API: TestModelInstance", () => {
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

        let req = {
            name: `models/${model_id}/instances/latest`
        }
        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
            'DeployModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'DeployModelInstance operation name': (r) => r && r.message.operation.name !== undefined,
            'DeployModelInstance operation metadata': (r) => r && r.message.operation.metadata === null,
            'DeployModelInstance operation done': (r) => r && r.message.operation.done === false,
        });

        // Check the model instance state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
        currentTime = new Date().getTime();
        timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', {
                name: `models/${model_id}/instances/latest`
            }, {})
            if (res.message.instance.state === "STATE_ONLINE") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', {
            name: `models/${model_id}/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/dog.jpg"}
            }]
        }, {}), {
            'TestModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TestModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TestModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TestModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', {
            name: `models/${model_id}/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff"}
            }]
        }, {}), {
            'TestModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TestModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TestModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
            'TestModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', {
            name: `models/non-existed/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/dog.jpg"}
            }]
        }, {}), {
            'TestModelInstance non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', {
            name: `models/${model_id}/instances/non-existed`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/imgs/dog.jpg"}
            }]
        }, {}), {
            'TestModelInstance non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', {
            name: `models/${model_id}/instances/latest`,
            task_inputs: [{
                classification: {image_url: "https://artifacts.instill.tech/non-existed.jpg"}
            }]
        }, {}), {
            'TestModelInstance non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
