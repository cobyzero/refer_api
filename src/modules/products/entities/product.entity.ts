import { Entity, Column, PrimaryGeneratedColumn, CreateDateColumn, OneToMany, ManyToOne, JoinColumn } from 'typeorm';
import { ProductAvailableSize } from './product-available-size.entity';
import { ProductCategory } from './product-category.entity';

@Entity('products')
export class Product {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  name: string;

  @Column()
  description: string;

  @Column({ nullable: true })
  categoryId: string;

  @ManyToOne(() => ProductCategory, (cat) => cat.products)
  @JoinColumn({ name: 'categoryId' })
  category: ProductCategory;

  @Column('float')
  price: number;

  @Column({ nullable: true })
  imageUrl: string;

  @Column({ default: true })
  isAvailable: boolean;

  @Column({ default: 0 })
  starsReward: number; // Stars earned when buying this

  @Column('float', { default: 0 })
  rating: number;

  @Column({ default: 0 })
  numReviews: number;

  @CreateDateColumn()
  createdAt: Date;

  @OneToMany(() => ProductAvailableSize, (sizeOption) => sizeOption.product)
  sizeOptions: ProductAvailableSize[];
}

