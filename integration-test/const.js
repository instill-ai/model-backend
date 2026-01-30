let proto;

if (__ENV.API_GATEWAY_PROTOCOL) {
  if (__ENV.API_GATEWAY_PROTOCOL !== "http" && __ENV.API_GATEWAY_PROTOCOL != "https") {
    fail("only allow `http` or `https` for API_GATEWAY_PROTOCOL")
  }
  proto = __ENV.API_GATEWAY_PROTOCOL
} else {
  proto = "http"
}

// API Gateway URL (localhost:8080 from host, api-gateway:8080 from container)
const apiGatewayUrl = __ENV.API_GATEWAY_URL || "localhost:8080";

// Determine if running from host (localhost) or container
export const isHostMode = apiGatewayUrl === "localhost:8080";
// API Gateway mode is always true now (we always use API Gateway)
export const apiGatewayMode = true;

export const defaultUserId = "admin"
// API Gateway uses AIP naming: namespaces/{namespace_id}/...
export const namespace = "namespaces/admin"
export const defaultPassword = "password"

// Public hosts (via API Gateway)
export const apiPublicHost = `${proto}://${apiGatewayUrl}`;
export const gRPCPublicHost = apiGatewayUrl;
export const mgmtPublicHost = `${proto}://${apiGatewayUrl}`;
export const mgmtGRPCPublicHost = apiGatewayUrl;

// Private hosts (direct backend, for internal service calls)
export const apiPrivateHost = "http://model-backend:3083";
export const gRPCPrivateHost = "model-backend:3083";
export const mgmtGRPCPrivateHost = "mgmt-backend:3084";
export const mgmtApiPrivateHost = "http://mgmt-backend:3084";

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
