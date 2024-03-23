BEGIN;

ALTER TABLE "model"
    DROP COLUMN IF EXISTS "region",
    DROP COLUMN IF EXISTS "hardware_spec",
    DROP COLUMN IF EXISTS "github_link",
    DROP COLUMN IF EXISTS "link",
    DROP COLUMN IF EXISTS "license",
    DROP COLUMN IF EXISTS "namespace";
    DROP COLUMN IF EXISTS "version";


DROP TABLE IF EXISTS `predict`;
DROP TABLE IF EXISTS `model_version`;


COMMIT;
