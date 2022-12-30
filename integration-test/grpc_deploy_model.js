import grpc from 'k6/net/grpc';
import { check, group, sleep } from 'k6';
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

export function DeployUndeployModel() {
    // Deploy ModelInstance check
    group("Model API: Deploy ModelInstance", () => {
        client.connect(constant.gRPCHost);

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("id", model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition", model_def_name);
        fd_cls.append("content", http.file(constant.cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${constant.apiHost}/v1alpha/models/multipart`, fd_cls.body(), {
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
            'DeployModelInstance operation name': (r) => r && r.message.operation.name !== undefined,
            'DeployModelInstance operation metadata': (r) => r && r.message.operation.metadata === null,
            'DeployModelInstance operation done': (r) => r && r.message.operation.done === false,
        });

        // Check the model instance state being updated in 120 secs (in integration test, model is dummy model without download time but in real use case, time will be longer)
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            var res = client.invoke('vdp.model.v1alpha.ModelService/GetModelInstance', { name: `models/${model_id}/instances/latest` }, {})
            if (res.message.instance.state === "STATE_ONLINE") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', { name: `models/non-existed/instances/latest` }), {
            'DeployModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', { name: `models/${model_id}/instances/non-existed` }, {}), {
            'DeployModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
