import { UsersService } from './users.service';

describe('UsersService', () => {
  const makeRepo = () => ({
    create: jest.fn(),
    save: jest.fn(),
    findOne: jest.fn(),
    update: jest.fn(),
    find: jest.fn(),
  });

  it('creates and saves a new user', async () => {
    const repo = makeRepo();
    const service = new UsersService(repo as any);
    const payload = { email: 'a@b.com' };
    const created = { id: 'u1', ...payload };

    repo.create.mockReturnValue(created);
    repo.save.mockResolvedValue(created);

    await expect(service.create(payload as any)).resolves.toEqual(created);
    expect(repo.create).toHaveBeenCalledWith(payload);
    expect(repo.save).toHaveBeenCalledWith(created);
  });

  it('throws when update target does not exist', async () => {
    const repo = makeRepo();
    const service = new UsersService(repo as any);

    repo.update.mockResolvedValue(undefined);
    repo.findOne.mockResolvedValue(null);

    await expect(service.update('u1', { name: 'New' } as any)).rejects.toThrow('User not found');
  });
});

