import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Order, OrderStatus } from './entities/order.entity';
import { CreateOrderDto } from './dto/create-order.dto';
import { OrdersGateway } from './orders.gateway';
import { ProductsService } from '../products/products.service';
import { StarsService } from '../stars/stars.service';

@Injectable()
export class OrdersService {
  constructor(
    @InjectRepository(Order)
    private readonly orderRepo: Repository<Order>,
    private readonly ordersGateway: OrdersGateway,
    private readonly productsService: ProductsService,
    private readonly starsService: StarsService,
  ) {}

  async create(userId: string, createOrderDto: CreateOrderDto): Promise<Order> {
    const orderNumber = `ER-${Math.floor(1000 + Math.random() * 9000)}`;
    
    const order = this.orderRepo.create({
      ...createOrderDto,
      userId,
      orderNumber,
      status: OrderStatus.ORDERED,
    });

    const savedOrder = await this.orderRepo.save(order);

    // Calculate stars
    let totalStars = 0;
    for (const item of createOrderDto.items) {
      const product = await this.productsService.findOne(item.productId);
      if (product) {
        totalStars += (product.starsReward || 0) * item.quantity;
      }
    }

    if (totalStars > 0) {
      await this.starsService.rewardStars(
        userId,
        totalStars,
        `Order ${orderNumber}`,
      );
    }

    return savedOrder;
  }

  async updateStatus(orderId: string, status: OrderStatus): Promise<Order> {
    const order = await this.orderRepo.findOne({
      where: { id: orderId },
      relations: ['items'],
    });

    if (!order) throw new Error('Order not found');

    order.status = status;
    const updatedOrder = await this.orderRepo.save(order);

    // Emit socket update
    this.ordersGateway.sendOrderStatusUpdate(order.userId, updatedOrder);

    return updatedOrder;
  }

  async findAllByUser(userId: string): Promise<Order[]> {
    return this.orderRepo.find({
      where: { userId },
      relations: ['items'],
      order: { createdAt: 'DESC' },
    });
  }
}
