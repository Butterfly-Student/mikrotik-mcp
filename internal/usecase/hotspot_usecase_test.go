package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

func newHotspotUC(repo *mockHotspotRepo) *HotspotUseCase {
	return NewHotspotUseCase(repo, zap.NewNop())
}

func TestListServers_Success(t *testing.T) {
	repo := &mockHotspotRepo{}
	repo.On("GetServers", context.Background()).Return([]entity.HotspotServer{
		{ID: "*1", Name: "hs1", Interface: "ether2"},
		{ID: "*2", Name: "hs2", Interface: "wlan1"},
	}, nil)

	resp, err := newHotspotUC(repo).ListServers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "hs1", resp.Servers[0].Name)
	repo.AssertExpectations(t)
}

func TestListUsers_Success(t *testing.T) {
	repo := &mockHotspotRepo{}
	repo.On("GetUsers", context.Background()).Return([]entity.HotspotUser{
		{ID: "*1", Name: "alice", Server: "hs1"},
		{ID: "*2", Name: "bob", Server: "hs1"},
	}, nil)

	resp, err := newHotspotUC(repo).ListUsers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "alice", resp.Users[0].Name)
	repo.AssertExpectations(t)
}

func TestListActiveUsers_Success(t *testing.T) {
	repo := &mockHotspotRepo{}
	repo.On("GetActiveUsers", context.Background()).Return([]entity.HotspotActive{
		{ID: "*1", User: "alice", Address: "192.168.1.10", Server: "hs1"},
	}, nil)

	resp, err := newHotspotUC(repo).ListActiveUsers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "alice", resp.Active[0].User)
	repo.AssertExpectations(t)
}

func TestAddUser_Success(t *testing.T) {
	repo := &mockHotspotRepo{}
	req := dto.CreateHotspotUserRequest{Name: "charlie", Password: "secret123"}
	repo.On("AddUser", context.Background(), req).Return(nil)

	err := newHotspotUC(repo).AddUser(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAddUser_EmptyName(t *testing.T) {
	repo := &mockHotspotRepo{}
	req := dto.CreateHotspotUserRequest{Name: "", Password: "secret123"}

	err := newHotspotUC(repo).AddUser(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
	repo.AssertExpectations(t)
}

func TestAddUser_EmptyPassword(t *testing.T) {
	repo := &mockHotspotRepo{}
	req := dto.CreateHotspotUserRequest{Name: "charlie", Password: ""}

	err := newHotspotUC(repo).AddUser(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
	repo.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	repo := &mockHotspotRepo{}
	repo.On("DeleteUser", context.Background(), "*3").Return(nil)

	err := newHotspotUC(repo).DeleteUser(context.Background(), "*3")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteUser_EmptyID(t *testing.T) {
	repo := &mockHotspotRepo{}

	err := newHotspotUC(repo).DeleteUser(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user id is required")
	repo.AssertExpectations(t)
}

func TestKickActiveUser_Success(t *testing.T) {
	repo := &mockHotspotRepo{}
	repo.On("KickActiveUser", context.Background(), "*7").Return(nil)

	err := newHotspotUC(repo).KickActiveUser(context.Background(), "*7")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestKickActiveUser_EmptyID(t *testing.T) {
	repo := &mockHotspotRepo{}

	err := newHotspotUC(repo).KickActiveUser(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "active session id is required")
	repo.AssertExpectations(t)
}

func TestListServers_RepoError(t *testing.T) {
	repo := &mockHotspotRepo{}
	repo.On("GetServers", context.Background()).Return([]entity.HotspotServer{}, errors.New("api error"))

	_, err := newHotspotUC(repo).ListServers(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "api error")
	repo.AssertExpectations(t)
}
