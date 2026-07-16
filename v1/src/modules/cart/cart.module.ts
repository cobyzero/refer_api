import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { CartItem } from './entities/cart-item.entity';
import { CartService } from './cart.service';
import { CartController } from './cart.controller';
import { CartGateway } from './cart.gateway';
import { ProductEnhancement } from '../products/entities/product-enhancement.entity';
import { AuthModule } from '../auth/auth.module';

@Module({
  imports: [
    TypeOrmModule.forFeature([CartItem, ProductEnhancement]),
    AuthModule, // Import AuthModule to use JwtModule's JwtService
  ],
  providers: [CartService, CartGateway],
  controllers: [CartController],
})
export class CartModule {}
