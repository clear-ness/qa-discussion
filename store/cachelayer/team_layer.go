package cachelayer

import (
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

// キャッシュすべきは、
// よく参照される前提で
// getSingle系 getCount系 複数だがほぼ不変で有限個
// 辺りが良さそう。
// チーム内ユーザー、グループ内ユーザー、コレクションのリスト
// など？
type CacheTeamStore struct {
	store.TeamStore
	rootStore *CacheStore
}

func (s CacheTeamStore) GetActiveMemberCount(teamId string) (int64, *model.AppError) {
	if count := s.rootStore.readCache(teamId); count != nil {
		return count.(int64), nil
	}

	count, err := s.TeamStore.GetActiveMemberCount(teamId)
	if err != nil {
		return 0, err
	}

	s.rootStore.addToCache(teamId, count)

	return count, nil
}

// TODO: team user add, removeした際に調整が必要
