package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

func newInterfaceUC(repo *mockInterfaceRepo) *InterfaceUseCase {
	return NewInterfaceUseCase(repo, zap.NewNop())
}

func TestListInterfaces_Success(t *testing.T) {
	repo := &mockInterfaceRepo{}
	repo.On("GetAll", context.Background()).Return([]entity.NetworkInterface{
		{ID: "*1", Name: "ether1", Type: "ether", Running: true},
		{ID: "*2", Name: "ether2", Type: "ether", Running: false},
		{ID: "*3", Name: "wlan1", Type: "wlan", Running: true},
	}, nil)

	resp, err := newInterfaceUC(repo).ListInterfaces(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, "ether1", resp.Interfaces[0].Name)
	assert.True(t, resp.Interfaces[0].Running)
	assert.False(t, resp.Interfaces[1].Running)
	repo.AssertExpectations(t)
}

func TestListInterfaces_RepoError(t *testing.T) {
	repo := &mockInterfaceRepo{}
	repo.On("GetAll", context.Background()).Return([]entity.NetworkInterface{}, errors.New("connection lost"))

	_, err := newInterfaceUC(repo).ListInterfaces(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
	repo.AssertExpectations(t)
}

func TestWatchTraffic_EmptyInterface(t *testing.T) {
	repo := &mockInterfaceRepo{}
	req := dto.WatchTrafficRequest{Interface: "", Seconds: 5}

	_, err := newInterfaceUC(repo).WatchTraffic(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "interface name is required")
	repo.AssertExpectations(t)
}

func TestWatchTraffic_InvalidSeconds_Zero(t *testing.T) {
	repo := &mockInterfaceRepo{}
	req := dto.WatchTrafficRequest{Interface: "ether1", Seconds: 0}

	_, err := newInterfaceUC(repo).WatchTraffic(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "seconds must be greater than 0")
	repo.AssertExpectations(t)
}

func TestWatchTraffic_InvalidSeconds_TooHigh(t *testing.T) {
	repo := &mockInterfaceRepo{}
	req := dto.WatchTrafficRequest{Interface: "ether1", Seconds: 61}

	_, err := newInterfaceUC(repo).WatchTraffic(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "seconds must not exceed 60")
	repo.AssertExpectations(t)
}

func TestWatchTraffic_Success(t *testing.T) {
	repo := &mockInterfaceRepo{}

	// StartTrafficMonitor sends one sample into the channel and returns nil.
	repo.On("StartTrafficMonitor", mock.Anything, "ether1", mock.Anything).
		Run(func(args mock.Arguments) {
			ch := args.Get(2).(chan<- entity.TrafficStat)
			ch <- entity.TrafficStat{
				Interface:       "ether1",
				RxBitsPerSecond: 1_000_000,
				TxBitsPerSecond: 500_000,
				Timestamp:       time.Now(),
			}
		}).
		Return(nil)

	// StopTrafficMonitor is called via deferred cleanup inside WatchTraffic.
	repo.On("StopTrafficMonitor", mock.Anything, "ether1").Return(nil)

	req := dto.WatchTrafficRequest{Interface: "ether1", Seconds: 1}
	resp, err := newInterfaceUC(repo).WatchTraffic(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "ether1", resp.Interface)
	assert.Equal(t, 1, resp.Duration)
	assert.NotEmpty(t, resp.Samples)
	assert.Equal(t, int64(1_000_000), resp.Samples[0].RxBps)
	repo.AssertExpectations(t)
}
