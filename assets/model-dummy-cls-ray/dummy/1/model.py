from typing import List

import ray
import numpy as np
from instill.configuration import CORE_RAY_ADDRESS
from instill.helpers.ray_helper import (
    InstillRayModelConfig,
    DataType,
    serialize_byte_tensor,
    deserialize_bytes_tensor,
    entry,
)

from ray_pb2 import (
    ModelReadyRequest,
    ModelReadyResponse,
    ModelMetadataRequest,
    ModelMetadataResponse,
    ModelInferRequest,
    ModelInferResponse,
    InferTensor,
)

ray.init(address=CORE_RAY_ADDRESS)
# this import must come after `ray.init()`
from ray import serve


@serve.deployment()
class MobileNet:
    def __init__(self, model_path: str):
        self.application_name = "_".join(model_path.split("/")[3:5])
        self.deployement_name = model_path.split("/")[4]

    def ModelMetadata(self, req: ModelMetadataRequest) -> ModelMetadataResponse:
        resp = ModelMetadataResponse(
            name=req.name,
            versions=req.version,
            framework="python",
            inputs=[
                ModelMetadataResponse.TensorMetadata(
                    name="input",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
            ],
            outputs=[
                ModelMetadataResponse.TensorMetadata(
                    name="output",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[1],
                ),
            ],
        )
        return resp

    def ModelReady(self, req: ModelReadyRequest) -> ModelReadyResponse:
        resp = ModelReadyResponse(ready=True)
        return resp

    async def ModelInfer(self, request: ModelInferRequest) -> ModelInferResponse:
        b_tensors = request.raw_input_contents[0]

        input_tensors = deserialize_bytes_tensor(b_tensors)

        out = serialize_byte_tensor(np.asarray([bytes("1:match", "utf-8")]))
        out = np.expand_dims(out, axis=0)

        return ModelInferResponse(
            model_name=request.model_name,
            model_version=request.model_version,
            outputs=[
                InferTensor(
                    name="output",
                    shape=[len(input_tensors), 1],
                ),
            ],
            raw_output_contents=out,
        )


def deploy_model(model_config: InstillRayModelConfig):
    c_app = MobileNet.options(
        name=model_config.application_name,
        ray_actor_options=model_config.ray_actor_options,
        max_concurrent_queries=model_config.max_concurrent_queries,
        autoscaling_config=model_config.ray_autoscaling_options,
    ).bind(model_config.model_path)

    serve.run(
        c_app, name=model_config.model_name, route_prefix=model_config.route_prefix
    )


def undeploy_model(model_name: str):
    serve.delete(model_name)


if __name__ == "__main__":
    func, model_config = entry()

    if func == "deploy":
        deploy_model(model_config=model_config)
    elif func == "undeploy":
        undeploy_model(model_name=model_config.model_name)
