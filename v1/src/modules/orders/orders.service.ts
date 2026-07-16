import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, DataSource } from 'typeorm';
import { randomUUID } from 'crypto';
import { Order, OrderStatus } from './entities/order.entity';
import { CreateOrderDto } from './dto/create-order.dto';
import { OrdersGateway } from './orders.gateway';
import { ProductsService } from '../products/products.service';

@Injectable()
export class OrdersService {
  constructor(
    @InjectRepository(Order)
    private readonly orderRepo: Repository<Order>,
    private readonly ordersGateway: OrdersGateway,
    private readonly productsService: ProductsService,
    private readonly dataSource: DataSource,
  ) { }

  async create(userId: string, createOrderDto: CreateOrderDto): Promise<Order> {
    const orderId = randomUUID();
    const orderNumber = `ER-${Math.floor(1000 + Math.random() * 9000)}`;

    // Enrich items with starsReward from DB so the SP can compute them
    const enrichedItems = await Promise.all(
      createOrderDto.items.map(async (item) => {
        const product = await this.productsService.findOne(item.productId);
        return {
          id: randomUUID(),
          productId: item.productId,
          name: item.name,
          quantity: item.quantity,
          price: item.price,
          imageUrl: item.imageUrl ?? null,
          size: item.size ?? null,
          type: item.type ?? null,
          enhancements: item.enhancements?.join(',') ?? null,
          starsReward: product?.starsReward ?? 0,
        };
      }),
    );

    const pickupTime = createOrderDto.pickupTime
      ? new Date(createOrderDto.pickupTime)
      : null;

    // ── Call sp_create_order ─────────────────────────────────────
    // OUT params are retrieved via MySQL session variables (@var)
    await this.dataSource.query(
      `CALL sp_create_order(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, @totalStars)`,
      [
        orderId,
        userId,
        createOrderDto.locationId,
        orderNumber,
        createOrderDto.total,
        createOrderDto.serviceFee,
        createOrderDto.paymentMethod,
        pickupTime,
        JSON.stringify(enrichedItems),
        createOrderDto.redeemedRewardId ?? null,
      ],
    );

    // Read the OUT param from the session variable
    const [{ totalStars }] = await this.dataSource.query(
      'SELECT @totalStars AS totalStars',
    );

    console.log(
      `[sp_create_order] Order ${orderNumber} created — ` +
      `${totalStars ?? 0} stars awarded to user ${userId}`,
    );

    // Fetch full order with relations for the real-time notification
    const fullOrder = await this.orderRepo.findOne({
      where: { id: orderId },
      relations: ['user', 'items'],
    });

    this.ordersGateway.sendNewOrderNotification(fullOrder);

    if (!fullOrder) {
      throw new Error('Order not found');
    }

    return fullOrder;
  }

  async findAll(): Promise<Order[]> {
    return this.orderRepo.find({
      relations: ['items', 'user'],
      order: { createdAt: 'DESC' },
    });
  }

  async findAllByUser(userId: string): Promise<Order[]> {
    return this.orderRepo.find({
      where: { userId },
      relations: ['items', 'user'],
      order: { createdAt: 'DESC' },
    });
  }

  async updateStatus(orderId: string, status: OrderStatus): Promise<Order> {
    const order = await this.orderRepo.findOne({
      where: { id: orderId },
      relations: ['items', 'user'],
    });

    if (!order) throw new Error('Order not found');

    order.status = status;
    const updatedOrder = await this.orderRepo.save(order);

    this.ordersGateway.sendOrderStatusUpdate(order.userId, updatedOrder);

    return updatedOrder;
  }
}
