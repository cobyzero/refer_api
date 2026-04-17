import { Entity, Column, PrimaryGeneratedColumn, CreateDateColumn, ManyToOne } from 'typeorm';
import { User } from '../../users/entities/user.entity';
import { Reward } from './reward.entity';

@Entity('redeemed_rewards')
export class RedeemedReward {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  userId: string;

  @ManyToOne(() => User)
  user: User;

  @Column()
  rewardId: string;

  @ManyToOne(() => Reward)
  reward: Reward;

  @Column()
  rewardTitle: string;

  @Column({ default: false })
  isClaimed: boolean;

  @Column({ nullable: true })
  orderId: string;

  @CreateDateColumn()
  createdAt: Date;
}
