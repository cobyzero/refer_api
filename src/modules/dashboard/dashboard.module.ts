import { Module } from '@nestjs/common';
import { DashboardService } from './dashboard.service';
import { DashboardController } from './dashboard.controller';
import { UsersModule } from '../users/users.module';
import { StarsModule } from '../stars/stars.module';
import { OrdersModule } from '../orders/orders.module';

@Module({
  imports: [UsersModule, StarsModule, OrdersModule],
  providers: [DashboardService],
  controllers: [DashboardController],
})
export class DashboardModule {}
