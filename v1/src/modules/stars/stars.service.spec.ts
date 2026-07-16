import { BadRequestException, NotFoundException } from '@nestjs/common';
import { StarsService } from './stars.service';

describe('StarsService', () => {
  const rewardRepository = { find: jest.fn(), findOne: jest.fn() };
  const transactionRepository = {
    find: jest.fn(),
    create: jest.fn(),
    save: jest.fn(),
    createQueryBuilder: jest.fn(),
  };
  const redeemedRewardRepository = { find: jest.fn(), findOne: jest.fn(), create: jest.fn(), save: jest.fn() };
  const usersService = {};

  let service: StarsService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new StarsService(
      rewardRepository as any,
      transactionRepository as any,
      redeemedRewardRepository as any,
      usersService as any,
    );
  });

  it('returns 0 balance when query has no total', async () => {
    transactionRepository.createQueryBuilder.mockReturnValue({
      select: () => ({
        where: () => ({
          getRawOne: async () => ({}),
        }),
      }),
    });

    await expect(service.getBalance('u1')).resolves.toBe(0);
  });

  it('throws on redeem when reward does not exist', async () => {
    jest.spyOn(service, 'getBalance').mockResolvedValue(100);
    rewardRepository.findOne.mockResolvedValue(null);
    await expect(service.redeemReward('u1', 'r1')).rejects.toBeInstanceOf(NotFoundException);
  });

  it('throws on redeem when stars are insufficient', async () => {
    jest.spyOn(service, 'getBalance').mockResolvedValue(10);
    rewardRepository.findOne.mockResolvedValue({ id: 'r1', isActive: true, starsRequired: 25 });
    await expect(service.redeemReward('u1', 'r1')).rejects.toBeInstanceOf(BadRequestException);
  });
});

