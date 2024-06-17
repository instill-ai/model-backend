import grpc from 'k6/net/grpc';
import {
  check,
  group,
} from 'k6';
import {
  randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

const client = new grpc.Client();
client.load(['proto/model/model/v1alpha'], 'model_definition.proto');
client.load(['proto/model/model/v1alpha'], 'model.proto');
client.load(['proto/model/model/v1alpha'], 'model_public_service.proto');

import * as constant from "./const.js"

export function CreateUserModel(header) {
  // CreateModel check
  group("Model API: CreateUserModel", () => {
    client.connect(constant.gRPCPublicHost, {
      plaintext: true
    });
    let model_id = randomString(10)
    let createRes = client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        id: model_id,
        modelDefinition: constant.model_def_name,
        visibility: "VISIBILITY_PUBLIC",
        region: "REGION_GCP_EUROPE_WEST_4",
        hardware: "CPU",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
      parent: constant.namespace,
    }, header)
    check(createRes, {
      'CreateUserModel status': (r) => r && r.status === grpc.StatusOK,
      'CreateUserModel model': (r) => r && r.message.model !== undefined,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/GetUserModel', {
      name: `${constant.namespace}/models/${model_id}`,
    }, header), {
      'GetUserModel status': (r) => r && r.status === grpc.StatusOK,
      'GetUserModel output model id': (r) => r && r.message.model.id === model_id,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        id: randomString(10),
        modelDefinition: randomString(10),
        visibility: "VISIBILITY_PUBLIC",
        region: "REGION_GCP_EUROPE_WEST_4",
        hardware: "CPU",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
      parent: constant.namespace,
    }, header), {
      'status': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        modelDefinition: constant.model_def_name,
        visibility: "VISIBILITY_PUBLIC",
        region: "REGION_GCP_EUROPE_WEST_4",
        hardware: "CPU",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
      parent: constant.namespace,
    }, header), {
      'missing name status': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        id: randomString(10),
        modelDefinition: constant.model_def_name,
        visibility: "VISIBILITY_PUBLIC",
        region: "REGION_GCP_EUROPE_WEST_4",
        hardware: "CPU",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
    }, header), {
      'missing namespace': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        id: randomString(10),
        modelDefinition: constant.model_def_name,
        visibility: "VISIBILITY_PUBLIC",
        region: "REGION_GCP_EUROPE_WEST_4",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
      parent: constant.namespace,
    }, header), {
      'missing hardware': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/CreateUserModel', {
      model: {
        id: randomString(10),
        modelDefinition: constant.model_def_name,
        visibility: "VISIBILITY_PUBLIC",
        hardware: "CPU",
        configuration: {
          task: "CLASSIFICATION"
        }
      },
      parent: constant.namespace,
    }, header), {
      'missing region': (r) => r && r.status == grpc.StatusInvalidArgument,
    });

    check(client.invoke('model.model.v1alpha.ModelPublicService/DeleteUserModel', {
      name: `${constant.namespace}/models/${model_id}`
    }, header), {
      'DeleteModel model status is OK': (r) => r && r.status === grpc.StatusOK,
    });

    client.close();
  });
};
