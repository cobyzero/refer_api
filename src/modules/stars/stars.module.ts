import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { StarsService } from './stars.service';
import { StarsController } from './stars.controller';
import { PerksController } from './perks.controller';
import { Reward } from './entities/reward.entity';
import { StarTransaction } from './entities/star-transaction.entity';
import { UsersModule } from '../users/users.module';

@Module({
  imports: [
    TypeOrmModule.forFeature([Reward, StarTransaction]),
    UsersModule,
  ],
  providers: [StarsService],
  controllers: [StarsController, PerksController],
  exports: [StarsService],
})
export class StarsModule {}
