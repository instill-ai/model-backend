import io
import logging
import argparse
import numpy as np
import struct
import ray
import torch
import requests

from enum import Enum
from torchvision import transforms
from PIL import Image
from typing import List

ray.init(address="ray://ray_server:10001")

from ray import serve


from const import Task
from ray_pb2 import (
    ModelReadyRequest,
    ModelReadyResponse,
    ModelMetadataRequest,
    ModelMetadataResponse,
    ModelInferRequest,
    ModelInferResponse,
    InferTensor,
)


class DataType(Enum):
    TYPE_BOOL = 1
    TYPE_UINT8 = 2
    TYPE_UINT16 = 3
    TYPE_UINT32 = 4
    TYPE_UINT64 = 5
    TYPE_INT8 = 6
    TYPE_INT16 = 7
    TYPE_INT32 = 8
    TYPE_INT64 = 9
    TYPE_FP16 = 10
    TYPE_FP32 = 11
    TYPE_FP64 = 12
    TYPE_STRING = 13


def serialize_byte_tensor(input_tensor):
    """
    Serializes a bytes tensor into a flat numpy array of length prepended
    bytes. The numpy array should use dtype of np.object_. For np.bytes_,
    numpy will remove trailing zeros at the end of byte sequence and because
    of this it should be avoided.
    Parameters
    ----------
    input_tensor : np.array
        The bytes tensor to serialize.
    Returns
    -------
    serialized_bytes_tensor : np.array
        The 1-D numpy array of type uint8 containing the serialized bytes in 'C' order.
    Raises
    ------
    InferenceServerException
        If unable to serialize the given tensor.
    """

    if input_tensor.size == 0:
        return ()

    # If the input is a tensor of string/bytes objects, then must flatten those
    # into a 1-dimensional array containing the 4-byte byte size followed by the
    # actual element bytes. All elements are concatenated together in "C" order.
    if (input_tensor.dtype == np.object_) or (input_tensor.dtype.type == np.bytes_):
        flattened_ls = []
        for obj in np.nditer(input_tensor, flags=["refs_ok"], order="C"):
            # If directly passing bytes to BYTES type,
            # don't convert it to str as Python will encode the
            # bytes which may distort the meaning
            if input_tensor.dtype == np.object_:
                if isinstance(obj.item(), bytes):
                    s = obj.item()
                else:
                    s = str(obj.item()).encode("utf-8")
            else:
                s = obj.item()
            flattened_ls.append(struct.pack("<I", len(s)))
            flattened_ls.append(s)
        flattened = b"".join(flattened_ls)
        return flattened
    return None


def deserialize_bytes_tensor(encoded_tensor):
    """
    Deserializes an encoded bytes tensor into an
    numpy array of dtype of python objects

    Parameters
    ----------
    encoded_tensor : bytes
        The encoded bytes tensor where each element
        has its length in first 4 bytes followed by
        the content
    Returns
    -------
    string_tensor : np.array
        The 1-D numpy array of type object containing the
        deserialized bytes in 'C' order.

    """
    strs = []
    offset = 0
    val_buf = encoded_tensor
    while offset < len(val_buf):
        l = struct.unpack_from("<I", val_buf, offset)[0]
        offset += 4
        sb = struct.unpack_from("<{}s".format(l), val_buf, offset)[0]
        offset += l
        strs.append(sb)
    return np.array(strs, dtype=bytes)


@serve.deployment()
class ClassificationModel:
    def __init__(self, model_path: str):
        import onnxruntime as ort

        self.application_name = "_".join(model_path.split("/")[3:5])
        self.deployement_name = model_path.split("/")[4]
        self.categories = self._image_labels()
        self.model = ort.InferenceSession(model_path)
        self.tf = transforms.Compose(
            [
                transforms.Resize(224),
                transforms.CenterCrop(224),
                transforms.ToTensor(),
                transforms.Normalize(
                    mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]
                ),
            ]
        )

    def _image_labels(self) -> List[str]:
        categories = []
        url = (
            "https://raw.githubusercontent.com/pytorch/hub/master/imagenet_classes.txt"
        )
        labels = requests.get(url, timeout=10).text
        for label in labels.split("\n"):
            categories.append(label.strip())
        return categories

    def process_model_outputs(self, output: np.array):
        probabilities = torch.nn.functional.softmax(torch.from_numpy(output), dim=0)
        prob, catid = torch.topk(probabilities, 1)

        return catid, prob

    def ModelMetadata(self, req: ModelMetadataRequest) -> ModelMetadataResponse:
        resp = ModelMetadataResponse(
            name=req.name,
            versions=req.version,
            framework="onnx",
            inputs=[
                ModelMetadataResponse.TensorMetadata(
                    name="input",
                    datatype=str(DataType.TYPE_STRING.name),
                    shape=[1],
                ),
            ],
            outputs=[
                ModelMetadataResponse.TensorMetadata(
                    name="output",
                    shape=[1000],
                ),
            ],
        )
        return resp

    def ModelReady(self, req: ModelReadyRequest) -> ModelReadyResponse:
        resp = ModelReadyResponse(ready=True)
        return resp

    async def ModelInfer(self, request: ModelInferRequest) -> ModelInferResponse:
        b_tensors = request.raw_input_contents[0]

        input_tensors = deserialize_bytes_tensor(b_tensors)

        batch_out = []
        for enc in input_tensors:
            img = Image.open(io.BytesIO(enc.astype(bytes)))
            image = np.array(img)
            np_tensor = self.tf(Image.fromarray(image, mode="RGB")).numpy()
            batch_out.append(np_tensor)

        batch_out = np.asarray(batch_out)
        out = self.model.run(None, {"input": batch_out})
        # shape=(1, batch_size, 1000)

        # tensor([[207], [294]]), tensor([[0.7107], [0.7309]])
        cat, score = self.process_model_outputs(out[0])
        s_out = [
            bytes(f"{score[i][0]}:{self.categories[cat[i]]}", "utf-8")
            for i in range(cat.size(0))
        ]

        out = serialize_byte_tensor(np.asarray(s_out))
        out = np.expand_dims(out, axis=0)

        return ModelInferResponse(
            model_name=request.model_name,
            model_version=request.model_version,
            outputs=[
                InferTensor(
                    name="output",
                    shape=[len(batch_out), 1000],
                ),
            ],
            raw_output_contents=out,
        )


def deploy_onnx_model(task: str, num_cpus: str, num_replicas: str, model_path: str):
    if task == Task.TASK_CLASSIFICATION.name:
        c_app = ClassificationModel.options(
            name=model_path.split("/")[5],
            ray_actor_options={
                "num_cpus": float(num_cpus),
            },
            num_replicas=int(num_replicas),
        ).bind(model_path)
        serve.run(
            c_app,
            name="_".join(model_path.split("/")[3:5]),
            route_prefix=f'/{model_path.split("/")[4]}',
        )


def undeploy_onnx_model(model_path: str):
    serve.delete("_".join(model_path.split("/")[3:5]))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--func", required=True, choices=["deploy", "undeploy"], help="deploy/undeploy"
    )
    parser.add_argument("--model", required=True, help="model path ofr the deployment")
    parser.add_argument("--task", default="TASK_UNSPECIFIED", help="task")
    parser.add_argument("--cpus", default="0.2", help="num of cpus for this deployment")
    parser.add_argument(
        "--replicas", default="1", help="num of replicas for this deployment"
    )
    args = parser.parse_args()

    if args.func == "deploy":
        deploy_onnx_model(args.task, args.cpus, args.replicas, args.model)
    elif args.func == "undeploy":
        undeploy_onnx_model(args.model)


# PYTHONPATH=/model-backend/pkg/ray python3 -c 'import ray_server; ray_server.deploy_onnx_model("1", "0.2", "2", "/model-repository/users/3bbb7a18-74a9-4171-9350-300cb168ca4b/m/latest/model.onnx")'
# PYTHONPATH=/model-backend/pkg/ray python3 -c 'import ray_server; ray_server.undeploy_onnx_model("/model-repository/users/3bbb7a18-74a9-4171-9350-300cb168ca4b/m/latest/model.onnx")'
