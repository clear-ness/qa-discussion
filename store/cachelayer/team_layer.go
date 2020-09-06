package cachelayer

import (
	"strconv"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
)

type CacheTeamStore struct {
	store.TeamStore
	rootStore *CacheStore
}

func (s CacheTeamStore) GetActiveMemberCount(teamId string) (int64, *model.AppError) {
	if countStr := s.rootStore.readCache(teamId); countStr != nil {
		if val, err := strconv.Atoi(*countStr); err == nil {
			return int64(val), nil
		}
	}

	count, err := s.TeamStore.GetActiveMemberCount(teamId)
	if err != nil {
		return 0, err
	}

	s.rootStore.addToCache(teamId, count, 0)

	return count, nil
}

func (s CacheTeamStore) SaveMember(member *model.TeamMember, maxUsersPerTeam int) (*model.TeamMember, *model.AppError) {
	member, err := s.TeamStore.SaveMember(member, maxUsersPerTeam)
	if err != nil {
		return nil, err
	}

	s.InvalidateMemberCount(member.TeamId)

	return member, nil
}

func (s CacheTeamStore) InvalidateMemberCount(teamId string) {
	s.rootStore.deleteCache([]string{teamId})
}

func (s CacheTeamStore) SaveMultipleMembers(members []*model.TeamMember, maxUsersPerTeam int) ([]*model.TeamMember, *model.AppError) {
	members, err := s.TeamStore.SaveMultipleMembers(members, maxUsersPerTeam)
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		s.InvalidateMemberCount(member.TeamId)
	}

	return members, nil
}

func (s CacheTeamStore) PermanentDelete(teamId string) *model.AppError {
	err := s.TeamStore.PermanentDelete(teamId)
	if err != nil {
		return err
	}

	s.InvalidateMemberCount(teamId)

	return nil
}

func (s CacheTeamStore) RemoveAllMembersByTeam(teamId string) *model.AppError {
	err := s.TeamStore.RemoveAllMembersByTeam(teamId)
	if err != nil {
		return err
	}

	s.InvalidateMemberCount(teamId)

	return nil
}

func (s CacheTeamStore) UpdateMultipleMembers(members []*model.TeamMember) ([]*model.TeamMember, *model.AppError) {
	members, err := s.TeamStore.UpdateMultipleMembers(members)
	if err != nil {
		return nil, err
	}

	for _, member := range members {
		s.InvalidateMemberCount(member.TeamId)
	}

	return members, nil
}

func (s CacheTeamStore) UpdateMember(member *model.TeamMember) (*model.TeamMember, *model.AppError) {
	member, err := s.TeamStore.UpdateMember(member)
	if err != nil {
		return nil, err
	}

	s.InvalidateMemberCount(member.TeamId)

	return member, nil
}
