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

func newFirewallUC(repo *mockFirewallRepo) *FirewallUseCase {
	return NewFirewallUseCase(repo, zap.NewNop())
}

func TestListRules_Success(t *testing.T) {
	repo := &mockFirewallRepo{}
	repo.On("GetAll", context.Background()).Return([]entity.FirewallRule{
		{ID: "*1", Chain: "input", Action: "accept"},
		{ID: "*2", Chain: "forward", Action: "drop"},
		{ID: "*3", Chain: "output", Action: "accept"},
	}, nil)

	resp, err := newFirewallUC(repo).ListRules(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, "input", resp.Rules[0].Chain)
	repo.AssertExpectations(t)
}

func TestListRules_RepoError(t *testing.T) {
	repo := &mockFirewallRepo{}
	repo.On("GetAll", context.Background()).Return([]entity.FirewallRule{}, errors.New("api error"))

	_, err := newFirewallUC(repo).ListRules(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "api error")
	repo.AssertExpectations(t)
}

func TestCreateRule_Success(t *testing.T) {
	repo := &mockFirewallRepo{}
	req := dto.CreateFirewallRuleRequest{Chain: "input", Action: "drop", Protocol: "tcp", DstPort: "22"}
	repo.On("Create", context.Background(), req).Return(nil)

	err := newFirewallUC(repo).CreateRule(context.Background(), req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateRule_EmptyChain(t *testing.T) {
	repo := &mockFirewallRepo{}
	req := dto.CreateFirewallRuleRequest{Chain: "", Action: "drop"}

	err := newFirewallUC(repo).CreateRule(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain is required")
	repo.AssertExpectations(t)
}

func TestCreateRule_EmptyAction(t *testing.T) {
	repo := &mockFirewallRepo{}
	req := dto.CreateFirewallRuleRequest{Chain: "input", Action: ""}

	err := newFirewallUC(repo).CreateRule(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "action is required")
	repo.AssertExpectations(t)
}

func TestDeleteRule_Success(t *testing.T) {
	repo := &mockFirewallRepo{}
	repo.On("Delete", context.Background(), "*5").Return(nil)

	err := newFirewallUC(repo).DeleteRule(context.Background(), "*5")

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteRule_EmptyID(t *testing.T) {
	repo := &mockFirewallRepo{}

	err := newFirewallUC(repo).DeleteRule(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
	repo.AssertExpectations(t)
}

func TestToggleRule_Disable(t *testing.T) {
	repo := &mockFirewallRepo{}
	repo.On("Toggle", context.Background(), "*2", true).Return(nil)

	err := newFirewallUC(repo).ToggleRule(context.Background(), "*2", true)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestToggleRule_Enable(t *testing.T) {
	repo := &mockFirewallRepo{}
	repo.On("Toggle", context.Background(), "*2", false).Return(nil)

	err := newFirewallUC(repo).ToggleRule(context.Background(), "*2", false)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestToggleRule_EmptyID(t *testing.T) {
	repo := &mockFirewallRepo{}

	err := newFirewallUC(repo).ToggleRule(context.Background(), "", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
	repo.AssertExpectations(t)
}
