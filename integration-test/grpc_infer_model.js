import grpc from 'k6/net/grpc';
import { check, group } from 'k6';
import http from "k6/http";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

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
        check(http.request("POST", `http://${constant.gRPCHost}/v1alpha/models/multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models/multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models/multipart (multipart) task cls response model.name": (r) =>
                r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/multipart (multipart) task cls response model.uid": (r) =>
                r.json().model.uid !== undefined,
            "POST /v1alpha/models/multipart (multipart) task cls response model.id": (r) =>
                r.json().model.id === model_id,
            "POST /v1alpha/models/multipart (multipart) task cls response model.description": (r) =>
                r.json().model.description === model_description,
            "POST /v1alpha/models/multipart (multipart) task cls response model.model_definition": (r) =>
                r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/multipart (multipart) task cls response model.configuration": (r) =>
                r.json().model.configuration !== undefined,
            "POST /v1alpha/models/multipart (multipart) task cls response model.visibility": (r) =>
                r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/multipart (multipart) task cls response model.owner": (r) =>
                r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/multipart (multipart) task cls response model.create_time": (r) =>
                r.json().model.create_time !== undefined,
            "POST /v1alpha/models/multipart (multipart) task cls response model.update_time": (r) =>
                r.json().model.update_time !== undefined,
        });

        let req = { name: `models/${model_id}/instances/latest` }
        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
            'DeployModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'DeployModelInstance instance id': (r) => r && r.message.instance.id === `latest`,
            'DeployModelInstance instance name': (r) => r && r.message.instance.name === `models/${model_id}/instances/latest`,
            'DeployModelInstance instance uid': (r) => r && r.message.instance.uid !== undefined,
            'DeployModelInstance instance state': (r) => r && r.message.instance.state === "STATE_ONLINE",
            'DeployModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'DeployModelInstance instance modelDefinition': (r) => r && r.message.instance.modelDefinition === model_def_name,
            'DeployModelInstance instance configuration': (r) => r && r.message.instance.configuration !== undefined,
            'DeployModelInstance instance createTime': (r) => r && r.message.instance.createTime !== undefined,
            'DeployModelInstance instance updateTime': (r) => r && r.message.instance.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }] }, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff" }] }, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
        });


        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/non-existed/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }] }, {}), {
            'TriggerModelInstance non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/non-existed`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }] }, {}), {
            'TriggerModelInstance non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/non-existed.jpg" }] }, {}), {
            'TriggerModelInstance non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
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
        check(http.request("POST", `http://${constant.gRPCHost}/v1alpha/models/multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models/multipart task cls response status": (r) =>
                r.status === 201,
            "POST /v1alpha/models/multipart (multipart) task cls response model.name": (r) =>
                r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/multipart (multipart) task cls response model.uid": (r) =>
                r.json().model.uid !== undefined,
            "POST /v1alpha/models/multipart (multipart) task cls response model.id": (r) =>
                r.json().model.id === model_id,
            "POST /v1alpha/models/multipart (multipart) task cls response model.description": (r) =>
                r.json().model.description === model_description,
            "POST /v1alpha/models/multipart (multipart) task cls response model.model_definition": (r) =>
                r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/multipart (multipart) task cls response model.configuration": (r) =>
                r.json().model.configuration !== undefined,
            "POST /v1alpha/models/multipart (multipart) task cls response model.visibility": (r) =>
                r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/multipart (multipart) task cls response model.owner": (r) =>
                r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/multipart (multipart) task cls response model.create_time": (r) =>
                r.json().model.create_time !== undefined,
            "POST /v1alpha/models/multipart (multipart) task cls response model.update_time": (r) =>
                r.json().model.update_time !== undefined,
        });

        let req = { name: `models/${model_id}/instances/latest` }
        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
            'DeployModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'DeployModelInstance instance id': (r) => r && r.message.instance.id === `latest`,
            'DeployModelInstance instance name': (r) => r && r.message.instance.name === `models/${model_id}/instances/latest`,
            'DeployModelInstance instance uid': (r) => r && r.message.instance.uid !== undefined,
            'DeployModelInstance instance state': (r) => r && r.message.instance.state === "STATE_ONLINE",
            'DeployModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'DeployModelInstance instance modelDefinition': (r) => r && r.message.instance.modelDefinition === model_def_name,
            'DeployModelInstance instance configuration': (r) => r && r.message.instance.configuration !== undefined,
            'DeployModelInstance instance createTime': (r) => r && r.message.instance.createTime !== undefined,
            'DeployModelInstance instance updateTime': (r) => r && r.message.instance.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }] }, {}), {
            'TestModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TestModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TestModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TestModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/tiff-sample.tiff" }] }, {}), {
            'TestModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TestModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TestModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category !== undefined,
            'TestModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', { name: `models/non-existed/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }] }, {}), {
            'TestModelInstance non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', { name: `models/${model_id}/instances/non-existed`, inputs: [{ image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }] }, {}), {
            'TestModelInstance non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/TestModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/non-existed.jpg" }] }, {}), {
            'TestModelInstance non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
