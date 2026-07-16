import { Controller, Get, Post, Patch, Delete, Param, Body, UseGuards, Request } from '@nestjs/common';
import { StarsService } from './stars.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Roles } from '../auth/decorators/roles.decorator';

@Controller('stars')
export class StarsController {
  constructor(private readonly starsService: StarsService) {}

  @UseGuards(JwtAuthGuard)
  @Get('balance')
  async getBalance(@Request() req: any) {
    return this.starsService.getBalance(req.user.userId);
  }

  @UseGuards(JwtAuthGuard)
  @Get('history')
  async getHistory(@Request() req: any) {
    return this.starsService.getHistory(req.user.userId);
  }

  @Get('rewards')
  async getRewards() {
    return this.starsService.getRewards();
  }

  @UseGuards(JwtAuthGuard)
  @Get('redeemed')
  async getRedeemed(@Request() req: any) {
    return this.starsService.getRedeemedRewards(req.user.userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('redeem')
  async redeem(@Request() req: any, @Body('rewardId') rewardId: string) {
    return this.starsService.redeemReward(req.user.userId, rewardId);
  }

  @Roles('ADMIN')
  @UseGuards(JwtAuthGuard, RolesGuard)
  @Get('admin/transactions')
  async getAdminTransactions() {
    return this.starsService.getAllTransactions();
  }

  @Roles('ADMIN')
  @UseGuards(JwtAuthGuard, RolesGuard)
  @Post('admin/rewards')
  async createReward(@Body() data: any) {
    return this.starsService.createReward(data);
  }

  @Roles('ADMIN')
  @UseGuards(JwtAuthGuard, RolesGuard)
  @Patch('admin/rewards/:id')
  async updateReward(@Param('id') id: string, @Body() data: any) {
    return this.starsService.updateReward(id, data);
  }

  @Roles('ADMIN')
  @UseGuards(JwtAuthGuard, RolesGuard)
  @Delete('admin/rewards/:id')
  async deleteReward(@Param('id') id: string) {
    await this.starsService.deleteReward(id);
    return { success: true };
  }
}
