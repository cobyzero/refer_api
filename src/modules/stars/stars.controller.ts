import { Controller, Get, Post, Body, UseGuards, Request } from '@nestjs/common';
import { StarsService } from './stars.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';

@Controller('stars')
@UseGuards(JwtAuthGuard)
export class StarsController {
  constructor(private readonly starsService: StarsService) {}

  @Get('balance')
  async getBalance(@Request() req: any) {
    return this.starsService.getBalance(req.user.userId);
  }

  @Get('history')
  async getHistory(@Request() req: any) {
    return this.starsService.getHistory(req.user.userId);
  }

  @Get('rewards')
  async getRewards() {
    return this.starsService.getRewards();
  }

  @Get('redeemed')
  async getRedeemed(@Request() req: any) {
    return this.starsService.getRedeemedRewards(req.user.userId);
  }

  @Post('redeem')
  async redeem(@Request() req: any, @Body('rewardId') rewardId: string) {
    return this.starsService.redeemReward(req.user.userId, rewardId);
  }
}
