import { Entity, PrimaryGeneratedColumn, Column, ManyToOne } from 'typeorm';
import { Order } from './order.entity';

@Entity('order_items')
export class OrderItem {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @ManyToOne(() => Order, (order) => order.items, { onDelete: 'CASCADE' })
  order: Order;

  @Column()
  orderId: string;

  // Stored as plain string to avoid FK constraint failures
  // if the product is later deleted or modified.
  @Column({ nullable: true })
  productId: string;

  @Column({ nullable: true })
  imageUrl: string;

  @Column()
  name: string;

  @Column()
  quantity: number;

  @Column({ type: 'float' })
  price: number;

  @Column({ nullable: true })
  size: string;

  @Column({ nullable: true })
  type: string;

  @Column({ type: 'simple-array', nullable: true })
  enhancements?: string[];
}
