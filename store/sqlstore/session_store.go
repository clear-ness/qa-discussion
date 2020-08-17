package sqlstore

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlSessionStore struct {
	store.Store
}

func NewSqlSessionStore(sqlStore store.Store) store.SessionStore {
	us := &SqlSessionStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Session{}, "Sessions").SetKeys(false, "Id")
	}

	return us
}

func (me SqlSessionStore) Save(session *model.Session) (*model.Session, *model.AppError) {
	if len(session.Id) > 0 {
		return nil, model.NewAppError("SqlSessionStore.Save", "store.sql_session.save.existing.app_error", nil, "id="+session.Id, http.StatusBadRequest)
	}

	session.PreSave()

	if err := me.GetMaster().Insert(session); err != nil {
		return nil, model.NewAppError("SqlSessionStore.Save", "store.sql_session.save.app_error", nil, "id="+session.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	teamMembers, err := me.Team().GetTeamsForUser(session.UserId)
	if err != nil {
		return nil, model.NewAppError("SqlSessionStore.Save", "store.sql_session.save.app_error", nil, "id="+session.Id+", "+err.Error(), http.StatusInternalServerError)
	}

	session.TeamMembers = make([]*model.TeamMember, 0, len(teamMembers))
	for _, tm := range teamMembers {
		if tm.DeleteAt == 0 {
			session.TeamMembers = append(session.TeamMembers, tm)
		}
	}

	return session, nil
}

func (me SqlSessionStore) Get(sessionIdOrToken string) (*model.Session, *model.AppError) {
	var sessions []*model.Session

	if _, err := me.GetReplica().Select(&sessions, "SELECT * FROM Sessions WHERE Token = :Token OR Id = :Id LIMIT 1", map[string]interface{}{"Token": sessionIdOrToken, "Id": sessionIdOrToken}); err != nil {
		return nil, model.NewAppError("SqlSessionStore.Get", "store.sql_session.get.app_error", nil, "sessionIdOrToken="+sessionIdOrToken+", "+err.Error(), http.StatusInternalServerError)
	} else if len(sessions) == 0 {
		return nil, model.NewAppError("SqlSessionStore.Get", "store.sql_session.get.app_error", nil, "sessionIdOrToken="+sessionIdOrToken, http.StatusNotFound)
	}
	session := sessions[0]

	tempMembers, err := me.Team().GetTeamsForUser(sessions[0].UserId)
	if err != nil {
		return nil, model.NewAppError("SqlSessionStore.Get", "store.sql_session.get.app_error", nil, "sessionIdOrToken="+sessionIdOrToken+", "+err.Error(), http.StatusInternalServerError)
	}
	sessions[0].TeamMembers = make([]*model.TeamMember, 0, len(tempMembers))
	for _, tm := range tempMembers {
		if tm.DeleteAt == 0 {
			sessions[0].TeamMembers = append(sessions[0].TeamMembers, tm)
		}
	}

	return session, nil
}

func (me SqlSessionStore) Remove(sessionIdOrToken string) *model.AppError {
	_, err := me.GetMaster().Exec("DELETE FROM Sessions WHERE Id = :Id Or Token = :Token", map[string]interface{}{"Id": sessionIdOrToken, "Token": sessionIdOrToken})

	if err != nil {
		return model.NewAppError("SqlSessionStore.RemoveSession", "store.sql_session.remove.app_error", nil, "id="+sessionIdOrToken+", err="+err.Error(), http.StatusInternalServerError)
	}
	return nil
}

func (me SqlSessionStore) RemoveByUserId(userId string) *model.AppError {
	_, err := me.GetMaster().Exec("DELETE FROM Sessions WHERE UserId = :UserId", map[string]interface{}{"UserId": userId})

	if err != nil {
		return model.NewAppError("SqlSessionStore.RemoveByUserId", "store.sql_session.remove_by_user_id.app_error", nil, "user_id="+userId+", err="+err.Error(), http.StatusInternalServerError)
	}
	return nil
}
