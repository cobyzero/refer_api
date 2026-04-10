import { Injectable, NotFoundException, BadRequestException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Reward } from './entities/reward.entity';
import { StarTransaction } from './entities/star-transaction.entity';
import { UsersService } from '../users/users.service';

@Injectable()
export class StarsService {
  constructor(
    @InjectRepository(Reward)
    private readonly rewardRepository: Repository<Reward>,
    @InjectRepository(StarTransaction)
    private readonly transactionRepository: Repository<StarTransaction>,
    private readonly usersService: UsersService,
  ) {}

  async getBalance(userId: string) {
    const user = await this.usersService.findById(userId);
    return { stars: user?.stars || 0 };
  }

  async getHistory(userId: string) {
    return this.transactionRepository.find({
      where: { userId },
      order: { date: 'DESC' },
    });
  }

  async getRewards() {
    return this.rewardRepository.find({ where: { isActive: true } });
  }

  async redeemReward(userId: string, rewardId: string) {
    const user = await this.usersService.findById(userId);
    const reward = await this.rewardRepository.findOne({ where: { id: rewardId, isActive: true } });

    if (!user) throw new NotFoundException('User not found');
    if (!reward) throw new NotFoundException('Reward not found');

    if (user.stars < reward.starsRequired) {
      throw new BadRequestException('Not enough stars');
    }

    // Process redemption
    const newStars = user.stars - reward.starsRequired;
    await this.usersService.create({ ...user, stars: newStars });

    const transaction = this.transactionRepository.create({
      userId,
      type: 'redeem',
      title: reward.title,
      stars: -reward.starsRequired,
    });
    await this.transactionRepository.save(transaction);

    return {
      success: true,
      message: `Redeemed ${reward.title}`,
      remainingStars: newStars,
    };
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
