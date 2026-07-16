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
    const userRepo = makeRepo();
    const roleRepo = makeRepo();
    const service = new UsersService(userRepo as any, roleRepo as any);
    const payload = { email: 'a@b.com' };
    const created = { id: 'u1', ...payload };

    userRepo.create.mockReturnValue(created);
    userRepo.save.mockResolvedValue(created);
    roleRepo.findOne.mockResolvedValue({ id: 'r1', name: 'USER' });

    await expect(service.create(payload as any)).resolves.toEqual(created);
    expect(userRepo.create).toHaveBeenCalledWith(payload);
    expect(userRepo.save).toHaveBeenCalledWith(created);
  });

  it('throws when update target does not exist', async () => {
    const userRepo = makeRepo();
    const roleRepo = makeRepo();
    const service = new UsersService(userRepo as any, roleRepo as any);

    userRepo.findOne.mockResolvedValue(null);

    await expect(service.update('u1', { name: 'New' } as any)).rejects.toThrow('User not found');
  });
});

