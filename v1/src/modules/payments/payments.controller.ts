import { Controller, Post, Body, UseGuards, Request } from '@nestjs/common';
import { PaymentsService } from './payments.service';
import { JwtAuthGuard } from '../auth/guards/jwt-auth.guard';

@Controller('payments')
export class PaymentsController {
  constructor(private readonly paymentsService: PaymentsService) {}

  @UseGuards(JwtAuthGuard)
  @Post('create-payment-intent')
  async createPaymentIntent(@Body() data: { amount: number; currency?: string }) {
    return this.paymentsService.createPaymentIntent(data.amount, data.currency);
  }
}
