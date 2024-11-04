CREATE TABLE IF NOT EXISTS score_overrides (
    student TEXT NOT NULL CHECK (student ~ '^[\w-]+\..+$'),
    lab VARCHAR(3) NOT NULL,
    score INTEGER NOT NULL,
    course VARCHAR(6) NOT NULL,
    reason TEXT,
    CONSTRAINT score_overrides_pkey PRIMARY KEY (course, lab, student)
);
