import { Entity, Column, PrimaryGeneratedColumn, ManyToOne, JoinColumn } from 'typeorm';
import { Product } from './product.entity';

@Entity('product_types')
export class ProductType {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  name: string; // e.g., 'Oat Milk', 'Vanilla Syrup', 'Extra shot'

  @Column('float', { default: 0 })
  price: number; 

  @Column()
  productId: string;

  @ManyToOne(() => Product, (product) => product.options)
  @JoinColumn({ name: 'productId' })
  product: Product;
}
