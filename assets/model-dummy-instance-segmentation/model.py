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
class InstanceSegmentation:
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
                    name="boxes",
                    datatype=str(DataType.TYPE_FP32),
                    shape=[-1, 4],
                ),
                Metadata(
                    name="labels",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[-1],
                ),
                Metadata(
                    name="scores",
                    datatype=str(DataType.TYPE_FP32),
                    shape=[-1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        resp_outputs = []
        resp_raw_outputs = []
        for b_tensors in req.raw_input_contents:
            input_tensors = deserialize_bytes_tensor(b_tensors)

            # rles
            resp_outputs.append(
                Metadata(
                    name="rles",
                    shape=[len(input_tensors), 1],
                    datatype=str(DataType.TYPE_STRING),
                )
            )
            rles_out = [
                "2918,12,382,33,96,25,248,39,88,41,236,43,82,49,230,47,77,55,225,50,74,61,218,55,69,67,211,62,62,76,202,69,53,87,192,77,45,96,185,82,39,103,180,85,34,109,177,88,29,114,174,90,25,119,170,98,14,127,166,243,162,247,158,251,153,255,149,259,146,262,142,266,139,270,135,273,132,276,130,279,126,282,123,285,121,288,118,290,115,295,111,300,106,311,94,323,83,332,74,339,66,346,59,350,56,354,51,360,45,365,33,375,31,377,29,379,27,380,26,382,24,383,23,385,21,387,19,388,18,389,17,389,17,389,17,389,17,389,17,389,17,389,17,388,18,388,18,388,18,387,19,387,19,386,20,386,20,385,21,384,22,383,23,382,24,380,26,378,35,369,38,366,41,363,44,359,47,358,49,356,50,355,52,353,53,352,54,351,56,349,57,349,57,349,57,349,58,348,58,348,58,348,58,348,58,348,58,348,59,347,59,347,59,347,59,347,59,347,60,346,60,346,60,345,62,344,63,343,64,342,64,342,66,340,67,339,68,338,69,336,71,335,72,333,74,332,75,330,76,330,77,328,78,328,79,327,80,326,81,326,81,325,83,323,84,323,85,321,86,324,83,328,78,342,65,349,58,354,52,364,43,363,43,363,44,362,44,362,44,362,45,361,45,361,46,360,47,359,47,60,6,293,49,55,18,284,51,49,25,281,54,41,31,280,57,31,40,278,59,23,47,277,62,12,56,276,64,3,64,275,131,275,133,273,134,272,136,270,139,267,141,265,142,264,144,262,145,261,147,259,150,256,153,253,158,248,161,245,164,242,166,240,169,237,174,232,179,227,185,221,190,208,202,202,207,198,211,194,214,190,220,184,227,177,233,170,240,163,246,157,251,153,255,149,260,143,266,137,273,130,280,124,285,119,289,116,292,112,296,108,300,104,306,98,312,92,317,87,321,84,325,80,328,77,331,73,336,69,339,65,344,59,350,54,355,50,358,47,360,44,364,41,368,35,374,28,386,13,2525"
                for _ in range(len(input_tensors))
            ]
            rles_out = [bytes(f"{rles_out[i]}", "utf-8") for i in range(len(rles_out))]
            resp_raw_outputs.append(serialize_byte_tensor(np.asarray(rles_out)))

            # boxes
            resp_outputs.append(
                Metadata(
                    name="boxes",
                    shape=[len(input_tensors), 1, 4],
                    datatype=str(DataType.TYPE_FP32),
                )
            )
            boxes_out = [[[1, 1, 100, 100]] for _ in range(len(input_tensors))]
            resp_raw_outputs.append(np.asarray(boxes_out).astype(np.float32).tobytes())

            # labels
            resp_outputs.append(
                Metadata(
                    name="labels",
                    shape=[len(input_tensors), 1],
                    datatype=str(DataType.TYPE_STRING),
                )
            )
            labels = [["dog"] for _ in range(len(input_tensors))]
            labels_out = []
            for l in labels:
                labels_out.extend(l)
            labels_out = [
                bytes(f"{labels_out[i]}", "utf-8") for i in range(len(labels_out))
            ]
            resp_raw_outputs.append(serialize_byte_tensor(np.asarray(labels_out)))

            # scores
            resp_outputs.append(
                Metadata(
                    name="scores",
                    shape=[len(input_tensors), 1],
                    datatype=str(DataType.TYPE_FP32),
                )
            )
            scores_out = [[1.0] for _ in range(len(input_tensors))]
            resp_raw_outputs.append(np.asarray(scores_out).astype(np.float32).tobytes())

        return construct_infer_response(
            req=req,
            outputs=resp_outputs,
            raw_outputs=resp_raw_outputs,
        )


entrypoint = InstillDeployable(InstanceSegmentation).get_deployment_handle()
