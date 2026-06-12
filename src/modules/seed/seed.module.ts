import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { SeedService } from './seed.service';
import { User } from '../users/entities/user.entity';
import { Product } from '../products/entities/product.entity';
import { SeasonalProduct } from '../products/entities/seasonal-product.entity';
import { Reward } from '../stars/entities/reward.entity';
import { StarTransaction } from '../stars/entities/star-transaction.entity';
import { ProductSize } from '../products/entities/product-size.entity';
import { ProductAvailableSize } from '../products/entities/product-available-size.entity';
import { ProductCategory } from '../products/entities/product-category.entity';
import { ProductType } from '../products/entities/product-type.entity';
import { ProductEnhancement } from '../products/entities/product-enhancement.entity';
import { Location } from '../locations/entities/location.entity';
import { Order } from '../orders/entities/order.entity';
import { OrderItem } from '../orders/entities/order-item.entity';

import { Role } from '../users/entities/role.entity';

@Module({
  imports: [
    TypeOrmModule.forFeature([
      User, 
      Role,
      Product,
      SeasonalProduct, 
      Reward, 
      StarTransaction,
      ProductSize,
      ProductAvailableSize,
      ProductCategory,
      ProductType,
      ProductEnhancement,
      Location,
      Order,
      OrderItem
    ]),
  ],
  providers: [SeedService],
})
export class SeedModule {}
