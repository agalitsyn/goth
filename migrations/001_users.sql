CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id                  SERIAL      PRIMARY KEY,
    login               TEXT        UNIQUE NOT NULL,
    hashed_password     TEXT        NOT NULL,
    is_active           boolean     NOT NULL DEFAULT TRUE
);

CREATE TABLE user_sessions (
    uuid				UUID					PRIMARY KEY	DEFAULT uuid_generate_v4(),
    expires_at  		timestamp 				NOT NULL,
    user_id             INTEGER 				NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users (id)
);
