import {
    check,
    group
} from "k6";
import http from "k6/http";
import {
    randomString
} from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

import * as constant from "./const.js";

/**
 * TEST SUITE: AIP Resource Refactoring Invariants
 *
 * PURPOSE:
 * Tests the critical invariants defined in the AIP Resource Refactoring plan.
 * These invariants ensure the system maintains data integrity and follows AIP standards.
 *
 * RED FLAGS (RF) - Hard Invariants:
 * - RF-2: name is the only canonical identifier
 *
 * YELLOW FLAGS (YF) - Strict Guardrails:
 * - YF-2: Slug resolution must not leak into services
 */

export function checkInvariants(header) {
    const namespaceId = constant.defaultUserId;

    // ===============================================================
    // RF-2: name is the Only Canonical Identifier
    // id is derived from name, never authoritative alone
    // ===============================================================
    group("RF-2: name is the canonical identifier", () => {
        const randomSuffix = randomString(8);

        let modelID;
        let modelName;

        // Create a model to test with
        // Note: id is OUTPUT_ONLY after AIP refactoring - server generates it
        group("Setup: Create test model", () => {
            const createPayload = {
                displayName: `Test Invariant Model ${randomSuffix}`,
                description: "Model for AIP invariant testing",
                modelDefinition: "model-definitions/container",
                visibility: "VISIBILITY_PRIVATE",
                region: "REGION_GCP_EUROPE_WEST4",
                hardware: "CPU",
                task: "TASK_CLASSIFICATION",
                configuration: {}
            };

            const createResp = http.post(
                `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`,
                JSON.stringify(createPayload),
                header
            );

            if (createResp.status === 200 || createResp.status === 201) {
                const body = JSON.parse(createResp.body);
                if (body.operation && body.operation.response && body.operation.response.id) {
                    modelID = body.operation.response.id;
                    modelName = body.operation.response.name;
                } else if (body.model) {
                    modelID = body.model.id;
                    modelName = body.model.name;
                }
            }
        });

        if (!modelID) {
            console.error("Failed to create test model, skipping RF-2 tests");
            return;
        }

        // Test 2a: name contains the full resource path
        group("Verify name is full canonical path", () => {
            check({ modelName, namespaceId, modelID }, {
                "[RF-2a] name contains namespace": (d) => d.modelName && d.modelName.includes(`namespaces/${d.namespaceId}`),
                "[RF-2a] name contains resource type": (d) => d.modelName && d.modelName.includes("/models/"),
                "[RF-2a] name ends with id": (d) => d.modelName && d.modelName.endsWith(d.modelID),
                "[RF-2a] name format matches pattern": (d) => {
                    // Pattern: namespaces/{namespace}/models/{id}
                    const pattern = new RegExp(`^namespaces/[^/]+/models/[^/]+$`);
                    return d.modelName && pattern.test(d.modelName);
                }
            });
        });

        // Test 2b: id is derived from name (last segment)
        group("Verify id is derived from name", () => {
            check({ modelName, modelID }, {
                "[RF-2b] id equals last segment of name": (d) => {
                    if (!d.modelName) return false;
                    const segments = d.modelName.split("/");
                    const lastSegment = segments[segments.length - 1];
                    return lastSegment === d.modelID;
                }
            });
        });

        // Cleanup
        http.request(
            "DELETE",
            `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${modelID}`,
            null,
            header
        );
    });

    // ===============================================================
    // YF-2: Slug Resolution Must Not Leak into Services
    // Backend services only accept canonical IDs
    // ===============================================================
    group("YF-2: Backend only accepts canonical IDs", () => {
        const randomSuffix = randomString(8);

        let modelID;

        // Create test model
        // Note: id is OUTPUT_ONLY after AIP refactoring - server generates it
        group("Setup: Create test model", () => {
            const createPayload = {
                displayName: `Test Canonical ID Model ${randomSuffix}`,
                description: "Model for canonical ID testing",
                modelDefinition: "model-definitions/container",
                visibility: "VISIBILITY_PRIVATE",
                region: "REGION_GCP_EUROPE_WEST4",
                hardware: "CPU",
                task: "TASK_CLASSIFICATION",
                configuration: {}
            };

            const createResp = http.post(
                `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`,
                JSON.stringify(createPayload),
                header
            );

            if (createResp.status === 200 || createResp.status === 201) {
                const body = JSON.parse(createResp.body);
                if (body.operation && body.operation.response && body.operation.response.id) {
                    modelID = body.operation.response.id;
                } else if (body.model) {
                    modelID = body.model.id;
                }
            }
        });

        if (!modelID) {
            console.error("Failed to create test model, skipping YF-2 tests");
            return;
        }

        // Test: GET by canonical ID should work
        group("GET by canonical ID succeeds", () => {
            const getResp = http.get(
                `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${modelID}`,
                header
            );

            check(getResp, {
                "[YF-2a] GET by canonical ID returns 200": (r) => r.status === 200,
                "[YF-2a] GET by canonical ID returns correct model": (r) => {
                    const body = JSON.parse(r.body);
                    return body.model && body.model.id === modelID;
                }
            });
        });

        // Test: GET by invalid/fake ID should fail
        group("GET by invalid ID fails", () => {
            const getResp = http.get(
                `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/non-existent-model-id`,
                header
            );

            check(getResp, {
                "[YF-2b] GET by invalid ID returns 404": (r) => r.status === 404 || r.status === 400
            });
        });

        // Cleanup
        http.request(
            "DELETE",
            `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${modelID}`,
            null,
            header
        );
    });

    // ===============================================================
    // Model-specific: Verify model version name format
    // ===============================================================
    group("Model Version: name format validation", () => {
        const randomSuffix = randomString(8);

        let modelID;
        let versionID;
        let versionName;

        // Create test model
        // Note: id is OUTPUT_ONLY after AIP refactoring - server generates it
        group("Setup: Create test model", () => {
            const createPayload = {
                displayName: `Test Version Model ${randomSuffix}`,
                description: "Model for version testing",
                modelDefinition: "model-definitions/container",
                visibility: "VISIBILITY_PRIVATE",
                region: "REGION_GCP_EUROPE_WEST4",
                hardware: "CPU",
                task: "TASK_CLASSIFICATION",
                configuration: {}
            };

            const createResp = http.post(
                `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models`,
                JSON.stringify(createPayload),
                header
            );

            if (createResp.status === 200 || createResp.status === 201) {
                const body = JSON.parse(createResp.body);
                if (body.operation && body.operation.response && body.operation.response.id) {
                    modelID = body.operation.response.id;
                } else if (body.model) {
                    modelID = body.model.id;
                }
            }
        });

        if (!modelID) {
            console.error("Failed to create test model, skipping version tests");
            return;
        }

        // List versions to get the default version (models may auto-create a version)
        group("List model versions", () => {
            const listResp = http.get(
                `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${modelID}/versions`,
                header
            );

            if (listResp.status === 200) {
                const body = JSON.parse(listResp.body);
                if (body.versions && body.versions.length > 0) {
                    versionID = body.versions[0].id;
                    versionName = body.versions[0].name;
                }
            }
        });

        if (versionID && versionName) {
            // Test: Version name follows hierarchical pattern
            group("Verify version name is hierarchical", () => {
                check({ versionName, modelID, versionID }, {
                    "[Version] name contains model path": (d) => d.versionName.includes(`models/${d.modelID}`),
                    "[Version] name contains versions segment": (d) => d.versionName.includes("/versions/"),
                    "[Version] name ends with version id": (d) => d.versionName.endsWith(d.versionID),
                    "[Version] name format matches pattern": (d) => {
                        // Pattern: namespaces/{namespace}/models/{mid}/versions/{vid}
                        const pattern = new RegExp(`^namespaces/[^/]+/models/[^/]+/versions/[^/]+$`);
                        return pattern.test(d.versionName);
                    }
                });
            });
        } else {
            console.log("No model versions found, skipping version name tests");
        }

        // Cleanup
        http.request(
            "DELETE",
            `${constant.apiPublicHost}/v1alpha/${constant.namespace}/models/${modelID}`,
            null,
            header
        );
    });
}
