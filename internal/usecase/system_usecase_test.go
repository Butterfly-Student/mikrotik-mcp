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

func newSystemUC(repo *mockSystemRepo) *SystemUseCase {
	return NewSystemUseCase(repo, zap.NewNop())
}

func TestGetResource_Success(t *testing.T) {
	repo := &mockSystemRepo{}
	resource := &entity.SystemResource{
		Uptime:        "1d 2h",
		Version:       "7.12",
		CPULoad:       15,
		CPUCount:      2,
		FreeMemory:    64 * 1024 * 1024,  // 64 MB
		TotalMemory:   256 * 1024 * 1024, // 256 MB
		FreeHDDSpace:  512 * 1024 * 1024, // 512 MB
		TotalHDDSpace: 2 * 1024 * 1024 * 1024,
		BoardName:     "RB750Gr3",
		Platform:      "MikroTik",
	}
	repo.On("GetResource", context.Background()).Return(resource, nil)

	resp, err := newSystemUC(repo).GetResource(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "1d 2h", resp.Uptime)
	assert.Equal(t, 15, resp.CPULoad)
	assert.Equal(t, "64.00 MB", resp.FreeMemory)
	assert.Equal(t, "256.00 MB", resp.TotalMemory)
	assert.Equal(t, "512.00 MB", resp.FreeHDDSpace)
	assert.Equal(t, "2.00 GB", resp.TotalHDDSpace)
	assert.Equal(t, "RB750Gr3", resp.BoardName)
	repo.AssertExpectations(t)
}

func TestGetResource_RepoError(t *testing.T) {
	repo := &mockSystemRepo{}
	repo.On("GetResource", context.Background()).Return(nil, errors.New("no route to host"))

	_, err := newSystemUC(repo).GetResource(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no route to host")
	repo.AssertExpectations(t)
}

func TestGetLogs_Success(t *testing.T) {
	repo := &mockSystemRepo{}
	logReq := dto.GetLogsRequest{Limit: 5}
	logs := make([]entity.SystemLog, 5)
	for i := range logs {
		logs[i] = entity.SystemLog{ID: "*1", Time: "jan/01 00:00:00", Topics: "system", Message: "test"}
	}
	repo.On("GetLogs", context.Background(), logReq).Return(logs, nil)

	resp, err := newSystemUC(repo).GetLogs(context.Background(), logReq)

	require.NoError(t, err)
	assert.Equal(t, 5, resp.Total)
	repo.AssertExpectations(t)
}

func TestGetLogs_DefaultLimit(t *testing.T) {
	repo := &mockSystemRepo{}
	// When Limit=0, usecase sets it to 50 before calling repo.
	expectedReq := dto.GetLogsRequest{Limit: 50}
	repo.On("GetLogs", context.Background(), expectedReq).Return([]entity.SystemLog{}, nil)

	_, err := newSystemUC(repo).GetLogs(context.Background(), dto.GetLogsRequest{Limit: 0})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetIdentity_Success(t *testing.T) {
	repo := &mockSystemRepo{}
	repo.On("GetIdentity", context.Background()).Return(&entity.SystemIdentity{Name: "my-router"}, nil)

	resp, err := newSystemUC(repo).GetIdentity(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "my-router", resp.Name)
	repo.AssertExpectations(t)
}

func TestReboot_Success(t *testing.T) {
	repo := &mockSystemRepo{}
	repo.On("Reboot", context.Background()).Return(nil)

	err := newSystemUC(repo).Reboot(context.Background())

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestReboot_Error(t *testing.T) {
	repo := &mockSystemRepo{}
	repo.On("Reboot", context.Background()).Return(errors.New("permission denied"))

	err := newSystemUC(repo).Reboot(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
	repo.AssertExpectations(t)
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
	}

	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, formatBytes(tc.input))
		})
	}
}
