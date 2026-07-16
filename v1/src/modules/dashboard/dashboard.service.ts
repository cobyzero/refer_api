import { Injectable } from '@nestjs/common';
import { UsersService } from '../users/users.service';
import { StarsService } from '../stars/stars.service';
import { OrdersService } from '../orders/orders.service';

@Injectable()
export class DashboardService {
  constructor(
    private readonly usersService: UsersService,
    private readonly starsService: StarsService,
    private readonly ordersService: OrdersService,
  ) {}

  async getSummary(userId: string) {
    const user = await this.usersService.findById(userId);
    const stars = await this.starsService.getBalance(userId);
    const rewards = await this.starsService.getRewards();
    const history = await this.starsService.getHistory(userId);
    
    return {
      user: {
        name: user?.name,
        photoUrl: user?.photoUrl,
        stars: stars,
      },
      stars: stars,
      rewards: rewards,
      recentActivity: history.slice(0, 5).map(h => ({
        id: h.id,
        type: h.type,
        title: h.title,
        stars: h.stars,
        date: typeof h.date === 'string' ? h.date : (h.date ? h.date.toISOString() : new Date().toISOString()),
      })),
      nextRewardProgress: stars / 100 > 1 ? 1 : stars / 100,
    };
  }

  async getAdminSummary() {
    const users = await this.usersService.findAll();
    const orders = await this.ordersService.findAll();
    const totalStarsAwarded = await this.starsService.getTotalStarsAwarded();
    const totalStarsRedeemed = await this.starsService.getTotalStarsRedeemed();
    
    const totalOrders = orders.length;
    const totalUsers = users.length;
    
    // Recent activity: mix of new users and status updates
    const recentOrders = orders.slice(0, 5).map(o => ({
      user: o.user?.name || 'Usuario',
      action: `Realizó el pedido ${o.orderNumber}`,
      time: this.formatTime(o.createdAt),
      avatar: (o.user?.name || 'U').substring(0, 2).toUpperCase(),
    }));

    return {
      stats: [
        { label: 'Usuarios Totales', value: totalUsers.toLocaleString(), change: '+5%', icon: 'group', color: 'primary' },
        { label: 'Órdenes Totales', value: totalOrders.toLocaleString(), change: '+12%', icon: 'shopping_cart', color: 'secondary' },
        { label: 'Estrellas Repartidas', value: totalStarsAwarded > 1000 ? `${(totalStarsAwarded / 1000).toFixed(1)}K` : totalStarsAwarded.toString(), change: '+8%', icon: 'stars', color: 'accent' },
        { label: 'Puntos Canjeados', value: totalStarsRedeemed.toLocaleString(), change: '+15%', icon: 'redeem', color: 'success' },
      ],
      recentActivity: recentOrders,
    };
  }

  private formatTime(date: Date | string): string {
    const d = new Date(date);
    const now = new Date();
    const diffMs = now.getTime() - d.getTime();
    const diffMin = Math.floor(diffMs / 60000);
    
    if (diffMin < 60) return `hace ${diffMin} min`;
    const diffHrs = Math.floor(diffMin / 60);
    if (diffHrs < 24) return `hace ${diffHrs} hora${diffHrs > 1 ? 's' : ''}`;
    return d.toLocaleDateString();
  }
}
