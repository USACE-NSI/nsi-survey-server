package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/usace-nsi/nsi-survey-server/models"
	"github.com/usace-nsi/nsi-survey-server/stores"
	"github.com/usace/goquery/v3"
	"github.com/usace/microauth"
)

var newSurveyId string
var testDataStore goquery.DataStore = nil

func TestCreateSurvey(t *testing.T) {
	createJSON := `{"title":"Survey Test","description":"This is a description of the test survey","active":true}`
	rec, c := buildContext(http.MethodPost, createJSON, "987654")
	h := buildHandler()
	if assert.NoError(t, h.CreateNewSurvey(&c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		out := rec.Body.String()
		assert.Equal(t, `{"surveyId":`, out[0:12])
		newSurveyId = out[13 : len(out)-2]
		t.Log("Created new survey:" + newSurveyId)
	}
}

func TestUpdateSurvey(t *testing.T) {
	updateJSON := fmt.Sprintf(`{"id":"%s","title":"Survey Test Updated","description":"This is a description of survey edited","active":false}`, newSurveyId)
	rec, c := buildContext(http.MethodPost, updateJSON, "987654")
	//c.SetParamNames("surveyid")
	//c.SetParamValues(newSurveyId)
	sid, _ := uuid.Parse(newSurveyId)
	c.Set("NSISURVEY", sid)

	h := buildHandler()
	if assert.NoError(t, h.UpdateSurvey(&c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestInsertSurveyMember(t *testing.T) {
	payload := fmt.Sprintf(`{"surveyId":"%s","userId":"987654","isOwner":false}`, newSurveyId)
	rec, c := buildContext(http.MethodPost, payload, "987654")
	//c.SetParamNames("surveyid")
	//c.SetParamValues(newSurveyId)
	sid, _ := uuid.Parse(newSurveyId)
	c.Set("NSISURVEY", sid)
	h := buildHandler()
	if assert.NoError(t, h.UpsertSurveyMember(&c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
}

func TestUpdateSurveyMember(t *testing.T) {
	payload := fmt.Sprintf(`{"surveyId":"%s","userId":"987654","isOwner":true}`, newSurveyId)
	rec, c := buildContext(http.MethodPost, payload, "987654")
	//c.SetParamNames("surveyid")
	//c.SetParamValues(newSurveyId)
	sid, _ := uuid.Parse(newSurveyId)
	c.Set("NSISURVEY", sid)
	h := buildHandler()
	if assert.NoError(t, h.UpsertSurveyMember(&c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
}

func TestInsertSecondSurveyMember(t *testing.T) {
	payload := fmt.Sprintf(`{"surveyId":"%s","userId":"987655","isOwner":true}`, newSurveyId)
	rec, c := buildContext(http.MethodPost, payload, "987654")
	//c.SetParamNames("surveyid")
	//c.SetParamValues(newSurveyId)
	sid, _ := uuid.Parse(newSurveyId)
	c.Set("NSISURVEY", sid)
	h := buildHandler()
	if assert.NoError(t, h.UpsertSurveyMember(&c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
}

func TestInsertSurveyElements(t *testing.T) {
	payload := fmt.Sprintf(`
	[
		{"surveyId":"%s","surveyOrder":1,"fdId":95009, "isControl":false},
		{"surveyId":"%s","surveyOrder":2,"fdId":95008, "isControl":false},
		{"surveyId":"%s","surveyOrder":3,"fdId":95007, "isControl":false},
		{"surveyId":"%s","surveyOrder":4,"fdId":95006, "isControl":false},
		{"surveyId":"%s","surveyOrder":5,"fdId":95005, "isControl":true},
		{"surveyId":"%s","surveyOrder":6,"fdId":95004, "isControl":false},
		{"surveyId":"%s","surveyOrder":7,"fdId":95003, "isControl":true},
		{"surveyId":"%s","surveyOrder":8,"fdId":95002, "isControl":false},
		{"surveyId":"%s","surveyOrder":9,"fdId":95001, "isControl":false}
	]`, newSurveyId, newSurveyId, newSurveyId, newSurveyId, newSurveyId, newSurveyId, newSurveyId, newSurveyId, newSurveyId)
	rec, c := buildContext(http.MethodPost, payload, "987654")
	//c.SetParamNames("surveyid")
	//c.SetParamValues(newSurveyId)
	sid, _ := uuid.Parse(newSurveyId)
	c.Set("NSISURVEY", sid)
	h := buildHandler()
	if assert.NoError(t, h.InsertSurveyElements(&c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
	}
}

func TestInsertSurveyAssignments(t *testing.T) {
	se, err := getSurveyElement(1)
	if assert.NoError(t, err) {
		payload := fmt.Sprintf(`
		[
			{"seId":"%s","completed":false, "assignedTo":"987654"},
			{"seId":"%s","completed":false, "assignedTo":"987655"}
		]`, se.ID, se.ID)
		rec, c := buildContext(http.MethodPost, payload, "987654")
		//c.SetParamNames("surveyid")
		//c.SetParamValues(newSurveyId)
		sid, _ := uuid.Parse(newSurveyId)
		c.Set("NSISURVEY", sid)
		h := buildHandler()
		if assert.NoError(t, h.AddAssignments(&c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	}
}

func TestGetSurveyAssignment(t *testing.T) {
	testdata := []string{"4", "4", "4", "5", "4", "5", "5", "4", "5", "5", "4", "4", "4", "5"}
	//results:=[]int{9,8,7,9,6,5,4,5,3,2,3,1}
	for _, v := range testdata {
		userId := fmt.Sprintf("98765%s", v)
		fetchSurveyAssignment(userId, t)
		saveSurveyAssignment(userId, t)
	}
}

// /////////////////interior tests//////////////////
func fetchSurveyAssignment(userId string, t *testing.T) {
	rec, c := buildContext(http.MethodGet, "", userId)
	//c.SetParamNames("surveyid")
	//c.SetParamValues(newSurveyId)
	h := buildHandler()
	if assert.NoError(t, h.AssignSurveyElement(&c)) {
		fmt.Printf("Fetching next survey for %s\n", userId)
		fmt.Println(rec.Body.String())
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

// @TODO...add survey params to url to prevent unauthorized updates
func saveSurveyAssignment(userId string, t *testing.T) {
	structure, err := getStructure(userId)
	if structure.FDID == 0 {
		t.Log("No Assigned Structure Found")
	} else {
		if assert.NoError(t, err) {
			structure.FoundHt = 999.0
			structure.NoStreetView = true
			json, err := json.Marshal(structure)
			if err != nil {
				t.Log(err)
			}
			payload := string(json)
			rec, c := buildContext(http.MethodPost, payload, userId)
			h := buildHandler()
			if assert.NoError(t, h.SaveSurveyAssignment(&c)) {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		}
	}
}

////////////////////////////////////////////////

/////Private support methods///////

func TestMain(m *testing.M) {
	err := initDb()
	if err != nil {
		fmt.Println(err)
		return
	}
	retCode := m.Run()
	os.Exit(retCode)
}

func initDb() error {
	fmt.Println(">>>>>>>Initializing DB<<<<<<<<<<")
	cleanfile, ioErr := ioutil.ReadFile("/Users/rdcrlrsg/Projects/programming/hec/nsi_survey_server/clean-db.sql")
	if ioErr != nil {

		return ioErr
	}
	loadfile, ioErr := ioutil.ReadFile("/Users/rdcrlrsg/Projects/programming/hec/nsi_survey_server/nsi-survey.sql")
	if ioErr != nil {
		return ioErr
	}
	ds := getDataStore()
	cleansql := string(cleanfile)
	loadsql := string(loadfile)
	err := ds.Exec(goquery.NoTx, cleansql+loadsql)
	if err != nil {
		return err
	}
	return nil
}

func getDataStore() goquery.DataStore {
	if testDataStore != nil {
		return testDataStore
	} else {
		config := goquery.RdbmsConfigFromEnv()
		ds, err := goquery.NewRdbmsDataStore(config)
		if err != nil {
			fmt.Printf("Failed to connect to store:%s\n", err)
		}
		testDataStore = ds
		return ds
	}
}

func getSurveyElement(surveyOrder int) (models.SurveyElement, error) {
	ds := getDataStore()
	se := models.SurveyElement{}
	err := ds.Select("select * from survey_element where survey_id=$1 and survey_order=$2").
		Params(newSurveyId, surveyOrder).
		Dest(&se).
		Fetch()
	return se, err
}

func getSurveyAssignment(surveyID uuid.UUID, userID string) (models.SurveyAssignment, error) {
	ds := getDataStore()
	sa := models.SurveyAssignment{}
	err := ds.Select("select * from survey_assignment where survey_id=$1 and user_id=$2").
		Params(surveyID, userID).
		Dest(&sa).
		Fetch()
	return sa, err
}

func getClaims(userId string) microauth.JwtClaim {
	claims := microauth.JwtClaim{
		Sub:      userId,
		Aud:      []string{"nsi-survey"},
		UserName: "Test.User",
	}
	return claims
}

func buildHandler() *SurveyHandler {
	ds := getDataStore()
	surveystore := &stores.SurveyStore{DS: ds}
	return CreateSurveyHandler(surveystore)
}

func buildContext(method string, payload string, userId string) (*httptest.ResponseRecorder, echo.Context) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("NSIUSER", getClaims(userId))
	return rec, *c
}

func getStructure(userId string) (models.SurveyStructure, error) {
	var structure models.SurveyStructure
	var sterr error
	ds := getDataStore()
	ss := stores.SurveyStore{ds}
	sid, _ := uuid.Parse(newSurveyId)
	assignmentInfo, err := ss.GetAssignmentInfo(userId, sid)
	if assignmentInfo.SEID != nil && assignmentInfo.SAID != nil {
		structure, sterr = ss.GetStructure(*assignmentInfo.SEID, *assignmentInfo.SAID)
		if sterr != nil {
			if err == nil {
				err = sterr
			}
		}
	}
	return structure, err
}
