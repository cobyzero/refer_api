import { Injectable, NotFoundException, Inject, forwardRef } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { In, Repository } from 'typeorm';
import { CartItem } from './entities/cart-item.entity';
import { ProductEnhancement } from '../products/entities/product-enhancement.entity';
import { CartGateway } from './cart.gateway';

@Injectable()
export class CartService {
  constructor(
    @InjectRepository(CartItem)
    private readonly cartItemRepository: Repository<CartItem>,
    @InjectRepository(ProductEnhancement)
    private readonly enhancementRepository: Repository<ProductEnhancement>,
    @Inject(forwardRef(() => CartGateway))
    private readonly cartGateway: CartGateway, // Inject the gateway
  ) { }

  private async notifyCartUpdate(userId: string) {
    const cart = await this.getCart(userId);
    this.cartGateway.sendCartUpdate(userId, cart);
  }

  async getCart(userId: string) {
    const items = await this.cartItemRepository.find({
      where: { userId },
      relations: [
        'product',
        'product.category',
        'product.sizeOptions',
        'size',
        'type',
        'enhancements'
      ],
      order: { id: 'ASC' }
    });

    const mappedItems = items.map(item => {
      let unitPrice = item.product.price;
      if (item.size) {
        const sizeOption = item.product.sizeOptions.find(opt => opt.sizeId === item.sizeId);
        if (sizeOption) unitPrice += sizeOption.price;
      }
      if (item.type) unitPrice += item.type.price;

      const enhancementsPrice = item.enhancements.reduce((sum, e) => sum + (e.price || 0), 0);
      unitPrice += enhancementsPrice;

      return {
        id: item.id,
        productId: item.productId,
        name: item.product.name,
        imageUrl: item.product.imageUrl,
        category: item.product.category?.name,
        size: item.size?.name,
        type: item.type?.name,
        enhancements: item.enhancements.map(e => e.name),
        quantity: item.quantity,
        unitPrice: unitPrice,
        totalPrice: unitPrice * item.quantity,
        starsReward: (item.product.starsReward || 0) * item.quantity,
      };
    });

    const grandTotal = mappedItems.reduce((sum, item) => sum + item.totalPrice, 0);

    return {
      items: mappedItems,
      total: grandTotal,
    };
  }

  async addToCart(userId: string, data: {
    productId: string;
    sizeId?: string;
    typeId?: string;
    enhancementIds?: string[];
    quantity?: number;
  }) {
    const { productId, sizeId, typeId, enhancementIds = [], quantity = 1 } = data;

    const cartItem = this.cartItemRepository.create({
      userId,
      productId,
      sizeId,
      typeId,
      quantity,
    });

    if (enhancementIds.length > 0) {
      cartItem.enhancements = await this.enhancementRepository.find({
        where: { id: In(enhancementIds) }
      });
    }

    const saved = await this.cartItemRepository.save(cartItem);

    // Notify in real-time
    await this.notifyCartUpdate(userId);

    return saved;
  }

  async removeFromCart(userId: string, itemId: string) {
    const result = await this.cartItemRepository.delete({ id: itemId, userId });
    if (result.affected === 0) {
      throw new NotFoundException('Item not found in cart');
    }

    // Notify in real-time
    await this.notifyCartUpdate(userId);

    return { success: true };
  }

  async updateItemQuantity(userId: string, itemId: string, quantity: number) {
    if (quantity <= 0) {
      return this.removeFromCart(userId, itemId);
    }

    const item = await this.cartItemRepository.findOne({ where: { id: itemId, userId } });
    if (!item) {
      throw new NotFoundException('Item not found in cart');
    }

    item.quantity = quantity;
    await this.cartItemRepository.save(item);

    // Notify in real-time
    await this.notifyCartUpdate(userId);

    return { success: true };
  }

  async clearCart(userId: string) {
    await this.cartItemRepository.delete({ userId });

    // Notify in real-time
    await this.notifyCartUpdate(userId);

    return { success: true };
  }
}
