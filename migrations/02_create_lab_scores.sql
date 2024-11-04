CREATE TABLE IF NOT EXISTS lab_scores (
    deadline BIGINT NOT NULL,
    lab VARCHAR(3) NOT NULL,
    base_score INTEGER NOT NULL,
    course VARCHAR(6) NOT NULL,
    CONSTRAINT lab_scores_pkey PRIMARY KEY (course, lab)
);
