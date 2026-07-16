import { Controller, Get, UseGuards, Request } from '@nestjs/common';
import { DashboardService } from './dashboard.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { Roles } from '../auth/decorators/roles.decorator';
import { RolesGuard } from '../auth/guards/roles.guard';

@Controller('dashboard')
@UseGuards(JwtAuthGuard, RolesGuard)
export class DashboardController {
  constructor(private readonly dashboardService: DashboardService) { }

  @Get('summary')
  async getSummary(@Request() req: any) {
    return this.dashboardService.getSummary(req.user.userId);
  }

  @Roles('ADMIN')
  @Get('admin/summary')
  async getAdminSummary() {
    return this.dashboardService.getAdminSummary();
  }
}
