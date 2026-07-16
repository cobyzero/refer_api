import {
  WebSocketGateway,
  WebSocketServer,
  OnGatewayConnection,
  OnGatewayDisconnect,
} from '@nestjs/websockets';
import { Server, Socket } from 'socket.io';
import { JwtService } from '@nestjs/jwt';
import { Logger } from '@nestjs/common';

@WebSocketGateway({
  cors: {
    origin: '*',
  },
  namespace: 'orders',
})
export class OrdersGateway implements OnGatewayConnection, OnGatewayDisconnect {
  @WebSocketServer()
  server: Server;

  private logger = new Logger('OrdersGateway');

  constructor(private jwtService: JwtService) {}

  async handleConnection(client: Socket) {
    try {
      const token = client.handshake.auth.token || client.handshake.headers.authorization?.split(' ')[1];
      if (!token) {
        client.disconnect();
        return;
      }

      const payload = this.jwtService.verify(token);
      const userId = payload.sub || payload.userId;
      const roles = payload.roles || [];

      client.data.userId = userId;
      client.join(`user_${userId}`);

      if (roles.includes('ADMIN')) {
        client.join('admins');
        this.logger.log(`Admin connected: ${client.id}`);
      }

      this.logger.log(`Client connected to orders: ${client.id} (User: ${userId})`);
    } catch (e) {
      if (e.message === 'jwt expired') {
        this.logger.warn(`Orders socket connection rejected: token expired (Client: ${client.id})`);
      } else {
        this.logger.error(`Orders socket connection failed: ${e.message}`);
      }
      client.disconnect();
    }
  }

  handleDisconnect(client: Socket) {
    this.logger.log(`Client disconnected from orders: ${client.id}`);
  }

  sendOrderStatusUpdate(userId: string, orderData: any) {
    this.server.to(`user_${userId}`).emit('orderStatusUpdated', orderData);
    this.server.to('admins').emit('orderStatusUpdated', orderData);
  }

  sendNewOrderNotification(orderData: any) {
    this.logger.log(`Broadcasting new order ${orderData?.orderNumber || orderData?.id} to admins`);
    this.server.to('admins').emit('newOrder', orderData);
  }
}
