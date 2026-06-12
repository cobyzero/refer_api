import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { User } from './entities/user.entity';
import { Role } from './entities/role.entity';
import * as bcrypt from 'bcrypt';

@Injectable()
export class UsersService {
  constructor(
    @InjectRepository(User)
    private readonly userRepository: Repository<User>,
    @InjectRepository(Role)
    private readonly roleRepository: Repository<Role>,
  ) { }

  async create(userData: Partial<User> & { roleNames?: string[] }): Promise<User> {
    const { roleNames, password, ...rest } = userData;
    const user = this.userRepository.create(rest);
    
    if (password) {
      user.password = await bcrypt.hash(password, 10);
    } else {
      user.password = await bcrypt.hash('password123', 10);
    }

    if (roleNames && roleNames.length > 0) {
      const roles = await this.roleRepository.find({
        where: roleNames.map(name => ({ name })),
      });
      user.roles = roles;
    } else {
      const userRole = await this.roleRepository.findOne({ where: { name: 'USER' } });
      if (userRole) {
        user.roles = [userRole];
      }
    }

    return this.userRepository.save(user);
  }

  async findByEmail(email: string): Promise<User | null> {
    return this.userRepository.findOne({
      where: { email },
      select: ['id', 'name', 'email', 'password', 'keepUpdated', 'photoUrl'],
      relations: ['roles']
    });
  }

  async findById(id: string): Promise<User | null> {
    return this.userRepository.findOne({ where: { id }, relations: ['roles'] });
  }

  async findByIdWithPassword(id: string): Promise<User | null> {
    return this.userRepository.findOne({
      where: { id },
      select: ['id', 'password'],
    });
  }

  async update(id: string, updateData: Partial<User> & { roleNames?: string[] }): Promise<User> {
    const user = await this.userRepository.findOne({ where: { id }, relations: ['roles'] });
    if (!user) throw new Error('User not found');

    const { roleNames, password, ...rest } = updateData;

    Object.assign(user, rest);

    if (password) {
      user.password = await bcrypt.hash(password, 10);
    }

    if (roleNames) {
      const roles = await this.roleRepository.find({
        where: roleNames.map(name => ({ name })),
      });
      user.roles = roles;
    }

    return this.userRepository.save(user);
  }

  async delete(id: string): Promise<void> {
    await this.userRepository.delete(id);
  }

  async findAll(): Promise<User[]> {
    return this.userRepository.find({
      relations: ['roles'],
    });
  }
}
