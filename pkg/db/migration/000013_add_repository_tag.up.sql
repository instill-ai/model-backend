BEGIN;
CREATE TABLE IF NOT EXISTS repository_tag (
    -- "<repository>:tag", e.g. "melancholic-wombat/llava-34b:latest"
    name VARCHAR(255) PRIMARY KEY,
    digest VARCHAR(255) NOT NULL,
    create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);
COMMIT;
