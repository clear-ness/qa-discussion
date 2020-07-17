package sqlstore

import (
	"database/sql"
	"net/http"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type SqlFileInfoStore struct {
	store.Store
}

func NewSqlFileInfoStore(sqlStore store.Store) store.FileInfoStore {
	s := &SqlFileInfoStore{
		Store: sqlStore,
	}

	for _, db := range sqlStore.GetAllConns() {
		db.AddTableWithName(model.FileInfo{}, "FileInfo").SetKeys(false, "Id")
	}

	return s
}

func (s SqlFileInfoStore) Save(info *model.FileInfo) (*model.FileInfo, *model.AppError) {
	info.PreSave()
	if err := info.IsValid(); err != nil {
		return nil, err
	}

	if err := s.GetMaster().Insert(info); err != nil {
		return nil, model.NewAppError("SqlFileInfoStore.Save", "store.sql_file_info.save.app_error", nil, err.Error(), http.StatusInternalServerError)
	}

	return info, nil
}

func (s SqlFileInfoStore) Get(id string) (*model.FileInfo, *model.AppError) {
	info := &model.FileInfo{}

	if err := s.GetReplica().SelectOne(info,
		`SELECT
			*
		FROM
			FileInfo
		WHERE
			Id = :Id
			AND DeleteAt = 0`, map[string]interface{}{"Id": id}); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewAppError("SqlFileInfoStore.Get", "store.sql_file_info.get.app_error", nil, "id="+id+", "+err.Error(), http.StatusNotFound)
		}

		return nil, model.NewAppError("SqlFileInfoStore.Get", "store.sql_file_info.get.app_error", nil, "id="+id+", "+err.Error(), http.StatusInternalServerError)
	}

	return info, nil
}

func (s SqlFileInfoStore) AttachToPost(fileId, postId, userId string) *model.AppError {
	sqlResult, err := s.GetMaster().Exec(`
		UPDATE
			FileInfo
		SET
			PostId = :PostId
		WHERE
			Id = :Id
			AND PostId = ''
			AND UserId = :UserId
			AND DeleteAt = 0
	`, map[string]interface{}{
		"PostId": postId,
		"Id":     fileId,
		"UserId": userId,
	})
	if err != nil {
		return model.NewAppError("SqlFileInfoStore.AttachToPost",
			"store.sql_file_info.attach_to_post.app_error", nil, "post_id="+postId+", file_id="+fileId+", err="+err.Error(), http.StatusInternalServerError)
	}

	count, err := sqlResult.RowsAffected()
	if err != nil {
		return model.NewAppError("SqlFileInfoStore.AttachToPost",
			"store.sql_file_info.attach_to_post.app_error", nil, "post_id="+postId+", file_id="+fileId+", err="+err.Error(), http.StatusInternalServerError)
	} else if count == 0 {
		return model.NewAppError("SqlFileInfoStore.AttachToPost",
			"store.sql_file_info.attach_to_post.app_error", nil, "post_id="+postId+", file_id="+fileId, http.StatusBadRequest)
	}

	return nil
}

func (s SqlFileInfoStore) DeleteForPost(postId string) (string, *model.AppError) {
	if _, err := s.GetMaster().Exec(
		`UPDATE
				FileInfo
			SET
				DeleteAt = :DeleteAt
			WHERE
				PostId = :PostId`, map[string]interface{}{"DeleteAt": model.GetMillis(), "PostId": postId}); err != nil {
		return "", model.NewAppError("SqlFileInfoStore.DeleteForPost",
			"store.sql_file_info.delete_for_post.app_error", nil, "post_id="+postId+", err="+err.Error(), http.StatusInternalServerError)
	}

	return postId, nil
}
