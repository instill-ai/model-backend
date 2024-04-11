from instill.helpers.const import DataType
from instill.helpers.ray_config import instill_deployment, InstillDeployable
from instill.helpers import (
    construct_infer_response,
    construct_metadata_response,
    Metadata,
)

import torch


@instill_deployment
class TextToImage:
    def __init__(self):
        pass

    def ModelMetadata(self, req):
        resp = construct_metadata_response(
            req=req,
            inputs=[
                Metadata(
                    name="prompt",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
                Metadata(
                    name="negative_prompt",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
                Metadata(
                    name="prompt_image",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
                Metadata(
                    name="samples",
                    datatype=str(DataType.TYPE_INT32.name),
                    shape=[1],
                ),
                Metadata(
                    name="scheduler",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
                Metadata(
                    name="steps",
                    datatype=str(DataType.TYPE_INT32.name),
                    shape=[1],
                ),
                Metadata(
                    name="guidance_scale",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[1],
                ),
                Metadata(
                    name="seed",
                    datatype=str(DataType.TYPE_INT64.name),
                    shape=[1],
                ),
                Metadata(
                    name="extra_params",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
            ],
            outputs=[
                Metadata(
                    name="images",
                    datatype=str(DataType.TYPE_FP32.name),
                    shape=[-1, -1, -1, -1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        resp_outputs = []
        resp_raw_outputs = []
        for _ in req.raw_input_contents:
            generator = torch.Generator(device="cpu").manual_seed(0)
            image: torch.Tensor = torch.randn(
                (1, 3, 5, 5), generator=generator, device="cpu"
            )
            image = image.type(dtype=torch.float32)
            image = (image / 2 + 0.5).clamp(0, 1)
            image = image.cpu().permute(0, 2, 3, 1).numpy().tobytes()

            # rles
            resp_outputs.append(
                Metadata(
                    name="images",
                    shape=[1, 5, 5, 3],
                    datatype=str(DataType.TYPE_STRING),
                )
            )

            resp_raw_outputs.append(image)

        return construct_infer_response(
            req=req,
            outputs=resp_outputs,
            raw_outputs=resp_raw_outputs,
        )


entrypoint = InstillDeployable(TextToImage).get_deployment_handle()
