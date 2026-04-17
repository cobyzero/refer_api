import { Injectable, NotFoundException, BadRequestException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Reward } from './entities/reward.entity';
import { StarTransaction } from './entities/star-transaction.entity';
import { RedeemedReward } from './entities/redeemed-reward.entity';
import { UsersService } from '../users/users.service';

@Injectable()
export class StarsService {
  constructor(
    @InjectRepository(Reward)
    private readonly rewardRepository: Repository<Reward>,
    @InjectRepository(StarTransaction)
    private readonly transactionRepository: Repository<StarTransaction>,
    @InjectRepository(RedeemedReward)
    private readonly redeemedRewardRepository: Repository<RedeemedReward>,
    private readonly usersService: UsersService,
  ) {}

  async getBalance(userId: string): Promise<number> {
    const result = await this.transactionRepository
      .createQueryBuilder('transaction')
      .select('SUM(transaction.stars)', 'total')
      .where('transaction.userId = :userId', { userId })
      .getRawOne();
    
    return parseInt(result?.total || '0', 10);
  }

  async getHistory(userId: string) {
    return this.transactionRepository.find({
      where: { userId },
      order: { date: 'DESC' },
    });
  }

  async getRedeemedRewards(userId: string) {
    return this.redeemedRewardRepository.find({
      where: { userId },
      order: { createdAt: 'DESC' },
    });
  }

  async getRewards() {
    return this.rewardRepository.find({ where: { isActive: true } });
  }

  async redeemReward(userId: string, rewardId: string) {
    const currentStars = await this.getBalance(userId);
    const reward = await this.rewardRepository.findOne({ where: { id: rewardId, isActive: true } });

    if (!reward) throw new NotFoundException('Reward not found');

    if (currentStars < reward.starsRequired) {
      throw new BadRequestException('Not enough stars');
    }

    // 1. Create history transaction
    const transaction = this.transactionRepository.create({
      userId,
      type: 'redeem',
      title: reward.title,
      stars: -reward.starsRequired,
    });
    await this.transactionRepository.save(transaction);

    // 2. Create the actual item to be claimed
    const redeemedReward = this.redeemedRewardRepository.create({
      userId,
      rewardId: reward.id,
      rewardTitle: reward.title,
    });
    await this.redeemedRewardRepository.save(redeemedReward);

    return {
      success: true,
      message: `Redeemed ${reward.title}`,
      remainingStars: currentStars - reward.starsRequired,
      redeemedReward,
    };
  }

  async rewardStars(userId: string, stars: number, reason: string) {
    if (stars <= 0) return;

    const currentStars = await this.getBalance(userId);

    const transaction = this.transactionRepository.create({
      userId,
      type: 'earn',
      title: reason,
      stars: stars,
    });
    await this.transactionRepository.save(transaction);
    
    return {
      success: true,
      newBalance: currentStars + stars,
    };
  }

  async claimReward(redeemedRewardId: string, orderId: string) {
    const redeemedReward = await this.redeemedRewardRepository.findOne({ where: { id: redeemedRewardId } });
    if (!redeemedReward) throw new NotFoundException('Redeemed reward not found');
    if (redeemedReward.isClaimed) throw new BadRequestException('Reward already claimed');

    redeemedReward.isClaimed = true;
    redeemedReward.orderId = orderId;
    await this.redeemedRewardRepository.save(redeemedReward);
    
    return redeemedReward;
  }

  async getPerks() {
    return [
      {
        id: 'birthday',
        title: 'Birthday Treat',
        description: 'Get a free drink on your birthday!',
        status: 'active',
      },
      {
        id: 'double-stars',
        title: 'Double Star Day',
        description: 'Earn 2x stars every Tuesday.',
        status: 'active',
      }
    ];
  }
}
