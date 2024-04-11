import numpy as np
from instill.helpers.const import DataType, TextGenerationChatInput
from instill.helpers.ray_io import StandardTaskIO
from instill.helpers.ray_io import serialize_byte_tensor
from instill.helpers.ray_config import instill_deployment, InstillDeployable
from instill.helpers import (
    construct_infer_response,
    construct_metadata_response,
    Metadata,
)


@instill_deployment
class VisualQuestionAnswering:
    def __init__(self, _):
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
            ],
            outputs=[
                Metadata(
                    name="output",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[-1, -1],
                ),
            ],
        )
        return resp

    async def __call__(self, req):
        resp_outputs = []
        resp_raw_outputs = []

        task_text_generation_chat_input: TextGenerationChatInput = (
            StandardTaskIO.parse_task_text_generation_chat_input(request=req)
        )

        # output
        resp_outputs.append(
            Metadata(
                name="output",
                shape=[1, 1],
                datatype=str(DataType.TYPE_STRING),
            )
        )
        out = [task_text_generation_chat_input.prompt]
        out = [bytes(f"{out[i]}", "utf-8") for i in range(len(out))]
        resp_raw_outputs.append(serialize_byte_tensor(np.asarray(out)))

        return construct_infer_response(
            req=req,
            outputs=resp_outputs,
            raw_outputs=resp_raw_outputs,
        )


entrypoint = InstillDeployable(VisualQuestionAnswering).get_deployment_handle()
