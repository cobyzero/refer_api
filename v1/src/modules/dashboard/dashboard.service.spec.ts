import { DashboardService } from './dashboard.service';

describe('DashboardService', () => {
  const usersService = { findById: jest.fn(), findAll: jest.fn() };
  const starsService = {
    getBalance: jest.fn(),
    getRewards: jest.fn(),
    getHistory: jest.fn(),
    getTotalStarsAwarded: jest.fn(),
    getTotalStarsRedeemed: jest.fn(),
  };
  const ordersService = { findAll: jest.fn() };
  let service: DashboardService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new DashboardService(usersService as any, starsService as any, ordersService as any);
  });

  it('builds user summary', async () => {
    usersService.findById.mockResolvedValue({ name: 'Ana', photoUrl: 'p' });
    starsService.getBalance.mockResolvedValue(80);
    starsService.getRewards.mockResolvedValue([{ id: 'r1' }]);
    starsService.getHistory.mockResolvedValue([{ id: 'h1', type: 'earn', title: 'x', stars: 10, date: new Date() }]);

    const result = await service.getSummary('u1');
    expect(result.user.name).toBe('Ana');
    expect(result.nextRewardProgress).toBe(0.8);
  });
});

