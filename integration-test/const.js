let proto

export const apiGatewayMode = (__ENV.API_GATEWAY_URL && true);

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (__ENV.API_GATEWAY_PROTOCOL !== "http" && __ENV.API_GATEWAY_PROTOCOL != "https") {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL")
  }
  proto = __ENV.API_GATEWAY_PROTOCOL
} else {
  proto = "http"
}

export const defaultUserId = "instill-ai"
export const namespace = "users/instill-ai"

export const gRPCPrivateHost = "model-backend:3083"
export const apiPrivateHost = "http://model-backend:3083"

export const gRPCPublicHost = apiGatewayMode ? `${__ENV.API_GATEWAY_URL}/model` : `model-backend:8083`
export const apiPublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/model` : `http://model-backend:8083`

export const mgmtGRPCPrivateHost = "mgmt-backend:3084"
export const mgmtApiPrivateHost = "http://mgmt-backend:3084"

export const cls_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dummy-cls-model.zip`, "b");
export const det_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-det-model.zip`, "b");
export const keypoint_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-keypoint-model.zip`, "b");
export const unspecified_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-unspecified-model.zip`, "b");
export const cls_model_bz17 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-cls-model-bz17.zip`, "b");
export const det_model_bz9 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-det-model-bz9.zip`, "b");
export const keypoint_model_bz9 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-keypoint-model-bz9.zip`, "b");
export const unspecified_model_bz3 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-unspecified-model-bz3.zip`, "b");
export const empty_response_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/empty-response-model.zip`, "b");
export const cls_no_readme_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-cls-no-readme.zip`, "b");
export const semantic_segmentation_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-semantic-segmentation-model.zip`, "b");
export const semantic_segmentation_model_bz9 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-semantic-segmentation-model-bz9.zip`, "b");
export const instance_segmentation_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-instance-segmentation-model.zip`, "b");
export const instance_segmentation_model_bz9 = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-instance-segmentation-model-bz9.zip`, "b");
export const text_to_image_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-text-to-image-model.zip`, "b");
export const text_generation_model = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dummy-text-generation-model.zip`, "b");


export const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dog.jpg`, "b");
export const dog_rgba_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dog-rgba.png`, "b");
export const cat_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/cat.jpg`, "b");
export const bear_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/bear.jpg`, "b");
export const dance_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dance.jpg`, "b");
