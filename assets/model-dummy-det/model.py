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
class Det:
    def __init__(self, _):
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
                    name="output_bboxes",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[-1, -1, 5],
                ),
                Metadata(
                    name="output_labels",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[-1, -1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        b_tensors = req.raw_input_contents[0]
        input_tensors = deserialize_bytes_tensor(b_tensors)

        bboxes = [[[0, 0, 0, 0, 1.0]] for _ in range(len(input_tensors))]
        labels = [["test"] for _ in range(len(input_tensors))]

        labels_out = []
        for l in labels:
            labels_out.extend(l)

        labels_out = [bytes(f"{labels_out[i]}", "utf-8") for i in range(len(labels_out))]

        return construct_infer_response(
            req=req,
            outputs=[
                Metadata(
                    name="output_bboxes",
                    shape=[len(input_tensors), len(bboxes[0]), 5],
                    datatype=str(DataType.TYPE_FP32.name),
                ),
                Metadata(
                    name="output_labels",
                    shape=[len(input_tensors), len(labels[0])],
                    datatype=str(DataType.TYPE_STRING),
                ),
            ],
            raw_outputs=[
                np.asarray(bboxes).astype(np.float32).tobytes(),
                serialize_byte_tensor(np.asarray(labels_out)),
            ],
        )


entrypoint = InstillDeployable(Det).get_deployment_handle()
