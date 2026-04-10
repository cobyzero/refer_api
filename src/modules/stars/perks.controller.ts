import { Controller, Get, UseGuards } from '@nestjs/common';
import { StarsService } from './stars.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';

@Controller('perks')
export class PerksController {
  constructor(private readonly starsService: StarsService) {}

  @Get()
  async getPerks() {
    return this.starsService.getPerks();
  }
}
