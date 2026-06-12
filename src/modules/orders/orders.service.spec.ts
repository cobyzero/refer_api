import { DataSource } from 'typeorm';
import { OrdersService } from './orders.service';
import { OrderStatus } from './entities/order.entity';

describe('OrdersService', () => {
  const orderRepo = {
    create: jest.fn(),
    findOne: jest.fn(),
    find: jest.fn(),
    save: jest.fn(),
  };
  const ordersGateway = {
    sendNewOrderNotification: jest.fn(),
    sendOrderStatusUpdate: jest.fn(),
  };
  const productsService = { findOne: jest.fn() };
  const starsService = { rewardStars: jest.fn(), claimReward: jest.fn() };
  const queryRunner = {
    connect: jest.fn(),
    startTransaction: jest.fn(),
    manager: { save: jest.fn() },
    commitTransaction: jest.fn(),
    rollbackTransaction: jest.fn(),
    release: jest.fn(),
    isTransactionActive: true,
  };
  const dataSource = {
    createQueryRunner: jest.fn(() => queryRunner),
  } as unknown as DataSource;

  let service: OrdersService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new OrdersService(
      orderRepo as any,
      ordersGateway as any,
      productsService as any,
      starsService as any,
      dataSource,
    );
  });

  it('updates order status and notifies gateway', async () => {
    const order = { id: 'o1', userId: 'u1', status: OrderStatus.ORDERED };
    orderRepo.findOne.mockResolvedValue(order);
    orderRepo.save.mockResolvedValue({ ...order, status: OrderStatus.READY_FOR_PICKUP });

    const result = await service.updateStatus('o1', OrderStatus.READY_FOR_PICKUP);
    expect(result.status).toBe(OrderStatus.READY_FOR_PICKUP);
    expect(ordersGateway.sendOrderStatusUpdate).toHaveBeenCalled();
  });

  it('throws when order is not found on update status', async () => {
    orderRepo.findOne.mockResolvedValue(null);
    await expect(service.updateStatus('o1', OrderStatus.CANCELLED)).rejects.toThrow('Order not found');
  });
});

