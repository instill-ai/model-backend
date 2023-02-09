import grpc from 'k6/net/grpc';
import {
    check,
    group,
    sleep
} from 'k6';
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_service.proto');

import * as constant from "./const.js"

const model_def_name = "model-definitions/github"

export function CreateModel() {
    // CreateModelBinaryFileUpload check
    group("Model API: CreateModelBinaryFileUpload", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });
        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModelBinaryFileUpload', {}), {
            'Missing stream body status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        client.close();
    });


    // CreateModel check
    group("Model API: CreateModel with GitHub", () => {
        client.connect(constant.gRPCHost, {
            plaintext: true
        });
        let model_id = randomString(10)
        let createOperationRes = client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: model_id,
                model_definition: model_def_name,
                configuration: {
                    repository: "instill-ai/model-dummy-cls"
                }
            }
        })
        check(createOperationRes, {
            'CreateModel status': (r) => r && r.status === grpc.StatusOK,
            'CreateModel operation name': (r) => r && r.message.operation.name !== undefined,
        });

        // Check model creation finished
        let currentTime = new Date().getTime();
        let timeoutTime = new Date().getTime() + 120000;
        while (timeoutTime > currentTime) {
            let res = client.invoke('vdp.model.v1alpha.ModelService/GetModelOperation', {
                name: createOperationRes.message.operation.name
            }, {})
            if (res.message.operation.done === true) {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        let req = {
            name: `models/${model_id}/instances/v1.0-cpu`
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
                name: `models/${model_id}/instances/v1.0-cpu`
            }, {})
            if (res.message.instance.state === "STATE_ONLINE") {
                break
            }
            sleep(1)
            currentTime = new Date().getTime();
        }

        check(client.invoke('vdp.model.v1alpha.ModelService/TriggerModelInstance', {
            name: `models/${model_id}/instances/v1.0-cpu`,
            task_inputs: [{
                classification: { image_url: "https://artifacts.instill.tech/imgs/dog.jpg" }
            }]
        }, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.taskOutputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.taskOutputs[0].classification.category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.taskOutputs[0].classification.score === 1,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: randomString(10),
                configuration: {
                    repository: "instill-ai/model-dummy-cls"
                }
            }
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('vdp.model.v1alpha.ModelService/CreateModel', {
            model: {
                model_definition: model_def_name,
                configuration: {
                    repository: "instill-ai/model-dummy-cls"
                }
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

        check(client.invoke('vdp.model.v1alpha.ModelService/DeleteModel', {
            name: "models/" + model_id
        }), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });

        client.close();
    });
};