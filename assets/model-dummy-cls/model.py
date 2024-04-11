import numpy as np
from instill.helpers.const import DataType
from instill.helpers.ray_io import serialize_byte_tensor, deserialize_bytes_tensor
from instill.helpers.ray_config import instill_deployment, InstillDeployable
from instill.helpers import (
    construct_infer_response,
    construct_metadata_response,
    Metadata,
)


@instill_deployment
class MobileNet:
    def __init__(self):
        pass

    def ModelMetadata(self, req):
        resp = construct_metadata_response(
            req=req,
            inputs=[
                Metadata(
                    name="input",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
            ],
            outputs=[
                Metadata(
                    name="output",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        b_tensors = req.raw_input_contents[0]
        input_tensors = deserialize_bytes_tensor(b_tensors)

        out = [bytes("1:match", "utf-8") for _ in range(len(input_tensors))]

        out = serialize_byte_tensor(np.asarray(out))
        out = np.expand_dims(out, axis=0)

        return construct_infer_response(
            req=req,
            outputs=[
                Metadata(
                    name="output",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[len(input_tensors), 1],
                )
            ],
            raw_outputs=out,
        )


entrypoint = InstillDeployable(MobileNet).get_deployment_handle()
