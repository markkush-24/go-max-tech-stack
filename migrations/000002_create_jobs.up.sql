CREATE TABLE jobs (
                      id BIGSERIAL PRIMARY KEY,
                      status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),

                      owner_user_id BIGINT NOT NULL,

                      result_user_id BIGINT NULL,
                      error_payload JSONB NULL,

                      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                      updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                      started_at TIMESTAMPTZ NULL,
                      finished_at TIMESTAMPTZ NULL
);

CREATE INDEX jobs_owner_user_id_idx ON jobs (owner_user_id);
CREATE INDEX jobs_status_idx ON jobs (status);
CREATE INDEX jobs_created_at_idx ON jobs (created_at);
