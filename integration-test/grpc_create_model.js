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
const model_def_name = "model-definitions/github"

export function CreateModel() {
    // CreateModelBinaryFileUpload check
    group("Model API: CreateModelBinaryFileUpload", () => {
        client.connect('localhost:8083', {
            plaintext: true
        });
        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModelBinaryFileUpload', {}), {
            'Missing stream body status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        client.close();
    });


    // CreateModel check
    group("Model API: CreateModel with GitHub", () => {
        client.connect('localhost:8083', {
            plaintext: true
        });
        let model_id = randomString(10)
        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: model_id,
                model_definition: model_def_name,
                configuration: JSON.stringify( {
                    repository: "instill-ai/model-dummy-cls"
                })
            }
        }), {
            'CreateModel status': (r) => r && r.status === grpc.StatusOK,
            'CreateModel model name': (r) => r && r.message.model.name === "models/" + model_id,
            'CreateModel model id': (r) => r && r.message.model.id === model_id,
            'CreateModel model uid': (r) => r && r.message.model.uid !== undefined,
            'CreateModel model description': (r) => r && r.message.model.description !== undefined,
            'CreateModel model visibility': (r) => r && r.message.model.visibility === "VISIBILITY_PUBLIC",
            'CreateModel model createTime': (r) => r && r.message.model.createTime !== undefined,
            'CreateModel model updateTime': (r) => r && r.message.model.updateTime !== undefined,
            'CreateModel model configuration repository': (r) => r && JSON.parse(r.message.model.configuration).repository === "instill-ai/model-dummy-cls",
            'CreateModel model user': (r) => r && r.message.model.user !== undefined,
        });

        let req = { name: `models/${model_id}/instances/v1.0` }
        check(client.invoke('vdp.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
            'DeployModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'DeployModelInstance instance id': (r) => r && r.message.instance.id === `v1.0`,
            'DeployModelInstance instance name': (r) => r && r.message.instance.name === `models/${model_id}/instances/v1.0`,
            'DeployModelInstance instance uid': (r) => r && r.message.instance.uid !== undefined,
            'DeployModelInstance instance state': (r) => r && r.message.instance.state === "STATE_ONLINE",
            'DeployModelInstance instance task': (r) => r && r.message.instance.task === "TASK_CLASSIFICATION",
            'DeployModelInstance instance modelDefinition': (r) => r && r.message.instance.modelDefinition === model_def_name,
            'DeployModelInstance instance configuration': (r) => r && r.message.instance.configuration !== undefined,
            'DeployModelInstance instance createTime': (r) => r && r.message.instance.createTime !== undefined,
            'DeployModelInstance instance updateTime': (r) => r && r.message.instance.updateTime !== undefined,
        });
        sleep(5) // triton take time after update status

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', { name: `models/${model_id}/instances/v1.0`, inputs: [{ image_url: "https://artifacts.instill.tech/dog.jpg" }] }, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: randomString(10),
                configuration: JSON.stringify({
                    repository: "instill-ai/model-dummy-cls"
                })
            }
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: model_def_name,
                configuration: JSON.stringify({
                    repository: "invalid-repo"
                })
            }
        }), {
            'invalid github repo status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                model_definition: model_def_name,
                configuration: JSON.stringify({
                    repository: "instill-ai/model-dummy-cls"
                })
            }
        }), {
            'missing name status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: model_def_name,
            }
        }), {
            'missing github url status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', { name: "models/" + model_id }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });

        client.close();
    });
};
