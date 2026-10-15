ALTER TABLE access_requests
  DROP CONSTRAINT IF EXISTS access_requests_role_arn_check;

ALTER TABLE access_requests
  DROP COLUMN IF EXISTS role_arn;
