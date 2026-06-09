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



insert into users values ('987654','Randy Goss');
insert into users values ('987655','Will Lehman');
insert into users values ('987656','Nick Lutz');
insert into users values ('987657','Jack Goss');

/*
insert into survey_element (fd_id,is_control) values (9,false);
insert into survey_element (fd_id,is_control) values (8,false);
insert into survey_element (fd_id,is_control) values (7,false);
insert into survey_element (fd_id,is_control) values (6,false);
insert into survey_element (fd_id,is_control) values (5,true);
insert into survey_element (fd_id,is_control) values (4,true);
insert into survey_element (fd_id,is_control) values (3,false);
insert into survey_element (fd_id,is_control) values (2,true);
insert into survey_element (fd_id,is_control) values (1,false);

insert into survey_assignment (se_id,assigned_to,completed) values (1,'nn',true);
insert into survey_assignment (se_id,assigned_to,completed) values (2,'rr',true);
insert into survey_assignment (se_id,assigned_to,completed) values (3,'rr',true);
insert into survey_assignment (se_id,assigned_to,completed) values (4,'ww',false);
insert into survey_assignment (se_id,assigned_to,completed) values (5,'rr',true);
insert into survey_assignment (se_id,assigned_to,completed) values (6,'rr',false);
insert into survey_assignment (se_id,assigned_to,completed) values (5,'nn',true);
*/
