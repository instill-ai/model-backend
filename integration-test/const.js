let proto, host, publicPort, privatePort

if (__ENV.MODE == "api-gateway") {
    // api-gateway mode
    proto = "http"
    host = "api-gateway"
    publicPort = 8080
    privatePort = 3083
} else if (__ENV.MODE == "localhost") {
    // localhost mode for GitHub Actions
    proto = "http"
    host = "localhost"
    publicPort = 8080
    privatePort = 3083
} else {
    // direct microservice mode
    proto = "http"
    host = "model-backend"
    publicPort = 8083
    privatePort = 3083
}

export const gRPCPrivateHost = `${host}:${privatePort}`
export const apiPrivateHost = `${proto}://${host}:${privatePort}`

export const gRPCPublicHost = `${host}:${publicPort}`
export const apiPublicHost = `${proto}://${host}:${publicPort}`

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
