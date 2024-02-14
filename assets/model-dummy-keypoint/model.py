import numpy as np
from instill.helpers.const import DataType
from instill.helpers.ray_io import deserialize_bytes_tensor
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
                    name="kpoints",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[-1, 17, 3],
                ),
                Metadata(
                    name="boxes",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[-1, 4],
                ),
                Metadata(
                    name="scores",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[-1, 1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        resp_outputs = []
        resp_raw_outputs = []
        for b_tensors in req.raw_input_contents:
            input_tensors = deserialize_bytes_tensor(b_tensors)

            kps = [[i, i, 1] for i in range(17)]

            kpoints = [[kps] for _ in range(len(input_tensors))]
            boxes = [[[1, 1, 1, 1]] for _ in range(len(input_tensors))]
            scores = [[1] for _ in range(len(input_tensors))]

            resp_outputs.append(
                Metadata(
                    name="kpoints",
                    shape=[len(input_tensors), len(kpoints[0]), 17, 3],
                    datatype=str(DataType.TYPE_FP32.name),
                )
            )

            resp_raw_outputs.append(np.asarray(kpoints).astype(np.float32).tobytes())

            resp_outputs.append(
                Metadata(
                    name="boxes",
                    shape=[len(input_tensors), len(boxes[0]), 4],
                    datatype=str(DataType.TYPE_FP32.name),
                )
            )

            resp_raw_outputs.append(np.asarray(boxes).astype(np.float32).tobytes())

            resp_outputs.append(
                Metadata(
                    name="scores",
                    shape=[len(input_tensors), 1],
                    datatype=str(DataType.TYPE_FP32.name),
                )
            )

            resp_raw_outputs.append(np.asarray(scores).astype(np.float32).tobytes())

        return construct_infer_response(
            req=req,
            outputs=resp_outputs,
            raw_outputs=resp_raw_outputs,
        )


deployable = InstillDeployable(Det, "/", False)
