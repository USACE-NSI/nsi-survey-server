create table survey (
    id uuid not null default gen_random_uuid() primary key,
    title varchar(200) not null,
    description text,
    active boolean,
    due_date varchar(12),
    inventory_source varchar(50),
    stratification_type varchar(15),
    margin double precision default 0.0,
    proportions jsonb not null default '{}'::jsonb,
    confidence double precision default 0.0,
    pct_control double precision default 0.0,
    perimeter_geom geometry(Geometry, 4326)
);

create table survey_element (
    id uuid not null default gen_random_uuid() primary key,
    survey_id uuid not null,
    survey_order int not null,
    fd_id int not null,
    is_control boolean default false,
    strata varchar(15)
);

create table users(
    user_id varchar(50) not null primary key,
    user_name text not null
);

create table survey_member(
    id uuid not null default gen_random_uuid() primary key,
    survey_id uuid not null,
    user_id varchar(50) not null,
    is_owner bool not null default false,
    UNIQUE(survey_id,user_id),
    CONSTRAINT fk_sm_user
        FOREIGN KEY(user_id)
            REFERENCES users(user_id)
);

create table survey_assignment (
    id uuid not null default gen_random_uuid() primary key,
    se_id uuid not null,
    completed boolean DEFAULT false,
    assigned_to varchar(50),
    CONSTRAINT fk_survey_element
        FOREIGN KEY(se_id)
            REFERENCES survey_element(id),
    CONSTRAINT fk_user
        FOREIGN KEY(assigned_to)
            REFERENCES users(user_id)
);

create table survey_result(
    id uuid not null default gen_random_uuid() primary key,
    sa_id uuid not null UNIQUE,
    fd_id int not null,
    X double precision not null,
    Y double precision not null,
    invalid_structure boolean not null,
    no_street_view boolean not null,
    cbfips varchar(15),
    occtype varchar(9),
    st_damcat varchar(3),
    found_ht double precision,
    num_story double precision,
    sqft double precision,
    found_type varchar(4),
    replacement_type varchar(50),
    quality varchar(50),
    const_type varchar(50),
    garage varchar(50),
    roof_style varchar(50),

    CONSTRAINT fk_survey_assignment
        FOREIGN KEY(sa_id)
            REFERENCES survey_assignment(id)

);

CREATE UNIQUE INDEX idx_sr_said ON survey_result (sa_id);
ALTER TABLE survey_result ADD CONSTRAINT unique_sa_id UNIQUE USING INDEX idx_sr_said;

-- Default "training-survey" with its survey elements
insert into survey (id, title, description, active, perimeter_geom)
values ('00000000-0000-0000-0000-000000000001', 'training-survey', 'Default survey used for training.', true,
    -- Bounding box covering the lower 48 (continental US): xmin, ymin, xmax, ymax in EPSG:4326
    ST_MakeEnvelope(-124.848974, 24.396308, -66.885444, 49.384358, 4326));

insert into survey_element (survey_id, survey_order, fd_id, is_control, strata)
values
    ('00000000-0000-0000-0000-000000000001', 1, 566378484, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 2, 523367802, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 3, 523235984, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 4, 523367321, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 5, 566606286, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 6, 565241573, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 7, 537488273, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 8, 537488274, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 9, 523474401, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 10, 566320142, true, 'Test'),
    ('00000000-0000-0000-0000-000000000001', 11, 523292042, true, 'Test');