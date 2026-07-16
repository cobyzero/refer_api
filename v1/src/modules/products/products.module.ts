import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { Product } from './entities/product.entity';
import { SeasonalProduct } from './entities/seasonal-product.entity';
import { ProductSize } from './entities/product-size.entity';
import { ProductAvailableSize } from './entities/product-available-size.entity';
import { ProductCategory } from './entities/product-category.entity';
import { ProductType } from './entities/product-type.entity';
import { ProductEnhancement } from './entities/product-enhancement.entity';
import { ProductsService } from './products.service';
import { ProductsController } from './products.controller';

@Module({
  imports: [TypeOrmModule.forFeature([Product, SeasonalProduct, ProductSize, ProductAvailableSize, ProductCategory, ProductType, ProductEnhancement])],
  providers: [ProductsService],
  controllers: [ProductsController],
  exports: [ProductsService],
})
export class ProductsModule { }
