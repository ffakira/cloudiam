ALTER TABLE access_requests
  ADD COLUMN role_arn TEXT NOT NULL DEFAULT '';

ALTER TABLE access_requests
  ADD CONSTRAINT access_requests_role_arn_check
  CHECK (length(trim(role_arn)) > 0) NOT VALID;
