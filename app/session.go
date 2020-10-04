package app

import (
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
)

func (a *App) CreateSession(session *model.Session) (*model.Session, *model.AppError) {
	session.Token = ""

	session, err := a.Srv.Store.Session().Save(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (a *App) GetSession(token string) (*model.Session, *model.AppError) {
	var session *model.Session
	var err *model.AppError

	// TODO: まずセッションL1キャッシュから取得を試みる

	if session, err = a.Srv.Store.Session().Get(token); err == nil {
		if session != nil {
			if session.Token != token {
				return nil, model.NewAppError("GetSession", "api.context.invalid_token.error", map[string]interface{}{"Token": token, "Error": ""}, "", http.StatusUnauthorized)
			}
		}
	} else if err.StatusCode == http.StatusInternalServerError {
		return nil, err
	}

	if session == nil || session.IsExpired() {
		return nil, model.NewAppError("GetSession", "api.context.invalid_token.error", map[string]interface{}{"Token": token}, "", http.StatusUnauthorized)
	}

	return session, nil
}

func (a *App) RevokeSessionById(sessionId string) *model.AppError {
	session, err := a.Srv.Store.Session().Get(sessionId)
	if err != nil {
		err.StatusCode = http.StatusBadRequest
		return err
	}
	return a.RevokeSession(session)
}

func (a *App) RevokeSession(session *model.Session) *model.AppError {
	if session.IsOAuth {
		// sessionだけでなく関連するaccessDataも削除
		if err := a.RevokeAccessToken(session.Token); err != nil {
			return err
		}
	} else {
		if err := a.Srv.Store.Session().Remove(session.Id); err != nil {
			return err
		}
	}

	a.ClearSessionCacheForUser(session.UserId)

	return nil
}

func (a *App) RevokeAllSessions(userId string) *model.AppError {
	sessions, err := a.Srv.Store.Session().GetSessions(userId)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.IsOAuth {
			a.RevokeAccessToken(session.Token)
		} else {
			if err := a.Srv.Store.Session().Remove(session.Id); err != nil {
				return err
			}
		}
	}

	a.ClearSessionCacheForUser(userId)

	return nil
}

// 自サーバーのL1キャッシュ(user session)を削除
func (a *App) ClearLocalSessionCacheForUser(userId string) {
	if keys, err := a.Srv.sessionCache.Keys(); err == nil {
		var session *model.Session
		for _, key := range keys {
			if err := a.Srv.sessionCache.Get(key, &session); err == nil {
				if session.UserId == userId {
					a.Srv.sessionCache.Remove(key)
				}
			}
		}
	}

	// 関連するwebConnが持つセッション情報も削除
	a.InvalidateWebConnSessionCacheForUser(userId)
}

// 自サーバーのL1キャッシュ(user session)を削除し、
// (自サーバー以外の)クラスタ全体にも周知する。
func (a *App) ClearSessionCacheForUser(userId string) {
	a.ClearLocalSessionCacheForUser(userId)

	if a.Srv.Cluster != nil {
		cm := &model.ClusterMessage{
			OmitCluster: a.Srv.clusterId,
			Event:       model.CLUSTER_EVENT_CLEAR_SESSION_CACHE_FOR_USER,
			Data:        userId,
		}

		a.Srv.Cluster.SendClusterMessage(cm)
	}
}
