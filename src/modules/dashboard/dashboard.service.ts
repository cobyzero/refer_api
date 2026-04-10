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
    const rewards = await this.starsService.getRewards();
    const history = await this.starsService.getHistory(userId);
    
    return {
      user: {
        name: user?.name,
        photoUrl: user?.photoUrl,
        stars: user?.stars || 0,
      },
      stars: user?.stars || 0,
      rewards: rewards,
      recentActivity: history.slice(0, 5).map(h => ({
        id: h.id,
        type: h.type,
        title: h.title,
        stars: h.stars,
        date: h.date.toISOString(),
      })),
      nextRewardProgress: (user?.stars || 0) / 100 > 1 ? 1 : (user?.stars || 0) / 100,
    };
  }
}
