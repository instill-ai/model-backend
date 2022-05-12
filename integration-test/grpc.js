import grpc from 'k6/net/grpc';
import { check, sleep, group } from 'k6';
import http from "k6/http";
import {FormData} from "https://jslib.k6.io/formdata/0.0.2/index.js";
import {randomString} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";
import {URL} from "https://jslib.k6.io/url/1.0.0/index.js";

import {
    genHeader,
    base64_image,
} from "./helpers.js";

const client = new grpc.Client();
client.load(['proto'], 'model_definition.proto');
client.load(['proto'], 'model.proto');
client.load(['proto'], 'model_service.proto');

const apiHost = "http://localhost:8080";
const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
const model_def_name = "model-definitions/github"

export function setup() {
}

export default () => {
    // Liveness check
    {
        group("Model API: Liveness", () => {
            client.connect('localhost:8080', {
                plaintext: true
            });
            const response = client.invoke('instill.model.v1alpha.ModelService/Liveness', {});
            check(response, {
                'Status is OK': (r) => r && r.status === grpc.StatusOK,
                'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
            });
        });
    }

    // Readiness check
    group("Model API: Readiness", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        const response = client.invoke('instill.model.v1alpha.ModelService/Readiness', {});
        check(response, {
            'Status is OK': (r) => r && r.status === grpc.StatusOK,
            'Response status is SERVING_STATUS_SERVING': (r) => r && r.message.healthCheckResponse.status === "SERVING_STATUS_SERVING",
        });
        client.close();
    });

    // CreateModelBinaryFileUpload check
    group("Model API: CreateModelBinaryFileUpload", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        check(client.invoke('instill.model.v1alpha.ModelService/CreateModelBinaryFileUpload', {}), {
            'Missing stream body status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        client.close();
    });

    // ListModel check
    group("Model API: ListModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });
        check(client.invoke('instill.model.v1alpha.ModelService/ListModel', {}, {}), {
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

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // GetModel check
    group("Model API: GetModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: "models/"+model_id}, {}), {
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

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: "models/"+randomString(10)}, {}), {
            'GetModel non-existed model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // Deploy ModelInstance check
    group("Model API: Deploy ModelInstance", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        let req = {name: `models/${model_id}/instances/latest`}
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

        check(client.invoke('instill.model.v1alpha.ModelService/DeployModelInstance', {name: `models/non-existed/instances/latest`}), {
            'DeployModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeployModelInstance', {name: `models/${model_id}/instances/non-existed`}, {}), {
            'DeployModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // DeleteModel check
    group("Model API: DeleteModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: randomString(10)}, {}), {
            'DeleteModel model status invalid': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+randomString(10)}, {}), {
            'DeleteModel non-exist model status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: "models/"+model_id}, {}), {
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

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}, {}), {
            'DeleteModel status OK': (r) => r && r.status === grpc.StatusOK,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModel', {name: "models/"+model_id}, {}), {
            'GetModel after delete version status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusNotFound,
        });
        client.close();
    });

    // TriggerModelInstance check
    group("Model API: TriggerModelInstance", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        let req = {name: `models/${model_id}/instances/latest`}
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

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', {name: `models/${model_id}/instances/latest`, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });


        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', {name: `models/non-existed/instances/latest`, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModelInstance non-existed model name status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', {name: `models/${model_id}/instances/non-existed`, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModelInstance non-existed model version  status': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', {name: `models/${model_id}/instances/latest`, inputs: [{image_url: "https://artifacts.instill.tech/non-existed.jpg"}]}, {}), {
            'TriggerModelInstance non-existed model url status': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // CreateModel check
    group("Model API: CreateModel with GitHub", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });
        let model_id = randomString(10)
        check(client.invoke('instill.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: model_id,
                model_definition: model_def_name,
                configuration: {
                    repo: "https://github.com/Phelan164/test-repo.git",
                    tag: "v1.0",
                    html_url: ""
                }
            }
        }), {
            'CreateModel status': (r) => r && r.status === grpc.StatusOK,
            'CreateModel model name': (r) => r && r.message.model.name === "models/"+model_id,
            'CreateModel model id': (r) => r && r.message.model.id === model_id,
            'CreateModel model uid': (r) => r && r.message.model.uid !== undefined,
            'CreateModel model description': (r) => r && r.message.model.description !== undefined,
            'CreateModel model visibility': (r) => r && r.message.model.visibility === "VISIBILITY_PUBLIC",
            'CreateModel model createTime': (r) => r && r.message.model.createTime !== undefined,
            'CreateModel model updateTime': (r) => r && r.message.model.updateTime !== undefined,
            'CreateModel model configuration repo': (r) => r && r.message.model.configuration.repo === "https://github.com/Phelan164/test-repo.git",
            'CreateModel model user': (r) => r && r.message.model.user !== undefined,
        });

        let req = {name: `models/${model_id}/instances/v1.0`}
        check(client.invoke('instill.model.v1alpha.ModelService/DeployModelInstance', req, {}), {
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

        check(client.invoke('instill.model.v1alpha.ModelService/TriggerModelInstance', {name: `models/${model_id}/instances/v1.0`, inputs: [{image_url: "https://artifacts.instill.tech/dog.jpg"}]}, {}), {
            'TriggerModelInstance status': (r) => r && r.status === grpc.StatusOK,
            'TriggerModelInstance output classification_outputs length': (r) => r && r.message.output.classification_outputs.length === 1,
            'TriggerModelInstance output classification_outputs category': (r) => r && r.message.output.classification_outputs[0].category === "match",
            'TriggerModelInstance output classification_outputs score': (r) => r && r.message.output.classification_outputs[0].score === 1,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: model_def_name,
                configuration: {
                    repo: "https://github.com/Phelan164/test-repo.git",
                    tag: "non-existed",
                    html_url: ""
                }
            }
        }), {
            'status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: model_def_name,
                configuration: {
                    repo: "invalid-repo",
                    tag: "v1.0",
                    html_url: ""
                }
            }
        }), {
            'invalid github repo status': (r) => r && r.status == grpc.StatusFailedPrecondition,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModel', {
            model: {
                model_definition: model_def_name,
                configuration: {
                    repo: "https://github.com/Phelan164/test-repo.git",
                    tag: "v1.0",
                    html_url: ""
                }
            }
        }), {
            'missing name status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/CreateModel', {
            model: {
                id: randomString(10),
                model_definition: model_def_name,
            }
        }), {
            'missing github url status': (r) => r && r.status == grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
        });

        client.close();
    });

    // LookUpModel check
    group("Model API: LookUpModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        let res = http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(res, {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/LookUpModel', {permalink: "models/"+res.json().model.uid}, {}), {
            "LookUpModel response status": (r) => r.status === grpc.StatusOK, 
            "LookUpModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "LookUpModel response model.uid": (r) => r.message.model.uid === res.json().model.uid,
            "LookUpModel response model.id": (r) => r.message.model.id === model_id,          
            "LookUpModel response model.description": (r) => r.message.model.description === model_description,
            "LookUpModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "LookUpModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "LookUpModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "LookUpModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "LookUpModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "LookUpModel response model.update_time": (r) => r.message.model.updateTime !== undefined,    
        });

        check(client.invoke('instill.model.v1alpha.ModelService/LookUpModel', {permalink: "models/"+randomString(10)}, {}), {
            'LookUpModel non-existed model status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // GetModelInstance check
    group("Model API: GetModelInstance", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        let req = {name: `models/${model_id}/instances/latest`}
        check(client.invoke('instill.model.v1alpha.ModelService/GetModelInstance', req, {}), {
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
        sleep(5) // triton take time after update status

        check(client.invoke('instill.model.v1alpha.ModelService/GetModelInstance', {name: `models/non-existed/instances/latest`}), {
            'UpdateModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/GetModelInstance', {name: `models/${model_id}/instances/non-existed`}, {}), {
            'UpdateModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusNotFound,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });    

    // LookUpModelInstance check
    group("Model API: LookUpModelInstance", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        let res_model = http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        })
        check(res_model, {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });
        let res_model_instance = client.invoke('instill.model.v1alpha.ModelService/GetModelInstance', {name: `models/${model_id}/instances/latest`}, {})
        check(res_model_instance, {
            'GetModelInstance status': (r) => r && r.status === grpc.StatusOK,
        });    

        let req = {permalink: `models/${res_model.json().model.uid}/instances/${res_model_instance.message.instance.uid}`}
        check(client.invoke('instill.model.v1alpha.ModelService/LookUpModelInstance', req, {}), {
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

        check(client.invoke('instill.model.v1alpha.ModelService/LookUpModelInstance', {permalink: `models/non-existed/instances/${res_model_instance.message.instance.uid}`}), {
            'LookUpModelInstance non-existed model name status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/LookUpModelInstance', {permalink: `models/${res_model.json().model.uid}}/instances/non-existed`}, {}), {
            'LookUpModelInstance non-existed instance name status not found': (r) => r && r.status === grpc.StatusInvalidArgument,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });     

    // UpdateModel check
    group("Model API: UpdateModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        let res = client.invoke('instill.model.v1alpha.ModelService/UpdateModel', {
            model: {
                name: "models/"+model_id,
                description: "new_description"
            },
            update_mask: "description"
        })
        check(res, {
            "UpdateModel response status": (r) => r.status === grpc.StatusOK, 
            "UpdateModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "UpdateModel response model.uid": (r) => r.message.model.uid !== undefined,
            "UpdateModel response model.id": (r) => r.message.model.id === model_id,          
            "UpdateModel response model.description": (r) => r.message.model.description === "new_description",
            "UpdateModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "UpdateModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "UpdateModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "UpdateModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "UpdateModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "UpdateModel response model.update_time": (r) => r.message.model.updateTime !== undefined,    
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });

    // PublishModel/UnpublishModel check
    group("Model API: PublishModel/UnpublishModel", () => {
        client.connect('localhost:8080', {
            plaintext: true
        });

        let fd_cls = new FormData();
        let model_id = randomString(10)
        let model_description = randomString(20)
        fd_cls.append("name", "models/"+model_id);
        fd_cls.append("description", model_description);
        fd_cls.append("model_definition_name", model_def_name);
        fd_cls.append("content", http.file(cls_model, "dummy-cls-model.zip"));
        check(http.request("POST", `${apiHost}/v1alpha/models/upload`, fd_cls.body(), {
            headers: genHeader(`multipart/form-data; boundary=${fd_cls.boundary}`),
        }), {
            "POST /v1alpha/models github task cls response status": (r) =>
            r.status === 201, 
            "POST /v1alpha/models/upload (multipart) task cls response model.name": (r) =>
            r.json().model.name === `models/${model_id}`,
            "POST /v1alpha/models/upload (multipart) task cls response model.uid": (r) =>
            r.json().model.uid !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.id": (r) =>
            r.json().model.id === model_id,          
            "POST /v1alpha/models/upload (multipart) task cls response model.description": (r) =>
            r.json().model.description === model_description,
            "POST /v1alpha/models/upload (multipart) task cls response model.model_definition": (r) =>
            r.json().model.model_definition === model_def_name,
            "POST /v1alpha/models/upload (multipart) task cls response model.configuration": (r) =>
            r.json().model.configuration !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.visibility": (r) =>
            r.json().model.visibility === "VISIBILITY_PRIVATE",
            "POST /v1alpha/models/upload (multipart) task cls response model.owner": (r) =>
            r.json().model.user === 'users/local-user',
            "POST /v1alpha/models/upload (multipart) task cls response model.create_time": (r) =>
            r.json().model.create_time !== undefined,
            "POST /v1alpha/models/upload (multipart) task cls response model.update_time": (r) =>
            r.json().model.update_time !== undefined,
        });

        check(client.invoke('instill.model.v1alpha.ModelService/PublishModel', {name: "models/"+model_id}), {
            "PublishModel response status": (r) => r.status === grpc.StatusOK, 
            "PublishModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "PublishModel response model.uid": (r) => r.message.model.uid !== undefined,
            "PublishModel response model.id": (r) => r.message.model.id === model_id,          
            "PublishModel response model.description": (r) => r.message.model.description === model_description,
            "PublishModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "PublishModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "PublishModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PUBLIC",
            "PublishModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "PublishModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "PublishModel response model.update_time": (r) => r.message.model.updateTime !== undefined,    
        });

        check(client.invoke('instill.model.v1alpha.ModelService/UnpublishModel', {name: "models/"+model_id}), {
            "UnpublishModel response status": (r) => r.status === grpc.StatusOK, 
            "UnpublishModel response model.name": (r) => r.message.model.name === `models/${model_id}`,
            "UnpublishModel response model.uid": (r) => r.message.model.uid !== undefined,
            "UnpublishModel response model.id": (r) => r.message.model.id === model_id,          
            "UnpublishModel response model.description": (r) => r.message.model.description === model_description,
            "UnpublishModel response model.model_definition": (r) => r.message.model.modelDefinition === model_def_name,
            "UnpublishModel response model.configuration": (r) => r.message.model.configuration !== undefined,
            "UnpublishModel response model.visibility": (r) => r.message.model.visibility === "VISIBILITY_PRIVATE",
            "UnpublishModel response model.owner": (r) => r.message.model.user === 'users/local-user',
            "UnpublishModel response model.create_time": (r) => r.message.model.createTime !== undefined,
            "UnpublishModel response model.update_time": (r) => r.message.model.updateTime !== undefined,    
        });       
        
        check(client.invoke('instill.model.v1alpha.ModelService/PublishModel', {name: "models/"+randomString(10)}), {
            "PublishModel response not found status": (r) => r.status === grpc.StatusNotFound, 
        });

        check(client.invoke('instill.model.v1alpha.ModelService/UnpublishModel', {name: "models/"+randomString(10)}), {
            "UnpublishModel response not found status": (r) => r.status === grpc.StatusNotFound, 
        });

        check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: "models/"+model_id}), {
            'Delete model status is OK': (r) => r && r.status === grpc.StatusOK,
        });
        client.close();
    });
};

export function teardown() {
    client.connect('localhost:8080', {
        plaintext: true
    });
    group("Model API: Delete all models created by this test", () => {
        for (const model of client.invoke('instill.model.v1alpha.ModelService/ListModel', {}, {}).message.models) {
            check(client.invoke('instill.model.v1alpha.ModelService/DeleteModel', {name: model.name}), {
                'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
            });
        }
    });
    client.close();
}
