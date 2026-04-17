import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, DataSource } from 'typeorm';
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
    private readonly dataSource: DataSource,
  ) { }

  async create(userId: string, createOrderDto: CreateOrderDto): Promise<Order> {
    const queryRunner = this.dataSource.createQueryRunner();
    await queryRunner.connect();
    await queryRunner.startTransaction();

    try {
      const orderNumber = `ER-${Math.floor(1000 + Math.random() * 9000)}`;

      const order = this.orderRepo.create({
        ...createOrderDto,
        userId,
        orderNumber,
        status: OrderStatus.ORDERED,
      });

      const savedOrder = await queryRunner.manager.save(order);

      // Calculate stars
      let totalStars = 0;
      for (const item of createOrderDto.items) {
        const product = await this.productsService.findOne(item.productId);
        if (product) {
          totalStars += (product.starsReward || 0) * item.quantity;
        }
      }

      if (totalStars > 0) {
        console.log(`Rewarding ${totalStars} stars to user ${userId} for order ${orderNumber}`);
        await this.starsService.rewardStars(
          userId,
          totalStars,
          `Order ${orderNumber}`,
        );
      }

      if (createOrderDto.redeemedRewardId) {
        console.log(`Claiming reward ${createOrderDto.redeemedRewardId} for order ${orderNumber}`);
        await this.starsService.claimReward(
          createOrderDto.redeemedRewardId,
          savedOrder.id,
        );
      }

      await queryRunner.commitTransaction();
      return savedOrder;
    } catch (err) {
      await queryRunner.rollbackTransaction();
      throw err;
    } finally {
      await queryRunner.release();
    }
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
