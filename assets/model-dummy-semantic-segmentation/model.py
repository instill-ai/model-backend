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
class SemanticSegmentation:
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
                    name="rles",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[-1],
                ),
                Metadata(
                    name="labels",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[-1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        resp_outputs = []
        resp_raw_outputs = []
        b_tensors = req.raw_input_contents[0]
        input_tensors = deserialize_bytes_tensor(b_tensors)

        # rles
        rles_out = [
            ["376,7,505,7,505,7,505,7,505,7,505,7,505,7,520833"]
            for _ in range(len(input_tensors))
        ]

        resp_outputs.append(
            Metadata(
                name="rles",
                shape=[len(input_tensors), 1],
                datatype=str(DataType.TYPE_STRING),
            )
        )

        rles_out = [bytes(f"{rles_out[i]}", "utf-8") for i in range(len(rles_out))]
        resp_raw_outputs.append(serialize_byte_tensor(np.asarray(rles_out)))

        # labels
        labels = [["tree"] for _ in range(len(input_tensors))]

        labels_out = []
        for l in labels:
            labels_out.extend(l)

        resp_outputs.append(
            Metadata(
                name="labels",
                shape=[len(input_tensors), 1],
                datatype=str(DataType.TYPE_STRING),
            )
        )

        labels_out = [
            bytes(f"{labels_out[i]}", "utf-8") for i in range(len(labels_out))
        ]
        resp_raw_outputs.append(serialize_byte_tensor(np.asarray(labels_out)))

        return construct_infer_response(
            req=req,
            outputs=resp_outputs,
            raw_outputs=resp_raw_outputs,
        )


entrypoint = InstillDeployable(SemanticSegmentation).get_deployment_handle()
