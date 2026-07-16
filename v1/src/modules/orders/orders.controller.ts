import { Controller, Post, Get, Body, UseGuards, Request, Param, Patch } from '@nestjs/common';
import { OrdersService } from './orders.service';
import { CreateOrderDto } from './dto/create-order.dto';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';
import { OrderStatus } from './entities/order.entity';
import { RolesGuard } from '../auth/guards/roles.guard';
import { Roles } from '../auth/decorators/roles.decorator';

@Controller('orders')
@UseGuards(JwtAuthGuard, RolesGuard)
export class OrdersController {
  constructor(private readonly ordersService: OrdersService) { }

  @UseGuards(JwtAuthGuard)
  @Post()
  async create(@Request() req: any, @Body() createOrderDto: CreateOrderDto) {
    return this.ordersService.create(req.user.userId, createOrderDto);
  }

  @Roles('ADMIN')
  @Patch(':id/status')
  async updateStatus(@Param('id') id: string, @Body('status') status: OrderStatus) {
    return this.ordersService.updateStatus(id, status);
  }

  @Get()
  async findAll(@Request() req: any) {
    const isAdmin = req.user.roles?.includes('ADMIN');
    if (isAdmin) {
      return this.ordersService.findAll();
    }
    return this.ordersService.findAllByUser(req.user.userId);
  }
}
