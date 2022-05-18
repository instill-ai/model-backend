import grpc from 'k6/net/grpc';
import { check, sleep, group } from 'k6';
import http from "k6/http";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";
import { randomString } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import { URL } from "https://jslib.k6.io/url/1.0.0/index.js";

import {
    genHeader,
    base64_image,
} from "./helpers.js";

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_service.proto');

const apiHost = "http://localhost:8083";
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const model_def_name = "model-definitions/local"


export function InferModel() {
    // TriggerModelInstance check
    group("Model API: TriggerModelInstance", () => {
        client.connect('localhost:8083', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/" + model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models:multipart`, fd_cls.body(), {
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
                r.json().model.user === 'users/2a06c2f7-8da9-4046-91ea-240f88a5d729',
            "POST /v1alpha/models:multipart (multipart) task cls response model.create_time": (r) =>
                r.json().model.create_time !== undefined,
            "POST /v1alpha/models:multipart (multipart) task cls response model.update_time": (r) =>
                r.json().model.update_time !== undefined,
        });

        let req = { name: `models/${model_id}/instances/latest` }
        check(client.invoke('instill.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
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
        sleep(5) // triton take time after update status

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/dog.jpg" }] }, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/non-existed/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/dog.jpg" }] }, {}), {
            'TriggerModelInstance non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/non-existed`, inputs: [{ image_url: "https://artifacts.instill.tech/dog.jpg" }] }, {}), {
            'TriggerModelInstance non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/latest`, inputs: [{ image_url: "https://artifacts.instill.tech/non-existed.jpg" }] }, {}), {
            'TriggerModelInstance non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};
