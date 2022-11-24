import io
import numpy as np
import json
import os
from typing import List
from PIL import Image
from transformers import AutoFeatureExtractor

from triton_python_backend_utils import Tensor, InferenceResponse, \
    get_input_tensor_by_name, InferenceRequest


class TritonPythonModel(object):
    def __init__(self):
        self.tf = None

    def initialize(self, args):
        dir_path = os.path.dirname(os.path.realpath(__file__))
        self.feature_extractor = AutoFeatureExtractor.from_pretrained(dir_path + '/config.json')  

    def execute(self, inference_requests: List[InferenceRequest]) -> List[InferenceResponse]:
        input_name = 'input'
        output_name = 'output'

        responses = []
        for request in inference_requests:
            # This model only process one input per request. We use
            # get_input_tensor_by_name instead of checking
            # len(request.inputs()) to allow for multiple inputs but
            # only process the one we want. Same rationale for the outputs
            batch_in_tensor: Tensor = get_input_tensor_by_name(request, input_name)
            if batch_in_tensor is None:
                raise ValueError(f'Input tensor {input_name} not found '
                                 f'in request {request.request_id()}')

            if output_name not in request.requested_output_names():
                raise ValueError(f'The output with name {output_name} is '
                                 f'not in the requested outputs '
                                 f'{request.requested_output_names()}')

            batch_in = batch_in_tensor.as_numpy()  # shape (batch_size, 1)

            if batch_in.dtype.type is not np.object_:
                raise ValueError(f'Input datatype must be np.object_, '
                                 f'got {batch_in.dtype.type}')
            
            batch_out = []
            for img in batch_in:  # img is shape (1,)
                pil_img = Image.open(io.BytesIO(img.astype(bytes)))
                inputs = self.feature_extractor(images=pil_img, return_tensors="pt")
                batch_out.append(np.squeeze(np.asarray(inputs['pixel_values']), axis=0))
            batch_out = np.asarray(batch_out)
            # Format outputs to build an InferenceResponse
            # Assumes there is only one output
            output_tensors = [Tensor(output_name, batch_out)]

            # TODO: should set error field from InferenceResponse constructor
            # to handle errors
            response = InferenceResponse(output_tensors)
            responses.append(response)

        return responses
