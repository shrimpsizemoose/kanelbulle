CREATE TABLE IF NOT EXISTS entries (
    timestamp BIGINT NOT NULL,
    event_type TEXT NOT NULL,
    lab VARCHAR(3) NOT NULL,
    student TEXT NOT NULL,
    course VARCHAR(6) NOT NULL,
    comment TEXT
);

