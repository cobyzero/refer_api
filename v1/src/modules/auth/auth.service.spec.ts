import { ConflictException, UnauthorizedException } from '@nestjs/common';
import * as bcrypt from 'bcrypt';
import { AuthService } from './auth.service';

jest.mock('bcrypt', () => ({
  hash: jest.fn(),
  compare: jest.fn(),
}));

describe('AuthService', () => {
  const usersService = {
    findByEmail: jest.fn(),
    create: jest.fn(),
    findByIdWithPassword: jest.fn(),
    update: jest.fn(),
  };
  const jwtService = {
    sign: jest.fn(),
    verify: jest.fn(),
  };

  let service: AuthService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new AuthService(usersService as any, jwtService as any);
  });

  it('registers a new user', async () => {
    usersService.findByEmail.mockResolvedValue(null);
    usersService.create.mockResolvedValue({
      id: 'u1',
      name: 'Test',
      email: 'test@mail.com',
      keepUpdated: true,
    });
    jwtService.sign.mockReturnValue('token');
    (bcrypt.hash as jest.Mock).mockResolvedValue('hashed');

    const result = await service.register({
      name: 'Test',
      email: 'test@mail.com',
      password: 'secret',
      keepUpdated: true,
    });

    expect(result.accessToken).toBe('token');
    expect(result.user.email).toBe('test@mail.com');
  });

  it('throws conflict when email already exists', async () => {
    usersService.findByEmail.mockResolvedValue({ id: 'u1' });
    await expect(service.register({} as any)).rejects.toBeInstanceOf(ConflictException);
  });

  it('throws unauthorized on invalid password login', async () => {
    usersService.findByEmail.mockResolvedValue({ password: 'hashed' });
    (bcrypt.compare as jest.Mock).mockResolvedValue(false);

    await expect(service.login({ email: 'a', password: 'b' } as any)).rejects.toBeInstanceOf(
      UnauthorizedException,
    );
  });
});
