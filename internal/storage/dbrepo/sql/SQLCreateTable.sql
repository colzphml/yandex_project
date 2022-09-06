CREATE TABLE IF NOT EXISTS public.metrics (
    id varchar(50) NOT NULL,
    mtype varchar(50) NULL,
    delta int8 NULL,
    value float8 NULL,
    CONSTRAINT metrics_pkey PRIMARY KEY (id)
);