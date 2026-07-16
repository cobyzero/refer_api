import { Entity, Column, PrimaryGeneratedColumn, ManyToOne, JoinColumn } from 'typeorm';
import { Product } from './product.entity';
import { ProductSize } from './product-size.entity';

@Entity('product_available_sizes')
export class ProductAvailableSize {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  productId: string;

  @Column()
  sizeId: string;

  @Column('float', { nullable: true })
  price: number;

  @ManyToOne(() => Product, (product) => product.sizeOptions)
  @JoinColumn({ name: 'productId' })
  product: any;

  @ManyToOne(() => ProductSize)
  @JoinColumn({ name: 'sizeId' })
  size: ProductSize;
}
