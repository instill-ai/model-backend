import io
import numpy as np
import json

from typing import List
from PIL import Image

from triton_python_backend_utils import Tensor, InferenceResponse, \
    get_input_tensor_by_name, InferenceRequest, get_output_config_by_name, \
    get_input_config_by_name, TritonModelException, triton_string_to_numpy


class TritonPythonModel(object):
    def __init__(self):
        self.tf = None
        self.output_names = {
            'orig_img_hw': 'orig_img_hw'
        }

    def initialize(self, args):
        model_config = json.loads(args['model_config'])

        output_configs = {k: get_output_config_by_name(
            model_config, name) for k, name in self.output_names.items()}
        for k, cfg in output_configs.items():
            if cfg is None:
                raise ValueError(
                    f'Output {self.output_names[k]} is not defined in the model config')
            if 'dims' not in cfg:
                raise ValueError(
                    f'Dims for output {self.output_names[k]} are not defined in the model config')
            if 'name' not in cfg:
                raise ValueError(
                    f'Name for output {self.output_names[k]} is not defined in the model config')
            if 'data_type' not in cfg:
                raise ValueError(
                    f'Data type for output {self.output_names[k]} is not defined in the model config')

        self.output_dtypes = {k: triton_string_to_numpy(
            cfg['data_type']) for k, cfg in output_configs.items()}


    def execute(self, inference_requests: List[InferenceRequest]) -> List[InferenceResponse]:
        input_name = 'input'
        output_name = 'output'

        responses = []
        for request in inference_requests:
            # This model only process one input per request. We use
            # get_input_tensor_by_name instead of checking
            # len(request.inputs()) to allow for multiple inputs but
            # only process the one we want. Same rationale for the outputs
            batch_out = {k: [] for k, name in self.output_names.items(
            ) if name in request.requested_output_names()}
            for i in range(request.inputs()[0].as_numpy().shape[0]):
                batch_out['orig_img_hw'].append([10, 10])

            # Format outputs to build an InferenceResponse
            # Assumes there is only one output
            output_tensors = [Tensor(self.output_names[k], np.asarray(
                out, dtype=self.output_dtypes[k])) for k, out in batch_out.items()]

            # TODO: should set error field from InferenceResponse constructor
            # to handle errors
            response = InferenceResponse(output_tensors)
            responses.append(response)

        return responses
