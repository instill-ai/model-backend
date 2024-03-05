import ray
from ray import serve


def undeploy(application_name: str, ray_addr: str):
    if not ray.is_initialized():
        ray.init(address=ray_addr)
    serve.delete(application_name)
