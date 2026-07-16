import { Controller, Get, Patch, Post, Delete, Param, UseGuards, Request, Body, UseInterceptors, UploadedFile } from '@nestjs/common';
import { FileInterceptor } from '@nestjs/platform-express';
import { diskStorage } from 'multer';
import { extname } from 'path';
import { UsersService } from './users.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Roles } from '../auth/decorators/roles.decorator';
import { User } from './entities/user.entity';

@Controller('users')
@UseGuards(JwtAuthGuard, RolesGuard)
export class UsersController {
  constructor(private readonly usersService: UsersService) {}

  @UseGuards(JwtAuthGuard)
  @Get('profile')
  async getProfile(@Request() req: any) {
    const user = await this.usersService.findById(req.user.userId);
    return user;
  }

  @UseGuards(JwtAuthGuard)
  @Patch('profile')
  async updateProfile(@Request() req: any, @Body() updateData: Partial<User>) {
    return this.usersService.update(req.user.userId, updateData);
  }

  @UseGuards(JwtAuthGuard)
  @Post('upload')
  @UseInterceptors(FileInterceptor('file', {
    storage: diskStorage({
      destination: './uploads',
      filename: (req, file, cb) => {
        const randomName = Array(32).fill(null).map(() => (Math.round(Math.random() * 16)).toString(16)).join('');
        return cb(null, `${randomName}${extname(file.originalname)}`);
      }
    })
  }))
  async uploadFile(@Request() req: any, @UploadedFile() file: Express.Multer.File) {
    const photoUrl = `http://localhost:${process.env.PORT || 3000}/uploads/${file.filename}`;
    await this.usersService.update(req.user.userId, { photoUrl });
    return { photoUrl };
  }

  @Roles('ADMIN')
  @Get()
  async findAll() {
    return this.usersService.findAll();
  }

  @Roles('ADMIN')
  @Get(':id')
  async findOne(@Param('id') id: string) {
    return this.usersService.findById(id);
  }

  @Roles('ADMIN')
  @Post()
  async create(@Body() createData: Partial<User> & { roleNames?: string[] }) {
    return this.usersService.create(createData);
  }

  @Roles('ADMIN')
  @Patch(':id')
  async update(@Param('id') id: string, @Body() updateData: Partial<User> & { roleNames?: string[] }) {
    return this.usersService.update(id, updateData);
  }

  @Roles('ADMIN')
  @Delete(':id')
  async remove(@Param('id') id: string) {
    await this.usersService.delete(id);
    return { success: true };
  }
}
