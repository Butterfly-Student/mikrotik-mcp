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

func newIPPoolUC(repo *mockIPPoolRepo) *IPPoolUseCase {
	return NewIPPoolUseCase(repo, zap.NewNop())
}

func TestListPools_Success(t *testing.T) {
	repo := &mockIPPoolRepo{}
	repo.On("GetAll", context.Background()).Return([]entity.IPPool{
		{ID: "*1", Name: "pool-a", Ranges: "192.168.1.100-192.168.1.200"},
		{ID: "*2", Name: "pool-b", Ranges: "10.0.0.1-10.0.0.254"},
	}, nil)

	resp, err := newIPPoolUC(repo).ListPools(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "pool-a", resp.Pools[0].Name)
	assert.Equal(t, "pool-b", resp.Pools[1].Name)
	repo.AssertExpectations(t)
}

func TestListPools_RepoError(t *testing.T) {
	repo := &mockIPPoolRepo{}
	repo.On("GetAll", context.Background()).Return([]entity.IPPool{}, errors.New("db error"))

	_, err := newIPPoolUC(repo).ListPools(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
	repo.AssertExpectations(t)
}

func TestListUsed_Success(t *testing.T) {
	repo := &mockIPPoolRepo{}
	repo.On("GetUsed", context.Background()).Return([]entity.IPPoolUsed{
		{Pool: "pool-a", Address: "192.168.1.101", Owner: "pc1"},
	}, nil)

	resp, err := newIPPoolUC(repo).ListUsed(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "pool-a", resp.Used[0].Pool)
	repo.AssertExpectations(t)
}

func TestCreatePool_Success(t *testing.T) {
	repo := &mockIPPoolRepo{}
	req := dto.CreateIPPoolRequest{Name: "mypool", Ranges: "192.168.1.1-192.168.1.254"}
	repo.On("Create", context.Background(), req).Return(nil)

	err := newIPPoolUC(repo).CreatePool(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreatePool_EmptyName(t *testing.T) {
	repo := &mockIPPoolRepo{}
	req := dto.CreateIPPoolRequest{Name: "", Ranges: "192.168.1.1-192.168.1.254"}

	err := newIPPoolUC(repo).CreatePool(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
	repo.AssertExpectations(t)
}

func TestCreatePool_EmptyRanges(t *testing.T) {
	repo := &mockIPPoolRepo{}
	req := dto.CreateIPPoolRequest{Name: "mypool", Ranges: ""}

	err := newIPPoolUC(repo).CreatePool(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ranges is required")
	repo.AssertExpectations(t)
}

func TestCreatePool_RepoError(t *testing.T) {
	repo := &mockIPPoolRepo{}
	req := dto.CreateIPPoolRequest{Name: "mypool", Ranges: "192.168.1.1-192.168.1.254"}
	repo.On("Create", context.Background(), req).Return(errors.New("router offline"))

	err := newIPPoolUC(repo).CreatePool(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "router offline")
	repo.AssertExpectations(t)
}

func TestUpdatePool_Success(t *testing.T) {
	repo := &mockIPPoolRepo{}
	req := dto.UpdateIPPoolRequest{ID: "*1", Ranges: "10.0.0.1-10.0.0.100"}
	repo.On("Update", context.Background(), req).Return(nil)

	err := newIPPoolUC(repo).UpdatePool(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdatePool_EmptyID(t *testing.T) {
	repo := &mockIPPoolRepo{}
	req := dto.UpdateIPPoolRequest{ID: ""}

	err := newIPPoolUC(repo).UpdatePool(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
	repo.AssertExpectations(t)
}

func TestDeletePool_Success(t *testing.T) {
	repo := &mockIPPoolRepo{}
	repo.On("Delete", context.Background(), "*3").Return(nil)

	err := newIPPoolUC(repo).DeletePool(context.Background(), "*3")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeletePool_EmptyID(t *testing.T) {
	repo := &mockIPPoolRepo{}

	err := newIPPoolUC(repo).DeletePool(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
	repo.AssertExpectations(t)
}
