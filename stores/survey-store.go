package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/usace-nsi/nsi-survey-server/config"
	"github.com/usace-nsi/nsi-survey-server/models"
	_ "github.com/usace/goquery/adapters/postgres/v3"
	"github.com/usace/goquery/v3"
)

var NoResults string = "no rows in result set"

type SurveyStore struct {
	DS goquery.DataStore
}

func CreateSurveyStore(appConfig *config.Config) (*SurveyStore, error) {
	//dbconf := appConfig.Rdbmsconfig()
	dbconf := goquery.RdbmsConfigFromEnv()
	ds, err := goquery.NewRdbmsDataStore(dbconf)
	if err != nil {
		log.Printf("Unable to connect to database during startup: %s", err)
		return nil, fmt.Errorf("connecting to database %s:%s/%s as %s: %w",
			dbconf.Dbhost, dbconf.Dbport, dbconf.Dbname, dbconf.Dbuser, err)
		//os.Exit(1)
	}
	// pgx pools connect lazily, so NewRdbmsDataStore succeeding doesn't prove the
	// DB is reachable. Force a round-trip so a dead DB fails HERE, not on the
	// first request (which is what panicked in AddUser).
	if err := ds.Exec(goquery.NoTx, "SELECT 1"); err != nil {
		return nil, fmt.Errorf("database %s:%s/%s not reachable: %w",
			dbconf.Dbhost, dbconf.Dbport, dbconf.Dbname, err)
	}

	// Only log success once we KNOW it's true.
	log.Printf("Connected as %s to database %s:%s/%s",
		dbconf.Dbuser, dbconf.Dbhost, dbconf.Dbport, dbconf.Dbname)
	return &SurveyStore{ds}, nil

}
func CreateSurveyStoreWithRetry(appConfig *config.Config) (*SurveyStore, error) {
	const (
		maxAttempts = 10
		baseDelay   = 1 * time.Second
		maxDelay    = 30 * time.Second
	)
	delay := baseDelay
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ss, err := CreateSurveyStore(appConfig)
		if err == nil {
			log.Printf("datastore connected (attempt %d/%d)", attempt, maxAttempts)
			return ss, nil
		}
		lastErr = err
		log.Printf("datastore connect failed (attempt %d/%d): %v; retrying in %s",
			attempt, maxAttempts, err, delay)
		if attempt < maxAttempts {
			time.Sleep(delay)
			if delay *= 2; delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	return nil, fmt.Errorf("could not connect to datastore after %d attempts: %w", maxAttempts, lastErr)
}

// AddUser upserts the user and reports whether the row was newly inserted
// (true) versus an existing user re-authenticating (false). The xmax = 0 trick
// distinguishes an INSERT from an ON CONFLICT ... DO UPDATE in a single round trip.
func (ss *SurveyStore) AddUser(user models.User) (bool, error) {
	var inserted bool
	err := ss.DS.Select(`
		INSERT INTO users (user_id, user_name)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET user_name = EXCLUDED.user_name
		RETURNING (xmax = 0) AS inserted`).
		Params(user.UserID, user.Username).
		Dest(&inserted).
		Fetch()
	return inserted, err
}

// AddUserToTrainingSurvey enrolls a user as a (non-owner) member of the survey
// titled "training-survey" if it exists. If no such survey exists, it is a no-op
// so first-time login still succeeds. ON CONFLICT DO NOTHING keeps it idempotent
// and never clobbers an existing membership/owner flag.
func (ss *SurveyStore) AddUserToTrainingSurvey(userId string) error {
	var surveyId uuid.UUID
	err := ss.DS.Select(`SELECT id FROM survey WHERE title = $1`).
		Params("training-survey").
		Dest(&surveyId).
		Fetch()
	if err != nil {
		if err.Error() == NoResults {
			return nil // no training-survey; nothing to enroll into
		}
		return err
	}
	return ss.DS.Exec(goquery.NoTx, `
		INSERT INTO survey_member (survey_id, user_id, is_owner)
		VALUES ($1, $2, false)
		ON CONFLICT (survey_id, user_id) DO NOTHING`,
		surveyId, userId)
}

func (ss *SurveyStore) GetSurveysforUser(userId string) (*[]models.Survey, error) {
	surveys := []models.Survey{}
	err := ss.DS.Select().
		DataSet(&surveyTable).
		StatementKey("user-surveys").
		Params(userId).
		Dest(&surveys).
		Fetch()
	return &surveys, err
}

func (ss *SurveyStore) GetSurveysforAdmin() (*[]models.Survey, error) {
	surveys := []models.Survey{}
	err := ss.DS.Select().
		DataSet(&surveyTable).
		StatementKey("admin-surveys").
		Params().
		Dest(&surveys).
		Fetch()
	return &surveys, err
}
func (ss *SurveyStore) GetSurveyOwners(surveyId uuid.UUID) (*[]models.User, error) {
	owners := []models.User{}
	err := ss.DS.Select().
		DataSet(&surveyTable).
		StatementKey("owners").
		Params(surveyId).
		Dest(&owners).
		Fetch()
	return &owners, err
}
func (ss *SurveyStore) GetSurveyMembers(surveyId uuid.UUID) (*[]models.SurveyMemberAlt, error) {
	members := []models.SurveyMemberAlt{}
	err := ss.DS.Select().
		DataSet(&surveyTable).
		StatementKey("members").
		Params(surveyId).
		Dest(&members).
		Fetch()
	return &members, err
}
func (ss *SurveyStore) GetSurveyProgress(surveyId uuid.UUID) (models.SurveyProgress, error) {
	p := models.SurveyProgress{}
	err := ss.DS.Select().
		DataSet(&surveyElementTable).
		StatementKey("progress").
		Params(surveyId).
		Dest(&p).
		Fetch()
	return p, err
}

func (ss *SurveyStore) GetSurveyElements(surveyId uuid.UUID) (*[]models.SurveyElementAlt, error) {
	elements := []models.SurveyElementAlt{}
	err := ss.DS.Select().
		DataSet(&surveyElementTable).
		StatementKey("select_elements").
		Params(surveyId).
		Dest(&elements).
		Fetch()
	return &elements, err
}

func (ss *SurveyStore) GetSurvey(surveyId uuid.UUID) (models.Survey, error) {
	survey := models.Survey{}
	err := ss.DS.Select().
		DataSet(&surveyTable).
		StatementKey("selectById").
		Dest(&survey).
		Params(surveyId).
		Fetch()
	return survey, err
}

func (ss *SurveyStore) CreateNewSurvey(survey models.Survey, userId string) (uuid.UUID, error) {
	var surveyId uuid.UUID
	if survey.Proportions == nil {
		survey.Proportions = map[string]float64{}
	}

	propsJSON, err := json.Marshal(survey.Proportions) // nil map -> "null"; see note
	if err != nil {
		return surveyId, fmt.Errorf("failed to encode proportions: %w", err)
	}

	err = ss.DS.Transaction(func(tx goquery.Tx) {
		err := ss.DS.Select().
			DataSet(&surveyTable).
			Tx(&tx).
			StatementKey("insert").
			Params(
				survey.Title,
				survey.Description,
				survey.Active,
				survey.DueDate,
				survey.InventorySource,
				survey.StratificationType,
				survey.Margin,
				propsJSON,
				survey.Confidence,
				survey.PercentControlStructures,
				survey.PerimeterGeom,
			).
			Dest(&surveyId).
			Fetch()
		if err != nil {
			panic(fmt.Errorf("failed to insert survey: %w", err))
		}

		ptx := tx.PgxTx()
		_, err = ptx.Exec(context.Background(), surveyTable.Statements["insert-owner"], surveyId, userId, true)
		if err != nil {
			panic(fmt.Errorf("failed to map survey owner record constraints: %w", err))
		}
	})
	return surveyId, err
}

func (ss *SurveyStore) UpdateSurvey(survey models.Survey) error {
	if survey.Proportions == nil {
		survey.Proportions = map[string]float64{}
	}

	propsJSON, err := json.Marshal(survey.Proportions) // nil map -> "null"; see note
	if err != nil {
		return fmt.Errorf("failed to encode proportions: %w", err)
	}
	return ss.DS.Exec(
		goquery.NoTx,
		surveyTable.Statements["update"],
		survey.Title,
		survey.Description,
		survey.Active,
		survey.DueDate,
		survey.InventorySource,
		survey.StratificationType,
		survey.Margin,
		propsJSON,
		survey.Confidence,
		survey.PercentControlStructures,
		survey.PerimeterGeom,
		survey.ID,
	)
}

func (ss *SurveyStore) UpsertSurveyMember(member models.SurveyMember) error {
	err := ss.DS.Exec(goquery.NoTx, surveyMemberTable.Statements["upsert"], member.SurveyID, member.UserID, member.IsOwner)
	return err
}

func (ss *SurveyStore) RemoveSurveyMember(memberId uuid.UUID) error {
	err := ss.DS.Exec(goquery.NoTx, surveyMemberTable.Statements["remove"], memberId)
	return err
}

func (ss *SurveyStore) RemoveMemberFromSurvey(memberId string, surveyId uuid.UUID) error {
	err := ss.DS.Exec(goquery.NoTx, surveyMemberTable.Statements["removeFromSurvey"], memberId, surveyId)
	return err
}

func (ss SurveyStore) InsertSurveyElements(elements *[]models.SurveyElement) error {
	err := ss.DS.Insert(&surveyElementTable).
		Records(elements).
		Batch(true).
		BatchSize(len(*elements)).
		Execute()

	if err != nil {
		log.Printf("Error inserting survey elements: %s", err)
	}
	return err
}

func (ss *SurveyStore) AssignSurvey(userId string, seId uuid.UUID) (uuid.UUID, error) {
	var saId uuid.UUID
	err := ss.DS.Select(surveyAssignmentTable.Statements["assignSurvey"]).
		Params(seId, userId).
		Dest(&saId).
		Fetch()
	return saId, err
}

// PreviousAssignedSurveyElement rolls back a user assignment by one survey element order step.
// It returns the newly created survey assignment UUID alongside its corresponding element UUID.
func (ss *SurveyStore) PreviousAssignedSurveyElement(userId string, surveyId uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	type prevRow struct {
		SAID uuid.UUID `db:"sa_id"`
		SEID uuid.UUID `db:"se_id"`
	}
	var row prevRow
	err := ss.DS.Select(surveyAssignmentTable.Statements["previousAssignmentExisting"]).
		Params(userId, surveyId).
		Dest(&row).
		Fetch()
	if err != nil {
		if err.Error() == NoResults {
			return uuid.Nil, uuid.Nil, nil // surveyor is on the first element
		}
		return uuid.Nil, uuid.Nil, err
	}
	return row.SAID, row.SEID, nil
}
func (ss SurveyStore) InsertSurveyAssignments(assignments *[]models.SurveyAssignment) error {
	err := ss.DS.Insert(&surveyAssignmentTable).
		Records(assignments).
		Execute()

	if err != nil {
		log.Printf("Error inserting survey assignments: %s", err)
	}
	return err
}

func (ss *SurveyStore) GetReport(surveyId uuid.UUID) ([]models.SurveyResult, error) {
	s := []models.SurveyResult{}
	err := ss.DS.Select(resultTable.Statements["surveyReport"]).
		Params(surveyId).
		Dest(&s).
		Fetch()
	return s, err
}

func (ss *SurveyStore) GetAssignmentInfo(userId string, surveyId uuid.UUID) (models.AssignmentInfo, error) {
	ai := models.AssignmentInfo{}
	err := ss.DS.Select(surveyAssignmentTable.Statements["assignmentInfo"]).
		Params(userId, surveyId).
		Dest(&ai).
		Fetch()

	if err != nil && err.Error() == NoResults {
		err = nil
	}

	return ai, err
}

func (ss *SurveyStore) GetFirstSurveyInEvent(surveyId uuid.UUID) (uuid.UUID, error) {
	var firstSurvey uuid.UUID
	err := ss.DS.Select("select id from survey_element where survey_order=(select min(survey_order) from survey_element where survey_event_id=$1)").
		Params(surveyId).
		Dest(firstSurvey).
		Fetch()
	return firstSurvey, err
}

func (ss *SurveyStore) GetStructure(seId uuid.UUID, saId uuid.UUID) (models.SurveyStructure, error) {
	s := models.SurveyStructure{}
	err := ss.DS.Select(surveyTable.Statements["survey"]).
		Params(saId).
		Dest(&s).
		Fetch()
	if err == nil {
		log.Printf("Returning existing Survey Result for assignment: %s", saId)
		return s, nil
	}
	if err.Error() != NoResults {
		log.Printf("Failed to query survey results for existing assignment: %s", err)
		return s, err
	}

	// No saved result yet — stub from survey_element so the client can hydrate from NSI.
	var fdId int
	ferr := ss.DS.Select("SELECT fd_id FROM survey_element WHERE id=$1").
		Params(seId).
		Dest(&fdId).
		Fetch()
	if ferr != nil {
		log.Printf("Failed to look up fd_id for seId %s: %s", seId, ferr)
		return s, ferr
	}
	s.SAID = saId
	s.FDID = fdId
	return s, nil
}

func (ss *SurveyStore) SaveSurvey(survey *models.SurveyStructure) error {
	err := ss.DS.Transaction(func(tx goquery.Tx) {
		pgtx := tx.PgxTx()
		_, txerr := pgtx.Exec(context.Background(), resultTable.Statements["upsertSurveyStructure"],
			survey.SAID, survey.FDID, survey.X, survey.Y, survey.InvalidStructure, survey.NoStreetView,
			survey.CBfips, survey.OccupancyType, survey.Damcat, survey.FoundHt, survey.Stories, survey.SqFt,
			survey.FoundType, survey.ReplacementType, survey.Quality, survey.ConstType, survey.Garage, survey.RoofStyle)
		if txerr != nil {
			panic(txerr)
		}
		_, txerr = pgtx.Exec(context.Background(), surveyAssignmentTable.Statements["updateAssignment"], survey.SAID)
		if txerr != nil {
			panic(txerr)
		}
	})
	return err
}
func (s *SurveyStore) DeleteSurvey(surveyId string) error {
	tx, err := s.DS.NewTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		resultTable.Statements["delete-results"],
		surveyAssignmentTable.Statements["deleteAssignments"],
		surveyElementTable.Statements["deleteElements"],
		surveyMemberTable.Statements["removeBySurvey"],
		surveyTable.Statements["delete"],
	}
	for _, q := range stmts {

		if err := s.DS.Exec(&tx, q, surveyId); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (ss *SurveyStore) IsOwner(surveyId uuid.UUID, userId string) bool {
	var owner int
	err := ss.DS.Select("select count(*) as owner from survey_member where survey_id=$1 and user_id=$2 and is_owner=true").
		Params(surveyId, userId).
		Dest(&owner).
		Fetch()
	if err != nil {
		log.Printf("Error in isOwner query:%s\n ", err)
		return false
	}
	return owner > 0
}

func (ss *SurveyStore) IsMember(surveyId uuid.UUID, userId string) bool {
	var member int
	err := ss.DS.Select("select count(*) as owner from survey_member where survey_id=$1 and user_id=$2").
		Params(surveyId, userId).
		Dest(&member).
		Fetch()
	if err != nil {
		log.Printf("Error in isMember query:%s\n ", err)
		return false
	}
	return member > 0
}
