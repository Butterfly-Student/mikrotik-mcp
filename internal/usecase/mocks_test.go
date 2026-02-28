package usecase

import (
	"context"

	"github.com/stretchr/testify/mock"
	"mikrotik-mcp/domain/dto"
	"mikrotik-mcp/domain/entity"
)

// ─── IP Pool Mock ────────────────────────────────────────────────────────────

type mockIPPoolRepo struct{ mock.Mock }

func (m *mockIPPoolRepo) GetAll(ctx context.Context) ([]entity.IPPool, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.IPPool), args.Error(1)
}

func (m *mockIPPoolRepo) GetUsed(ctx context.Context) ([]entity.IPPoolUsed, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.IPPoolUsed), args.Error(1)
}

func (m *mockIPPoolRepo) GetByName(ctx context.Context, name string) (*entity.IPPool, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.IPPool), args.Error(1)
}

func (m *mockIPPoolRepo) Create(ctx context.Context, req dto.CreateIPPoolRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *mockIPPoolRepo) Update(ctx context.Context, req dto.UpdateIPPoolRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *mockIPPoolRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// ─── Firewall Mock ───────────────────────────────────────────────────────────

type mockFirewallRepo struct{ mock.Mock }

func (m *mockFirewallRepo) GetAll(ctx context.Context) ([]entity.FirewallRule, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.FirewallRule), args.Error(1)
}

func (m *mockFirewallRepo) Create(ctx context.Context, req dto.CreateFirewallRuleRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *mockFirewallRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockFirewallRepo) Toggle(ctx context.Context, id string, disabled bool) error {
	return m.Called(ctx, id, disabled).Error(0)
}

// ─── Interface Mock ──────────────────────────────────────────────────────────

type mockInterfaceRepo struct{ mock.Mock }

func (m *mockInterfaceRepo) GetAll(ctx context.Context) ([]entity.NetworkInterface, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.NetworkInterface), args.Error(1)
}

func (m *mockInterfaceRepo) StartTrafficMonitor(ctx context.Context, iface string, ch chan<- entity.TrafficStat) error {
	args := m.Called(ctx, iface, ch)
	return args.Error(0)
}

func (m *mockInterfaceRepo) StopTrafficMonitor(ctx context.Context, iface string) error {
	return m.Called(ctx, iface).Error(0)
}

// ─── Hotspot Mock ────────────────────────────────────────────────────────────

type mockHotspotRepo struct{ mock.Mock }

func (m *mockHotspotRepo) GetServers(ctx context.Context) ([]entity.HotspotServer, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.HotspotServer), args.Error(1)
}

func (m *mockHotspotRepo) GetUsers(ctx context.Context) ([]entity.HotspotUser, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.HotspotUser), args.Error(1)
}

func (m *mockHotspotRepo) GetActiveUsers(ctx context.Context) ([]entity.HotspotActive, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.HotspotActive), args.Error(1)
}

func (m *mockHotspotRepo) AddUser(ctx context.Context, req dto.CreateHotspotUserRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *mockHotspotRepo) DeleteUser(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockHotspotRepo) KickActiveUser(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// ─── Queue Mock ──────────────────────────────────────────────────────────────

type mockQueueRepo struct{ mock.Mock }

func (m *mockQueueRepo) GetAllSimple(ctx context.Context) ([]entity.SimpleQueue, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.SimpleQueue), args.Error(1)
}

func (m *mockQueueRepo) GetAllTree(ctx context.Context) ([]entity.QueueTree, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.QueueTree), args.Error(1)
}

func (m *mockQueueRepo) CreateSimple(ctx context.Context, req dto.CreateSimpleQueueRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *mockQueueRepo) CreateTree(ctx context.Context, req dto.CreateQueueTreeRequest) error {
	return m.Called(ctx, req).Error(0)
}

func (m *mockQueueRepo) DeleteSimple(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockQueueRepo) DeleteTree(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// ─── System Mock ─────────────────────────────────────────────────────────────

type mockSystemRepo struct{ mock.Mock }

func (m *mockSystemRepo) GetResource(ctx context.Context) (*entity.SystemResource, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.SystemResource), args.Error(1)
}

func (m *mockSystemRepo) GetLogs(ctx context.Context, req dto.GetLogsRequest) ([]entity.SystemLog, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]entity.SystemLog), args.Error(1)
}

func (m *mockSystemRepo) GetIdentity(ctx context.Context) (*entity.SystemIdentity, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.SystemIdentity), args.Error(1)
}

func (m *mockSystemRepo) Reboot(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
