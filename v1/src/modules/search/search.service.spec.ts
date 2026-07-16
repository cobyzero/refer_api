import { SearchService } from './search.service';

describe('SearchService', () => {
  const productRepository = { find: jest.fn() };
  const categoryRepository = { find: jest.fn() };
  let service: SearchService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new SearchService(productRepository as any, categoryRepository as any);
  });

  it('searches and maps products', async () => {
    productRepository.find.mockResolvedValue([
      {
        id: 'p1',
        name: 'Latte',
        category: { name: 'Cafe', iconUrl: 'icon' },
        sizeOptions: [],
        options: [],
        enhancements: [],
      },
    ]);

    const result = await service.searchProducts('lat', 'All');
    expect(productRepository.find).toHaveBeenCalled();
    expect(result[0].category).toBe('Cafe');
  });

  it('returns categories', async () => {
    categoryRepository.find.mockResolvedValue([{ id: 'c1', name: 'Cafe' }]);
    await expect(service.getCategories()).resolves.toEqual([{ id: 'c1', name: 'Cafe' }]);
  });
});

