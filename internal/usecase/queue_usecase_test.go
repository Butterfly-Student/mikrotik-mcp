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

func newQueueUC(repo *mockQueueRepo) *QueueUseCase {
	return NewQueueUseCase(repo, zap.NewNop())
}

func TestListSimpleQueues_Success(t *testing.T) {
	repo := &mockQueueRepo{}
	repo.On("GetAllSimple", context.Background()).Return([]entity.SimpleQueue{
		{ID: "*1", Name: "q-alice", Target: "192.168.1.10/32", MaxLimit: "10M/10M"},
		{ID: "*2", Name: "q-bob", Target: "192.168.1.11/32", MaxLimit: "5M/5M"},
	}, nil)

	resp, err := newQueueUC(repo).ListSimpleQueues(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "q-alice", resp.Queues[0].Name)
	repo.AssertExpectations(t)
}

func TestListSimpleQueues_RepoError(t *testing.T) {
	repo := &mockQueueRepo{}
	repo.On("GetAllSimple", context.Background()).Return([]entity.SimpleQueue{}, errors.New("timeout"))

	_, err := newQueueUC(repo).ListSimpleQueues(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	repo.AssertExpectations(t)
}

func TestListTreeQueues_Success(t *testing.T) {
	repo := &mockQueueRepo{}
	repo.On("GetAllTree", context.Background()).Return([]entity.QueueTree{
		{ID: "*1", Name: "root", Parent: "global"},
		{ID: "*2", Name: "child", Parent: "root"},
	}, nil)

	resp, err := newQueueUC(repo).ListTreeQueues(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "global", resp.Queues[0].Parent)
	repo.AssertExpectations(t)
}

func TestAddSimpleQueue_Success(t *testing.T) {
	repo := &mockQueueRepo{}
	req := dto.CreateSimpleQueueRequest{Name: "q-new", Target: "192.168.1.50/32", MaxLimit: "20M/20M"}
	repo.On("CreateSimple", context.Background(), req).Return(nil)

	err := newQueueUC(repo).AddSimpleQueue(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAddSimpleQueue_EmptyName(t *testing.T) {
	repo := &mockQueueRepo{}
	req := dto.CreateSimpleQueueRequest{Name: "", Target: "192.168.1.50/32"}

	err := newQueueUC(repo).AddSimpleQueue(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue name is required")
	repo.AssertExpectations(t)
}

func TestAddSimpleQueue_EmptyTarget(t *testing.T) {
	repo := &mockQueueRepo{}
	req := dto.CreateSimpleQueueRequest{Name: "q-new", Target: ""}

	err := newQueueUC(repo).AddSimpleQueue(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue target is required")
	repo.AssertExpectations(t)
}

func TestAddTreeQueue_Success(t *testing.T) {
	repo := &mockQueueRepo{}
	req := dto.CreateQueueTreeRequest{Name: "branch", Parent: "global", MaxLimit: "100M"}
	repo.On("CreateTree", context.Background(), req).Return(nil)

	err := newQueueUC(repo).AddTreeQueue(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAddTreeQueue_EmptyName(t *testing.T) {
	repo := &mockQueueRepo{}
	req := dto.CreateQueueTreeRequest{Name: "", Parent: "global"}

	err := newQueueUC(repo).AddTreeQueue(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue name is required")
	repo.AssertExpectations(t)
}

func TestAddTreeQueue_EmptyParent(t *testing.T) {
	repo := &mockQueueRepo{}
	req := dto.CreateQueueTreeRequest{Name: "branch", Parent: ""}

	err := newQueueUC(repo).AddTreeQueue(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent is required")
	repo.AssertExpectations(t)
}

func TestDeleteSimpleQueue_Success(t *testing.T) {
	repo := &mockQueueRepo{}
	repo.On("DeleteSimple", context.Background(), "*4").Return(nil)

	err := newQueueUC(repo).DeleteSimpleQueue(context.Background(), "*4")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteSimpleQueue_EmptyID(t *testing.T) {
	repo := &mockQueueRepo{}

	err := newQueueUC(repo).DeleteSimpleQueue(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue id is required")
	repo.AssertExpectations(t)
}

func TestDeleteTreeQueue_Success(t *testing.T) {
	repo := &mockQueueRepo{}
	repo.On("DeleteTree", context.Background(), "*6").Return(nil)

	err := newQueueUC(repo).DeleteTreeQueue(context.Background(), "*6")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}
