package stores

import (
	"github.com/usace-nsi/nsi-survey-server/models"
	dq "github.com/usace/goquery"
)

var surveyTable = dq.TableDataSet{
	Name:   "survey",
	Schema: "public",
	Statements: map[string]string{
		"selectById": `
			SELECT 
				id, title, description, active, due_date, inventory_source,
				stratification_type, margin, proportions, confidence, pct_control,
				ST_AsGeoJSON(perimeter_geom) AS perimeter_geom
			FROM survey 
			WHERE id = $1`,

		"insert": `
			INSERT INTO survey (
				title, description, active, due_date, inventory_source, 
				stratification_type, margin, proportions, confidence, pct_control,
				perimeter_geom
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
			RETURNING id`,

		"update": `
			UPDATE survey SET 
				title = $1, description = $2, active = $3, due_date = $4,
				inventory_source = $5, stratification_type = $6, margin = $7,
				proportions = $8, confidence = $9, pct_control = $10,
				perimeter_geom = $11
			WHERE id = $12
			`,
		"survey": `
			SELECT 
				sa_id, fd_id, x, y, invalid_structure, no_street_view, cbfips, occtype, 
				st_damcat, found_ht, num_story, sqft, found_type, replacement_type, 
				quality, const_type, garage, roof_style
			FROM survey_result 
			WHERE sa_id = $1`,

		"user-surveys": `
			SELECT DISTINCT 
				s.id, s.title, s.description, s.active, s.due_date, s.inventory_source,
				s.stratification_type, s.margin, s.proportions, s.confidence, s.pct_control,
				ST_AsGeoJSON(s.perimeter_geom) AS perimeter_geom
			FROM survey s
			LEFT OUTER JOIN survey_member sm ON sm.survey_id = s.id
			WHERE sm.user_id = $1`,

		"admin-surveys": `
			SELECT DISTINCT 
				s.id, s.title, s.description, s.active, s.due_date, s.inventory_source,
				s.stratification_type, s.margin, s.proportions, s.confidence, s.pct_control,
				ST_AsGeoJSON(s.perimeter_geom) AS perimeter_geom
			FROM survey s
			LEFT OUTER JOIN survey_member sm ON sm.survey_id = s.id`,

		"insert-owner": `
			INSERT INTO survey_member (survey_id, user_id, is_owner) 
			VALUES ($1, $2, $3)`,
		"owners": `
			SELECT DISTINCT u.user_id, u.user_name
			FROM survey_member m
			LEFT OUTER JOIN users u ON m.user_id = u.user_id
			WHERE m.survey_id = $1 AND m.is_owner = true`,
		"members": `
			SELECT DISTINCT m.id, m.user_id, u.user_name, m.is_owner
			FROM survey_member m
			LEFT OUTER JOIN users u ON m.user_id = u.user_id
			WHERE m.survey_id = $1`,

		"updateGeometryFromElements": `
			UPDATE survey 
			SET perimeter_geom = (
				SELECT ST_ConvexHull(ST_Collect(ST_SetSRID(ST_MakePoint(x, y), 4326)))
				FROM survey_element
				WHERE survey_id = $1
				  AND x IS NOT NULL 
				  AND y IS NOT NULL
			)
			WHERE id = $1`,

		"updateGeometryFromGeoJSON": `
			UPDATE survey 
			SET perimeter_geom = ST_SetSRID(ST_GeomFromGeoJSON($2), 4326)
			WHERE id = $1`,
		"delete": `DELETE FROM survey WHERE id = $1`,
		"selectPerimeter": `
			SELECT ST_AsGeoJSON(perimeter_geom) AS perimeter_geom
			FROM survey
			WHERE id = $1`,
	},
	Fields: models.Survey{},
}

var usersTable = dq.TableDataSet{
	Statements: map[string]string{
		"insert": `insert into users (user_id, user_name) values ($1, $2)
		           ON CONFLICT (user_id) DO UPDATE SET user_name = EXCLUDED.user_name`,
	},
}

var surveyMemberTable = dq.TableDataSet{
	Statements: map[string]string{
		"upsert": `insert into survey_member(survey_id,user_id,is_owner) values ($1,$2,$3)
		                   ON CONFLICT(survey_id,user_id) do
						  update set is_owner=EXCLUDED.is_owner`,
		"select_owners":    "select * from survey_member where survey_id=$1",
		"remove":           `delete from survey_member where user_id=$1`,
		"removeFromSurvey": `delete from survey_member where user_id=$1 and survey_id=$2`,
		"removeBySurvey":   `delete from survey_member where survey_id=$1`,
	},
	Fields: models.SurveyMember{},
}

var surveyElementTable = dq.TableDataSet{
	Name: "survey_element",
	Statements: map[string]string{
		"select_elements": `select survey_order, fd_id, is_control from survey_element where survey_id=$1`,
		"deleteElements":  `DELETE FROM survey_element WHERE survey_id = $1`,
		"progress": `SELECT
    count(*) AS total,
    count(*) FILTER (WHERE EXISTS (
        SELECT 1 FROM survey_assignment sa
        WHERE sa.se_id = se.id
          AND sa.completed = true
    )) AS completed
FROM survey_element se
WHERE se.survey_id = $1
`,
	},
	Fields: models.SurveyElement{},
}

var surveyAssignmentTable = dq.TableDataSet{
	Name: "survey_assignment",
	Statements: map[string]string{
		"updateAssignment": `update survey_assignment set completed='true' where id=$1`,
		"deleteAssignments": `
			DELETE FROM survey_assignment
			WHERE se_id IN (
				SELECT id FROM survey_element WHERE survey_id = $1
			)`,
		"assignSurvey": `insert into survey_assignment (se_id,assigned_to) values ($1,$2) returning id`,
		"assignmentInfo": `
			select
				sa_id,
				se_id,
				completed,
				survey_order,
				next_survey_order,
				next_survey_seid,
				next_control_order,
				next_control_seid
			from (
				select
					t1.id as sa_id,
					t1.se_id,
					t1.completed,
					t2.survey_order,
					null as next_survey_order,
					null as next_survey_seid,
					null as next_control_order,
					null as next_control_seid
				from survey_element t2
				left outer join survey_assignment t1 on t1.se_id=t2.id
				where assigned_to=$1 and t2.survey_id=$2 and completed='false'
				union
				select
					sa_id,
					se_id,
					completed,
					next_assignment.survey_order,
					next_survey_order,
					se1.id as next_survey_seid,
					next_control_order,
					se2.id as next_control_seid
				from (
					select
						null::uuid as sa_id,
						null::uuid as se_id,
						null::bool as completed,
						null::integer as survey_order,
						(
							select case when (
								select max(t2.survey_order)
								from survey_assignment t1
								inner join survey_element t2 on t2.id=t1.se_id
								where t2.survey_id=$2 and t2.is_control='false'
							) is null then (
								select min(survey_order)
								from survey_element
								where survey_id=$2 and is_control='false'
							) else (
								select min(survey_order)
								from survey_element where survey_order> (
									select max(t2.survey_order)
										from survey_assignment t1
										inner join survey_element t2 on t2.id=t1.se_id
										where t2.survey_id=$2 and t2.is_control='false'
								) and is_control='false' and survey_id=$2
							) end
						) as next_survey_order,
						(
							select min(t1.survey_order)
							from survey_element t1
							left outer join (
								select *
								from survey_assignment
								where assigned_to=$1
							) t2 on t1.id=t2.se_id
							where assigned_to is null and is_control='true' and survey_id=$2
						) as next_control_order
								) next_assignment
								inner join survey_element se1 on se1.survey_order=next_assignment.next_survey_order
								left outer join survey_element se2 on se2.survey_order=next_assignment.next_control_order
								where se1.survey_id=$2 and (se2.survey_id=$2  or se2.survey_id is null)
							) assignment_query
							order by survey_order limit 1`,
		"previousAssignment": `
			SELECT id, survey_order, survey_id, fd_id, is_control
			FROM survey_element
			WHERE survey_id = $2 
			AND survey_order < (
				SELECT COALESCE(MAX(se.survey_order), 0)
				FROM survey_assignment sa
				INNER JOIN survey_element se ON se.id = sa.se_id
				WHERE sa.assigned_to = $1 AND se.survey_id = $2
			)
			ORDER BY survey_order DESC
			LIMIT 1`,
		"previousAssignmentExisting": `
			SELECT sa.id AS sa_id, se.id AS se_id
			FROM survey_assignment sa
			INNER JOIN survey_element se ON se.id = sa.se_id
			WHERE sa.assigned_to = $1
			AND se.survey_id = $2
			AND se.survey_order < (
				SELECT COALESCE(MAX(se2.survey_order), 0)
				FROM survey_assignment sa2
				INNER JOIN survey_element se2 ON se2.id = sa2.se_id
				WHERE sa2.assigned_to = $1 AND se2.survey_id = $2
			)
			ORDER BY se.survey_order DESC
			LIMIT 1`,
	},

	Fields: models.SurveyAssignment{},
}

var resultTable = dq.TableDataSet{
	Statements: map[string]string{
		"upsertSurveyStructure": `insert into survey_result
									(sa_id,fd_id,x,y,invalid_structure,no_street_view,cbfips,occtype,st_damcat,found_ht,num_story,sqft,found_type,replacement_type,quality,const_type,garage,roof_style)
									values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
									ON CONFLICT (sa_id)
									DO UPDATE SET x=EXCLUDED.x,y=EXCLUDED.y,invalid_structure=EXCLUDED.invalid_structure,no_street_view=EXCLUDED.no_street_view, cbfips=EXCLUDED.cbfips,
													occtype=EXCLUDED.occtype,st_damcat=EXCLUDED.st_damcat,found_ht=EXCLUDED.found_ht,num_story=EXCLUDED.num_story,
												sqft=EXCLUDED.sqft,found_type=EXCLUDED.found_type,replacement_type=EXCLUDED.replacement_type,
												quality=EXCLUDED.quality,const_type=EXCLUDED.const_type,garage=EXCLUDED.garage,roof_style=EXCLUDED.roof_style`,

		"surveyReport": `select
				t1.id as sr_id,
				t3.user_id,
				t3.user_name,
				t2.completed,
				t4.is_control,
				t1.sa_id,
				t1.fd_id,
				t1.x,
				t1.y,
				t1.cbfips,
				t1.occtype,
				t1.st_damcat,
				t1.found_ht,
				t1.num_story,
				t1.sqft,
				t1.found_type,
				t1.replacement_type,
				t1.quality,
				t1.const_type,
				t1.garage,
				t1.roof_style,
				t1.invalid_structure,
				t1.no_street_view
				from survey_result t1
				inner join survey_assignment t2 on t2.id=t1.sa_id
				inner join users t3 on t3.user_id=t2.assigned_to
				inner join survey_element t4 on t4.id=t2.se_id
				where t4.survey_id=$1`,
		"delete-results": `
			DELETE FROM survey_result
			WHERE sa_id IN (
				SELECT sa.id
				FROM survey_assignment sa
				INNER JOIN survey_element se ON se.id = sa.se_id
				WHERE se.survey_id = $1
			)`,
	},
}
