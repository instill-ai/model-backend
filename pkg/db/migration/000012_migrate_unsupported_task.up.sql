BEGIN;

UPDATE model SET task = 'TASK_IMAGE_TO_IMAGE' WHERE task = 'TASK_CUSTOM';

COMMIT;
