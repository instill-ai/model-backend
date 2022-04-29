import numpy as np
import json
import sys
import os

from pathlib import Path
from typing import List
from PIL import Image

from triton_python_backend_utils import Tensor, InferenceResponse, \
    get_input_tensor_by_name, InferenceRequest, get_input_config_by_name, \
    get_output_config_by_name, triton_string_to_numpy

class TritonPythonModel(object):
    def __init__(self):
        self.input_names = {
            'orig_img_hw': 'orig_img_hw'
        }
        self.output_names = {
            'output': 'output',
        }

    def initialize(self, args):
        model_config = json.loads(args['model_config'])

        output_configs = {k: get_output_config_by_name(
            model_config, name) for k, name in self.output_names.items()}

        self.output_dtypes = {k: triton_string_to_numpy(
            cfg['data_type']) for k, cfg in output_configs.items()}

    def execute(self, inference_requests: List[InferenceRequest]) -> List[InferenceResponse]:
        responses = []

        for request in inference_requests:
            batch_out = {k: [] for k, name in self.output_names.items(
            ) if name in request.requested_output_names()}

            batch_out['output'].append([0, 1])

            # Format outputs to build an InferenceResponse
            output_tensors = [Tensor(self.output_names[k], np.asarray(
                out, dtype=self.output_dtypes[k])) for k, out in batch_out.items()]

            # TODO: should set error field from InferenceResponse constructor to handle errors
            # https://github.com/triton-inference-server/python_backend#execute
            # https://github.com/triton-inference-server/python_backend#error-handling
            response = InferenceResponse(output_tensors)
            responses.append(response)

        return responses        