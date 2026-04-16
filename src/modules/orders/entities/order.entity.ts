import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, ManyToOne, OneToMany } from 'typeorm';
import { User } from '../../users/entities/user.entity';
import { Location } from '../../locations/entities/location.entity';
import { OrderItem } from './order-item.entity';

export enum OrderStatus {
  ORDERED = 'ORDERED',
  PREPARING = 'PREPARING',
  READY = 'READY',
  COMPLETED = 'COMPLETED',
  CANCELLED = 'CANCELLED',
}

@Entity('orders')
export class Order {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  orderNumber: string;

  @ManyToOne(() => User)
  user: User;

  @Column()
  userId: string;

  @ManyToOne(() => Location)
  location: Location;

  @Column()
  locationId: string;

  @Column({ type: 'float' })
  total: number;

  @Column({ type: 'float', default: 0 })
  serviceFee: number;

  @Column({
    type: 'simple-enum',
    enum: OrderStatus,
    default: OrderStatus.ORDERED,
  })
  status: OrderStatus;

  @Column()
  paymentMethod: string;

  @Column({ nullable: true })
  pickupTime: Date;

  @OneToMany(() => OrderItem, (item) => item.order, { cascade: true })
  items: OrderItem[];

  @CreateDateColumn()
  createdAt: Date;
}
