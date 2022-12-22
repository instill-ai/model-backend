let proto, host, port

if (__ENV.MODE == "api-gateway") {
    // api-gateway mode
    proto = "https"
    host = "api-gateway"
    port = 8080
} else if (__ENV.MODE == "localhost") {
    // localhost mode for GitHub Actions
    proto = "https"
    host = "localhost"
    port = 8080
} else {
    // direct microservice mode
    proto = "http"
    host = "model-backend"
    port = 8083
}

export const gRPCHost = `${host}:${port}`
export const apiHost = `${proto}://${host}:${port}`

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


export const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dog.jpg`, "b");
export const dog_rgba_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dog-rgba.png`, "b");
export const cat_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/cat.jpg`, "b");
export const bear_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/bear.jpg`, "b");