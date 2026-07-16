import { NotFoundException } from '@nestjs/common';
import { CartService } from './cart.service';

describe('CartService', () => {
  const cartItemRepository = {
    find: jest.fn(),
    create: jest.fn(),
    save: jest.fn(),
    delete: jest.fn(),
    findOne: jest.fn(),
  };
  const enhancementRepository = { find: jest.fn() };
  const cartGateway = { sendCartUpdate: jest.fn() };
  let service: CartService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new CartService(
      cartItemRepository as any,
      enhancementRepository as any,
      cartGateway as any,
    );
  });

  it('maps cart and calculates totals', async () => {
    cartItemRepository.find.mockResolvedValue([
      {
        id: 'i1',
        userId: 'u1',
        productId: 'p1',
        quantity: 2,
        sizeId: 's1',
        typeId: 't1',
        size: { name: 'Grande' },
        type: { name: 'Oat', price: 1 },
        product: {
          name: 'Latte',
          imageUrl: 'img',
          price: 5,
          starsReward: 2,
          category: { name: 'Cafe' },
          sizeOptions: [{ sizeId: 's1', price: 2 }],
        },
        enhancements: [{ name: 'Caramel', price: 1 }],
      },
    ]);

    const result = await service.getCart('u1');
    expect(result.items[0].unitPrice).toBe(9);
    expect(result.total).toBe(18);
  });

  it('throws when removing missing item', async () => {
    cartItemRepository.delete.mockResolvedValue({ affected: 0 });
    await expect(service.removeFromCart('u1', 'i1')).rejects.toBeInstanceOf(NotFoundException);
  });
});

