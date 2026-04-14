import {
  WebSocketGateway,
  WebSocketServer,
  OnGatewayConnection,
  OnGatewayDisconnect,
  SubscribeMessage,
  MessageBody,
  ConnectedSocket,
} from '@nestjs/websockets';
import { Server, Socket } from 'socket.io';
import { JwtService } from '@nestjs/jwt';
import { Logger, Inject, forwardRef } from '@nestjs/common';
import { CartService } from './cart.service';

@WebSocketGateway({
  cors: {
    origin: '*',
  },
  namespace: 'cart',
})
export class CartGateway implements OnGatewayConnection, OnGatewayDisconnect {
  @WebSocketServer()
  server: Server;

  private logger = new Logger('CartGateway');

  constructor(
    private jwtService: JwtService,
    @Inject(forwardRef(() => CartService))
    private cartService: CartService,
  ) { }

  async handleConnection(client: Socket) {
    try {
      const token = client.handshake.auth.token || client.handshake.headers.authorization?.split(' ')[1];
      if (!token) {
        client.disconnect();
        return;
      }

      const payload = this.jwtService.verify(token);
      const userId = payload.sub;

      client.data.userId = userId;
      client.join(`user_${userId}`);
      this.logger.log(`Client connected: ${client.id} (User: ${userId})`);

      // Send initial cart state
      const cart = await this.cartService.getCart(userId);
      client.emit('cartUpdated', cart);
    } catch (e) {
      this.logger.error(`Connection failed: ${e.message}`);
      client.disconnect();
    }
  }

  handleDisconnect(client: Socket) {
    this.logger.log(`Client disconnected: ${client.id}`);
  }

  @SubscribeMessage('addToCart')
  async handleAddToCart(@ConnectedSocket() client: Socket, @MessageBody() data: any) {
    const userId = client.data.userId;
    if (!userId) return;
    return this.cartService.addToCart(userId, data);
  }

  @SubscribeMessage('removeItem')
  async handleRemoveItem(@ConnectedSocket() client: Socket, @MessageBody() itemId: string) {
    const userId = client.data.userId;
    if (!userId) return;
    return this.cartService.removeFromCart(userId, itemId);
  }

  @SubscribeMessage('updateQuantity')
  async handleUpdateQuantity(@ConnectedSocket() client: Socket, @MessageBody() data: { itemId: string, quantity: number }) {
    const userId = client.data.userId;
    if (!userId) return;
    return this.cartService.updateItemQuantity(userId, data.itemId, data.quantity);
  }

  @SubscribeMessage('clearCart')
  async handleClearCart(@ConnectedSocket() client: Socket) {
    const userId = client.data.userId;
    if (!userId) return;
    return this.cartService.clearCart(userId);
  }

  sendCartUpdate(userId: string, cartData: any) {
    this.server.to(`user_${userId}`).emit('cartUpdated', cartData);
  }
}
