CREATE TABLE console_users (
	username TEXT PRIMARY KEY,
	password_hash TEXT NOT NULL,
	active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	CONSTRAINT console_users_username_not_blank CHECK (length(trim(username)) > 0),
	CONSTRAINT console_users_password_hash_argon2id CHECK (password_hash LIKE 'argon2id:%')
);
