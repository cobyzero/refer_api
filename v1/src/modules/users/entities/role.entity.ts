import { Entity, Column, PrimaryGeneratedColumn, ManyToMany } from 'typeorm';
import { User } from './user.entity';

@Entity('roles')
export class Role {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column({ unique: true })
  name: string; // e.g., 'ADMIN', 'USER'

  @ManyToMany(() => User, (user) => user.roles)
  users: User[];
}
