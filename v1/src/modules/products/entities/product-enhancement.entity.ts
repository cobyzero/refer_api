import { Entity, Column, PrimaryGeneratedColumn, ManyToOne, JoinColumn } from 'typeorm';
import { Product } from './product.entity';

@Entity('product_enhancements')
export class ProductEnhancement {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  name: string; // e.g., 'Extra Espresso Shot', 'Sweetener'

  @Column('float', { default: 0 })
  price: number; 

  @Column()
  productId: string;

  @ManyToOne(() => Product, (product) => product.enhancements)
  @JoinColumn({ name: 'productId' })
  product: any;
}
