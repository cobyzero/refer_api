import { SeedService } from './seed.service';

describe('SeedService', () => {
  const repo = () => ({
    clear: jest.fn(),
    save: jest.fn(),
    find: jest.fn(),
    query: jest.fn(),
  });

  const buildService = () =>
    new SeedService(
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
      repo() as any,
    );

  it('runs seed pipeline on module init', async () => {
    const service = buildService();
    const cleanSpy = jest.spyOn(service as any, 'cleanDatabase').mockResolvedValue(undefined);
    const usersSpy = jest.spyOn(service as any, 'seedUsers').mockResolvedValue(undefined);
    const categoriesSpy = jest
      .spyOn(service as any, 'seedCategories')
      .mockResolvedValue([{ id: 'c1', name: 'Cafe' }]);
    const productsSpy = jest.spyOn(service as any, 'seedProducts').mockResolvedValue(undefined);
    const sizesSpy = jest.spyOn(service as any, 'seedSizes').mockResolvedValue(undefined);
    const optionsSpy = jest.spyOn(service as any, 'seedOptions').mockResolvedValue(undefined);
    const enhancementsSpy = jest.spyOn(service as any, 'seedEnhancements').mockResolvedValue(undefined);
    const rewardsSpy = jest.spyOn(service as any, 'seedRewards').mockResolvedValue(undefined);
    const locationsSpy = jest.spyOn(service as any, 'seedLocations').mockResolvedValue(undefined);

    await service.onModuleInit();

    expect(cleanSpy).toHaveBeenCalled();
    expect(usersSpy).toHaveBeenCalled();
    expect(categoriesSpy).toHaveBeenCalled();
    expect(productsSpy).toHaveBeenCalledWith([{ id: 'c1', name: 'Cafe' }]);
    expect(sizesSpy).toHaveBeenCalled();
    expect(optionsSpy).toHaveBeenCalled();
    expect(enhancementsSpy).toHaveBeenCalled();
    expect(rewardsSpy).toHaveBeenCalled();
    expect(locationsSpy).toHaveBeenCalled();
  });
});

