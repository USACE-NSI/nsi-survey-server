package main

import (
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/usace/microauth"

	. "github.com/usace-nsi/nsi-survey-server/auth"
	"github.com/usace-nsi/nsi-survey-server/config"
	"github.com/usace-nsi/nsi-survey-server/handlers"
	"github.com/usace-nsi/nsi-survey-server/stores"
)

func main() {
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(err.Error())
	}
	//cfg.SkipJWT = true

	ss, err := stores.CreateSurveyStore(&cfg)
	if err != nil {
		log.Printf("Unable to connect to database during startup: %s", err)
	}
	//fmt.Println(cfg.Ippk)
	surveyHandler := handlers.CreateSurveyHandler(ss)
	auth := microauth.Auth{
		AuthRoute: Appauth,
		Aud:       cfg.Aud,
		Store:     ss,
	}
	if err := auth.LoadVerificationKey(microauth.VerificationKeyOptions{
		KeySource: microauth.KeyFile,
		KeyVal:    cfg.Ippk,
	}); err != nil {
		log.Fatalf("LoadVerificationKey failed: %v", err)
	}

	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	publicGroup := e.Group("survey-tool")
	loggedInGroup := e.Group("survey-tool")

	publicGroup.GET("/version", surveyHandler.Version)
	loggedInGroup.GET("/surveys", auth.AuthorizeRoute(surveyHandler.GetSurveysForUser, PUBLIC))
	loggedInGroup.POST("/survey", auth.AuthorizeRoute(surveyHandler.CreateNewSurvey, ADMIN, PUBLIC))
	loggedInGroup.DELETE("/survey/:surveyid", auth.AuthorizeRoute(surveyHandler.DeleteSurvey, ADMIN, SURVEY_OWNER))
	loggedInGroup.PUT("/survey/:surveyid", auth.AuthorizeRoute(surveyHandler.UpdateSurvey, ADMIN, SURVEY_OWNER))
	loggedInGroup.GET("/survey/:surveyid/members", auth.AuthorizeRoute(surveyHandler.GetSurveyMembers, ADMIN, SURVEY_OWNER))
	loggedInGroup.POST("/survey/:surveyid/member", auth.AuthorizeRoute(surveyHandler.UpsertSurveyMember, ADMIN, SURVEY_OWNER))
	loggedInGroup.DELETE("/survey/member/:memberid", auth.AuthorizeRoute(surveyHandler.RemoveSurveyMember, ADMIN, SURVEY_OWNER))
	loggedInGroup.DELETE("/survey/:surveyid/member/:memberid", auth.AuthorizeRoute(surveyHandler.RemoveMemberFromSurvey, ADMIN, SURVEY_OWNER))
	loggedInGroup.GET("/survey/:surveyid/perimeter", auth.AuthorizeRoute(surveyHandler.GetSurveyPerimeter, ADMIN, SURVEY_OWNER, SURVEY_MEMBER))
	loggedInGroup.GET("/survey/:surveyid/elements", auth.AuthorizeRoute(surveyHandler.GetSurveyElements, ADMIN, SURVEY_OWNER))
	loggedInGroup.POST("/survey/:surveyid/elements", auth.AuthorizeRoute(surveyHandler.InsertSurveyElements, ADMIN, SURVEY_OWNER))
	loggedInGroup.POST("/survey/:surveyid/assignments", auth.AuthorizeRoute(surveyHandler.AddAssignments, ADMIN, SURVEY_OWNER))
	loggedInGroup.GET("/survey/:surveyid/assignment", auth.AuthorizeRoute(surveyHandler.AssignSurveyElement, ADMIN, SURVEY_OWNER, SURVEY_MEMBER))
	loggedInGroup.POST("/survey/:surveyid/assignment", auth.AuthorizeRoute(surveyHandler.SaveSurveyAssignment, ADMIN, SURVEY_OWNER, SURVEY_MEMBER))
	loggedInGroup.GET("/survey/:surveyid/previous", auth.AuthorizeRoute(surveyHandler.PreviousSurveyElement, ADMIN, SURVEY_OWNER, SURVEY_MEMBER))
	loggedInGroup.GET("/users", auth.AuthorizeRoute(surveyHandler.GetAllUsers, PUBLIC))
	loggedInGroup.GET("/users/search", auth.AuthorizeRoute(surveyHandler.SearchUsers, PUBLIC))
	loggedInGroup.GET("/survey/valid", auth.AuthorizeRoute(surveyHandler.ValidSurveyName, PUBLIC))
	loggedInGroup.GET("/survey/:surveyid/report", auth.AuthorizeRoute(surveyHandler.GetSurveyReport, ADMIN, SURVEY_OWNER))

	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}

}
