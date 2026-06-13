package auth

import (
	"log"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/usace-nsi/nsi-survey-server/models"
	"github.com/usace-nsi/nsi-survey-server/stores"
	. "github.com/usace/microauth"
)

const (
	PUBLIC = iota
	ADMIN
	SURVEY_OWNER
	SURVEY_MEMBER
)

func Appauth(c *echo.Context, authstore interface{}, roles []int, claims JwtClaim) bool {
	c.Set("NSIUSER", claims)
	store := authstore.(*stores.SurveyStore)
	//adding user to the user table
	//adding user to the user table
	isNew, err := store.AddUser(models.User{
		UserID:   claims.Sub,
		Username: claims.UserName,
	})
	if err != nil {
		log.Printf("failed to add/update user %s: %v", claims.Sub, err)
	} else if isNew {
		// New user: auto-enroll in the training-survey if one exists.
		if err := store.AddUserToTrainingSurvey(claims.Sub); err != nil {
			log.Printf("failed to enroll new user %s in training-survey: %v", claims.Sub, err)
		}
	}


	surveyId, err := uuid.Parse(c.Param("surveyid"))
	if c.Param("surveyid") != "" && err == nil { // there is surveyId in url
		c.Set("NSISURVEY", surveyId)
		if Contains(roles, PUBLIC) {
			return true
		}
		if Contains(roles, ADMIN) && Contains_string(claims.Roles, "ADMIN") {
			return true
		}
		var flagOwner, flagMember bool
		if Contains(roles, SURVEY_OWNER) {
			flagOwner = store.IsOwner(surveyId, claims.Sub)
		}
		if Contains(roles, SURVEY_MEMBER) {
			flagMember = store.IsMember(surveyId, claims.Sub)
		}
		if flagMember || flagOwner {
			return true
		}
	}

	if Contains(roles, PUBLIC) {
		return true
	}
	if Contains(roles, ADMIN) && Contains_string(claims.Roles, "ADMIN") {
		return true
	}
	return false
}
