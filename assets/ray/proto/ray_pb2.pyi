from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ModelMetadataRequest(_message.Message):
    __slots__ = ["name", "version"]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    def __init__(self, name: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class InferTensor(_message.Message):
    __slots__ = ["name", "datatype", "shape"]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DATATYPE_FIELD_NUMBER: _ClassVar[int]
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    name: str
    datatype: str
    shape: _containers.RepeatedScalarFieldContainer[int]
    def __init__(self, name: _Optional[str] = ..., datatype: _Optional[str] = ..., shape: _Optional[_Iterable[int]] = ...) -> None: ...

class ModelMetadataResponse(_message.Message):
    __slots__ = ["name", "versions", "framework", "inputs", "outputs"]
    class TensorMetadata(_message.Message):
        __slots__ = ["name", "datatype", "shape"]
        NAME_FIELD_NUMBER: _ClassVar[int]
        DATATYPE_FIELD_NUMBER: _ClassVar[int]
        SHAPE_FIELD_NUMBER: _ClassVar[int]
        name: str
        datatype: str
        shape: _containers.RepeatedScalarFieldContainer[int]
        def __init__(self, name: _Optional[str] = ..., datatype: _Optional[str] = ..., shape: _Optional[_Iterable[int]] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    name: str
    versions: _containers.RepeatedScalarFieldContainer[str]
    framework: str
    inputs: _containers.RepeatedCompositeFieldContainer[ModelMetadataResponse.TensorMetadata]
    outputs: _containers.RepeatedCompositeFieldContainer[ModelMetadataResponse.TensorMetadata]
    def __init__(self, name: _Optional[str] = ..., versions: _Optional[_Iterable[str]] = ..., framework: _Optional[str] = ..., inputs: _Optional[_Iterable[_Union[ModelMetadataResponse.TensorMetadata, _Mapping]]] = ..., outputs: _Optional[_Iterable[_Union[ModelMetadataResponse.TensorMetadata, _Mapping]]] = ...) -> None: ...

class RayServiceCallRequest(_message.Message):
    __slots__ = ["model_name", "model_version", "inputs", "outputs", "raw_input_contents"]
    class InferRequestedOutputTensor(_message.Message):
        __slots__ = ["name"]
        NAME_FIELD_NUMBER: _ClassVar[int]
        name: str
        def __init__(self, name: _Optional[str] = ...) -> None: ...
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    RAW_INPUT_CONTENTS_FIELD_NUMBER: _ClassVar[int]
    model_name: str
    model_version: str
    inputs: _containers.RepeatedCompositeFieldContainer[InferTensor]
    outputs: _containers.RepeatedCompositeFieldContainer[RayServiceCallRequest.InferRequestedOutputTensor]
    raw_input_contents: _containers.RepeatedScalarFieldContainer[bytes]
    def __init__(self, model_name: _Optional[str] = ..., model_version: _Optional[str] = ..., inputs: _Optional[_Iterable[_Union[InferTensor, _Mapping]]] = ..., outputs: _Optional[_Iterable[_Union[RayServiceCallRequest.InferRequestedOutputTensor, _Mapping]]] = ..., raw_input_contents: _Optional[_Iterable[bytes]] = ...) -> None: ...

class RayServiceCallResponse(_message.Message):
    __slots__ = ["model_name", "model_version", "outputs", "raw_output_contents"]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    RAW_OUTPUT_CONTENTS_FIELD_NUMBER: _ClassVar[int]
    model_name: str
    model_version: str
    outputs: _containers.RepeatedCompositeFieldContainer[InferTensor]
    raw_output_contents: _containers.RepeatedScalarFieldContainer[bytes]
    def __init__(self, model_name: _Optional[str] = ..., model_version: _Optional[str] = ..., outputs: _Optional[_Iterable[_Union[InferTensor, _Mapping]]] = ..., raw_output_contents: _Optional[_Iterable[bytes]] = ...) -> None: ...
