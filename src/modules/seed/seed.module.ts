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

@Module({
  imports: [
    TypeOrmModule.forFeature([
      User, 
      Product, 
      SeasonalProduct, 
      Reward, 
      StarTransaction,
      ProductSize,
      ProductAvailableSize,
      ProductCategory
    ]),
  ],
  providers: [SeedService],
})
export class SeedModule {}
