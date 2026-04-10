import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { Product } from './entities/product.entity';
import { SeasonalProduct } from './entities/seasonal-product.entity';
import { ProductSize } from './entities/product-size.entity';
import { ProductAvailableSize } from './entities/product-available-size.entity';
import { ProductCategory } from './entities/product-category.entity';
import { ProductsService } from './products.service';
import { ProductsController } from './products.controller';

@Module({
  imports: [TypeOrmModule.forFeature([Product, SeasonalProduct, ProductSize, ProductAvailableSize, ProductCategory])],
  providers: [ProductsService],
  controllers: [ProductsController],
  exports: [ProductsService],
})
export class ProductsModule { }
