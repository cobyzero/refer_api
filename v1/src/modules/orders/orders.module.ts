import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { OrdersService } from './orders.service';
import { OrdersController } from './orders.controller';
import { Order } from './entities/order.entity';
import { OrderItem } from './entities/order-item.entity';
import { ProductsModule } from '../products/products.module';


import { OrdersGateway } from './orders.gateway';
import { JwtModule } from '@nestjs/jwt';

@Module({
  imports: [
    TypeOrmModule.forFeature([Order, OrderItem]),
    JwtModule.register({
      secret: process.env.JWT_SECRET || 'super-secret-key-12345',
      signOptions: { expiresIn: '1d' },
    }),
    ProductsModule,
  ],
  controllers: [OrdersController],
  providers: [OrdersService, OrdersGateway],
  exports: [OrdersService],
})
export class OrdersModule {}
