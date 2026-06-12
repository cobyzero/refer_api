import { LocationsService } from './locations.service';

describe('LocationsService', () => {
  const repo = { find: jest.fn(), findOne: jest.fn() };
  let service: LocationsService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new LocationsService(repo as any);
  });

  it('returns active locations', async () => {
    repo.find.mockResolvedValue([{ id: 'l1' }]);
    await expect(service.findAll()).resolves.toEqual([{ id: 'l1' }]);
    expect(repo.find).toHaveBeenCalledWith({ where: { isActive: true } });
  });

  it('throws when location is not found', async () => {
    repo.findOne.mockResolvedValue(null);
    await expect(service.findOne('x')).rejects.toThrow('Location not found');
  });
});

