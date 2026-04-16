import { Entity, Column, PrimaryGeneratedColumn, CreateDateColumn } from 'typeorm';

@Entity('rewards')
export class Reward {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  title: string;

  @Column()
  description: string;

  @Column()
  starsRequired: number;

  @Column({ nullable: true })
  imageUrl: string;

  @Column({ nullable: true })
  icon: string;

  @Column({ default: true })
  isActive: boolean;
}
