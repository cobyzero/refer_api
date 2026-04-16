import { Injectable, UnauthorizedException, ConflictException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { UsersService } from '../users/users.service';
import { RegisterDto, LoginDto } from './dto/auth.dto';
import * as bcrypt from 'bcrypt';

@Injectable()
export class AuthService {
  constructor(
    private readonly usersService: UsersService,
    private readonly jwtService: JwtService,
  ) {}

  async register(registerDto: RegisterDto) {
    const existingUser = await this.usersService.findByEmail(registerDto.email);
    if (existingUser) {
      throw new ConflictException('Email already exists');
    }

    const hashedPassword = await bcrypt.hash(registerDto.password, 10);
    const user = await this.usersService.create({
      ...registerDto,
      password: hashedPassword,
    });

    const token = this.generateToken(user.id, user.email, []); // New users have no roles by default
    return {
      user: {
        id: user.id,
        name: user.name,
        email: user.email,
        keepUpdated: user.keepUpdated,
        roles: [],
      },
      ...token,
    };
  }

  async login(loginDto: LoginDto) {
    const user = await this.usersService.findByEmail(loginDto.email);
    if (!user) {
      throw new UnauthorizedException('Invalid credentials');
    }

    const isPasswordValid = await bcrypt.compare(loginDto.password, user.password);
    if (!isPasswordValid) {
      throw new UnauthorizedException('Invalid credentials');
    }

    const roles = user.roles?.map(r => r.name) || [];
    const token = this.generateToken(user.id, user.email, roles);
    return {
      user: {
        id: user.id,
        name: user.name,
        email: user.email,
        keepUpdated: user.keepUpdated,
        photoUrl: user.photoUrl,
        stars: user.stars,
        roles: roles,
      },
      ...token,
    };
  }

  private generateToken(userId: string, email: string, roles: string[]) {
    const payload = { sub: userId, email, roles };
    return {
      accessToken: this.jwtService.sign(payload),
      // In a real app, you'd also generate a refresh token here
      refreshToken: 'mock-refresh-token-' + Math.random().toString(36).substring(7),
    };
  }

  async validateToken(token: string) {
    try {
      return this.jwtService.verify(token);
    } catch (e) {
      throw new UnauthorizedException('Invalid token');
    }
  }
}
