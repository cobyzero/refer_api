import { Entity, Column, PrimaryGeneratedColumn, CreateDateColumn, ManyToOne } from 'typeorm';
import { User } from '../../users/entities/user.entity';

@Entity('star_transactions')
export class StarTransaction {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  userId: string;

  @ManyToOne(() => User)
  user: User;

  @Column()
  type: 'earn' | 'redeem';

  @Column()
  title: string;

  @Column()
  stars: number; // Positive for earn, negative for redeem

  @CreateDateColumn()
  date: Date;
}
