import { ProductsService } from './products.service';

describe('ProductsService', () => {
  const productRepository = { find: jest.fn(), findOne: jest.fn() };
  const seasonalProductRepository = { find: jest.fn() };
  const categoryRepository = { find: jest.fn() };
  let service: ProductsService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new ProductsService(
      productRepository as any,
      seasonalProductRepository as any,
      categoryRepository as any,
    );
  });

  it('maps products on findAll', async () => {
    productRepository.find.mockResolvedValue([
      {
        id: 'p1',
        name: 'Coffee',
        category: { name: 'Cafe', iconUrl: 'icon' },
        sizeOptions: [{ size: { id: 's1', name: 'Grande' }, price: 2 }],
        options: [{ id: 'o1', name: 'Oat', price: 1 }],
        enhancements: [{ id: 'e1', name: 'Caramel', price: 1 }],
      },
    ]);

    const result = await service.findAll();
    expect(result[0].category).toBe('Cafe');
    expect(result[0].availableSizes).toHaveLength(1);
    expect(result[0].types).toHaveLength(1);
  });

  it('returns categories', async () => {
    categoryRepository.find.mockResolvedValue([{ id: 'c1', name: 'Cafe' }]);
    await expect(service.getCategories()).resolves.toEqual([{ id: 'c1', name: 'Cafe' }]);
  });
});

