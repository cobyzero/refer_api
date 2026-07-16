import { Entity, Column, PrimaryGeneratedColumn, ManyToOne, ManyToMany, JoinTable, JoinColumn } from 'typeorm';
import { User } from '../../users/entities/user.entity';
import { Product } from '../../products/entities/product.entity';
import { ProductSize } from '../../products/entities/product-size.entity';
import { ProductType } from '../../products/entities/product-type.entity';
import { ProductEnhancement } from '../../products/entities/product-enhancement.entity';

@Entity('cart_items')
export class CartItem {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  userId: string;

  @ManyToOne(() => User)
  @JoinColumn({ name: 'userId' })
  user: User;

  @Column()
  productId: string;

  @ManyToOne(() => Product)
  @JoinColumn({ name: 'productId' })
  product: Product;

  @Column({ nullable: true })
  sizeId: string;

  @ManyToOne(() => ProductSize)
  @JoinColumn({ name: 'sizeId' })
  size: ProductSize;

  @Column({ nullable: true })
  typeId: string;

  @ManyToOne(() => ProductType)
  @JoinColumn({ name: 'typeId' })
  type: ProductType;

  @Column({ default: 1 })
  quantity: number;

  @ManyToMany(() => ProductEnhancement)
  @JoinTable({
    name: 'cart_item_enhancements',
    joinColumn: { name: 'cartItemId', referencedColumnName: 'id' },
    inverseJoinColumn: { name: 'enhancementId', referencedColumnName: 'id' }
  })
  enhancements: ProductEnhancement[];
}
