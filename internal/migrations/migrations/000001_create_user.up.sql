CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS users (
	id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	login varchar(25) NOT NULL UNIQUE,
	password_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS orders (
	id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	number varchar(255) NOT NULL UNIQUE,
	user_id int NOT NULL references users(id),
	status varchar(25) NOT NULL DEFAULT 'NEW',
	uploaded_at timestamp DEFAULT CURRENT_TIMESTAMP,
	accrual double precision
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_status ON orders (status);

CREATE TABLE IF NOT EXISTS balances (
	user_id int PRIMARY KEY references users(id),
	current double precision DEFAULT 0.0 CHECK (current >= 0),
	withdrawn double precision DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS withdraw_requests (
	user_id int NOT NULL references users(id),
	order_number varchar(255) NOT NULL,
	sum_value double precision NOT NULL,
	processed_at timestamp DEFAULT CURRENT_TIMESTAMP
);
