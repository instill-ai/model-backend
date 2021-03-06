import grpc from 'k6/net/grpc';
import { check, group } from 'k6';
import http from "k6/http";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

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

export function GetModelInstance() {
    // GetModelInstance check
    group("Model API: GetModelInstance", () => {
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

        let req = { name: `models/${model_id}/instances/latest` }
        check(client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', req, {}), {
            'GetModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'GetModelInstance instance id': (r) => r && r.message.instance.id === `latest`,
            'GetModelInstance instance name': (r) => r && r.message.instance.name === `models/${model_id}/instances/latest`,
            'GetModelInstance instance uid': (r) => r && r.message.instance.uid !== undefined,
            'GetModelInstance instance state': (r) => r && r.message.instance.state === "STATE_OFFLINE",
            'GetModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'GetModelInstance instance modelDefinition': (r) => r && r.message.instance.modelDefinition === model_def_name,
            'GetModelInstance instance configuration': (r) => r && r.message.instance.configuration !== undefined,
            'GetModelInstance instance createTime': (r) => r && r.message.instance.createTime !== undefined,
            'GetModelInstance instance updateTime': (r) => r && r.message.instance.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', { name: `models/non-existed/instances/latest` }), {
            'UpdateModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', { name: `models/${model_id}/instances/non-existed` }, {}), {
            'UpdateModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // LookUpModelInstance check
    group("Model API: LookUpModelInstance", () => {
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
        let res_model = http.request("POST", `http://${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(res_model, {
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
        let res_model_instance = client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', { name: `models/${model_id}/instances/latest` }, {})
        check(res_model_instance, {
            'GetModelInstance status': (r) => r && r.status === grpc.StatusOK,
        });

        let req = { permalink: `models/${res_model.json().model.uid}/instances/${res_model_instance.message.instance.uid}` }
        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModelInstance', req, {}), {
            'LookUpModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'LookUpModelInstance instance id': (r) => r && r.message.instance.id === `latest`,
            'LookUpModelInstance instance name': (r) => r && r.message.instance.name === `models/${model_id}/instances/latest`,
            'LookUpModelInstance instance uid': (r) => r && r.message.instance.uid === res_model_instance.message.instance.uid,
            'LookUpModelInstance instance state': (r) => r && r.message.instance.state === "STATE_OFFLINE",
            'LookUpModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'LookUpModelInstance instance modelDefinition': (r) => r && r.message.instance.modelDefinition === model_def_name,
            'LookUpModelInstance instance configuration': (r) => r && r.message.instance.configuration !== undefined,
            'LookUpModelInstance instance createTime': (r) => r && r.message.instance.createTime !== undefined,
            'LookUpModelInstance instance updateTime': (r) => r && r.message.instance.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModelInstance', { permalink: `models/non-existed/instances/${res_model_instance.message.instance.uid}` }), {
            'LookUpModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModelInstance', { permalink: `models/${res_model.json().model.uid}}/instances/non-existed` }, {}), {
            'LookUpModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};

export function ListModelInstance() {
    // ListModelInstance check
    group("Model API: ListModelInstance", () => {
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
        let res_model = http.request("POST", `http://${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(res_model, {
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
        let res_model_instance = client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', { name: `models/${model_id}/instances/latest` }, {})
        check(res_model_instance, {
            'GetModelInstance status': (r) => r && r.status === grpc.StatusOK,
        });

        let req = { parent: `models/${res_model.json().model.id}` }
        check(client.invoke('vdp.model.v1alpha.ModelService/ListModelInstance', req, {}), {
            'ListModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'ListModelInstance instances[0] id': (r) => r && r.message.instances[0].id === `latest`,
            'ListModelInstance instances[0] name': (r) => r && r.message.instances[0].name === `models/${model_id}/instances/latest`,
            'ListModelInstance instances[0] uid': (r) => r && r.message.instances[0].uid === res_model_instance.message.instance.uid,
            'ListModelInstance instances[0] state': (r) => r && r.message.instances[0].state === "STATE_OFFLINE",
            'ListModelInstance instances[0] task': (r) => r && r.message.instances[0].task === "TASK_CLASSIFICATION",
            'ListModelInstance instances[0] modelDefinition': (r) => r && r.message.instances[0].modelDefinition === model_def_name,
            'ListModelInstance instances[0] configuration': (r) => r && r.message.instances[0].configuration !== undefined,
            'ListModelInstance instances[0] createTime': (r) => r && r.message.instances[0].createTime !== undefined,
            'ListModelInstance instances[0] updateTime': (r) => r && r.message.instances[0].updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/ListModelInstance', { parent: `models/non-existed` }), {
            'ListModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};

export function LookupModelInstance() {
    // LookUpModelInstance check
    group("Model API: LookUpModelInstance", () => {
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
        let res_model = http.request("POST", `http://${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(res_model, {
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
        let res_model_instance = client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', { name: `models/${model_id}/instances/latest` }, {})
        check(res_model_instance, {
            'GetModelInstance status': (r) => r && r.status === grpc.StatusOK,
        });

        let req = { permalink: `models/${res_model.json().model.uid}/instances/${res_model_instance.message.instance.uid}` }
        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModelInstance', req, {}), {
            'LookUpModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'LookUpModelInstance instance id': (r) => r && r.message.instance.id === `latest`,
            'LookUpModelInstance instance name': (r) => r && r.message.instance.name === `models/${model_id}/instances/latest`,
            'LookUpModelInstance instance uid': (r) => r && r.message.instance.uid === res_model_instance.message.instance.uid,
            'LookUpModelInstance instance state': (r) => r && r.message.instance.state === "STATE_OFFLINE",
            'LookUpModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'LookUpModelInstance instance modelDefinition': (r) => r && r.message.instance.modelDefinition === model_def_name,
            'LookUpModelInstance instance configuration': (r) => r && r.message.instance.configuration !== undefined,
            'LookUpModelInstance instance createTime': (r) => r && r.message.instance.createTime !== undefined,
            'LookUpModelInstance instance updateTime': (r) => r && r.message.instance.updateTime !== undefined,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModelInstance', { permalink: `models/non-existed/instances/${res_model_instance.message.instance.uid}` }), {
            'LookUpModelInstance non-existed model name status invalid uid': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/LookUpModelInstance', { permalink: `models/${res_model.json().model.uid}}/instances/non-existed` }, {}), {
            'LookUpModelInstance non-existed instance name invalid uid': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
