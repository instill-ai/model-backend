BEGIN;

ALTER TYPE valid_task ADD VALUE 'TASK_CHAT';
ALTER TYPE valid_task ADD VALUE 'TASK_COMPLETION';

COMMIT;
