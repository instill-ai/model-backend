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

export const defaultUserId = "admin"
export const namespace = "users/admin"
export const defaultPassword = "password"


export const gRPCPrivateHost = "localhost:3083"
export const apiPrivateHost = "http://model-backend:3083"

export const gRPCPublicHost = apiGatewayMode ? `${__ENV.API_GATEWAY_URL}` : `api-gateway:8080`
export const apiPublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/model` : `http://api-gateway:8080/model`

export const mgmtGRPCPublicHost = apiGatewayMode ? `${__ENV.API_GATEWAY_URL}` : `api-gateway:8080`
export const mgmtPublicHost = apiGatewayMode ? `${proto}://${__ENV.API_GATEWAY_URL}/core` : `http://api-gateway:8080/core`

export const mgmtGRPCPrivateHost = "mgmt-backend:3084"
export const mgmtApiPrivateHost = "http://mgmt-backend:3084"

export const model_def_name = "model-definitions/container"

export const cls_model = "dummy-cls"
export const det_model = "dummy-det"
export const keypoint_model = "dummy-keypoint"
export const semantic_segmentation_model = "dummy-semantic-segmentation"
export const instance_segmentation_model = "dummy-instance-segmentation"
export const text_to_image_model = "dummy-text-to-image"
export const image_to_image_model = "dummy-image-to-image"
export const text_generation_model = "dummy-text-generation"
export const text_generation_chat_model = "dummy-text-generation-chat"
export const visual_question_answering = "dummy-visual-question-answering"


export const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog.jpg`, "b");
export const dog_rgba_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dog-rgba.png`, "b");
export const cat_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/cat.jpg`, "b");
export const bear_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/bear.jpg`, "b");
export const dance_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test/data/dance.jpg`, "b");
