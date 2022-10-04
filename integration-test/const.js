export const gRPCHost = __ENV.HOSTNAME ? `${__ENV.HOSTNAME}:8083` : "localhost:8083";
export const apiHost = __ENV.HOSTNAME ? `http://${__ENV.HOSTNAME}:8083` : "http://localhost:8083";

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

export const dog_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dog.jpg`, "b");
export const dog_rgba_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/dog-rgba.png`, "b");
export const cat_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/cat.jpg`, "b");
export const bear_img = open(`${__ENV.TEST_FOLDER_ABS_PATH}/integration-test//data/bear.jpg`, "b");