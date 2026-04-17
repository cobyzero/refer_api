import { Injectable } from '@nestjs/common';
import { UsersService } from '../users/users.service';
import { StarsService } from '../stars/stars.service';

@Injectable()
export class DashboardService {
  constructor(
    private readonly usersService: UsersService,
    private readonly starsService: StarsService,
  ) {}

  async getSummary(userId: string) {
    const user = await this.usersService.findById(userId);
    const stars = await this.starsService.getBalance(userId);
    const rewards = await this.starsService.getRewards();
    const history = await this.starsService.getHistory(userId);
    
    return {
      user: {
        name: user?.name,
        photoUrl: user?.photoUrl,
        stars: stars,
      },
      stars: stars,
      rewards: rewards,
      recentActivity: history.slice(0, 5).map(h => ({
        id: h.id,
        type: h.type,
        title: h.title,
        stars: h.stars,
        date: typeof h.date === 'string' ? h.date : (h.date ? h.date.toISOString() : new Date().toISOString()),
      })),
      nextRewardProgress: stars / 100 > 1 ? 1 : stars / 100,
    };
  }
}
