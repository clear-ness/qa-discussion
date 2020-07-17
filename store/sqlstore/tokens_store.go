package sqlstore

import (
	"database/sql"
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlTokenStore struct {
	store.Store
}

func NewSqlTokenStore(sqlStore store.Store) store.TokenStore {
	s := &SqlTokenStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.Token{}, "Tokens").SetKeys(false, "Token")
	}

	return s
}

func (s SqlTokenStore) Save(token *model.Token) *model.AppError {
	if err := token.IsValid(); err != nil {
		return err
	}

	if err := s.GetMaster().Insert(token); err != nil {
		return model.NewAppError("SqlTokenStore.Save", "store.sql_recover.save.app_error", nil, "", http.StatusInternalServerError)
	}

	return nil
}

func (s SqlTokenStore) GetByToken(tokenString string) (*model.Token, *model.AppError) {
	token := &model.Token{}

	if err := s.GetReplica().SelectOne(token, "SELECT * FROM Tokens WHERE Token = :Token", map[string]interface{}{"Token": tokenString}); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlTokenStore.GetByToken", "store.sql_recover.get_by_code.app_error", nil, err.Error(), http.StatusBadRequest)
		}

		return nil, model.NewAppError("SqlTokenStore.GetByToken", "store.sql_recover.get_by_code.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return token, nil
}

func (s SqlTokenStore) Delete(token string) *model.AppError {
	if _, err := s.GetMaster().Exec("DELETE FROM Tokens WHERE Token =  :Token", map[string]interface{}{"Token": token}); err != nil {
		return model.NewAppError("SqlTokenStore.Delete", "store.sql_recover.delete.app_error", nil, "", http.StatusInternalServerError)
	}
	return nil
}
