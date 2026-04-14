import { Injectable, OnModuleInit, Logger } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { User } from '../users/entities/user.entity';
import { Product } from '../products/entities/product.entity';
import { SeasonalProduct } from '../products/entities/seasonal-product.entity';
import { Reward } from '../stars/entities/reward.entity';
import { StarTransaction } from '../stars/entities/star-transaction.entity';
import { ProductSize } from '../products/entities/product-size.entity';
import { ProductAvailableSize } from '../products/entities/product-available-size.entity';
import { ProductCategory } from '../products/entities/product-category.entity';
import { ProductType } from '../products/entities/product-type.entity';
import { ProductEnhancement } from '../products/entities/product-enhancement.entity';
import * as bcrypt from 'bcrypt';

@Injectable()
export class SeedService implements OnModuleInit {
  private readonly logger = new Logger(SeedService.name);

  constructor(
    @InjectRepository(User) private readonly userRepo: Repository<User>,
    @InjectRepository(Product) private readonly productRepo: Repository<Product>,
    @InjectRepository(Reward) private readonly rewardRepo: Repository<Reward>,
    @InjectRepository(StarTransaction) private readonly transactionRepo: Repository<StarTransaction>,
    @InjectRepository(SeasonalProduct) private readonly seasonalRepo: Repository<SeasonalProduct>,
    @InjectRepository(ProductSize) private readonly sizeRepo: Repository<ProductSize>,
    @InjectRepository(ProductAvailableSize) private readonly productSizeRepo: Repository<ProductAvailableSize>,
    @InjectRepository(ProductCategory) private readonly categoryRepo: Repository<ProductCategory>,
    @InjectRepository(ProductType) private readonly typeRepo: Repository<ProductType>,
    @InjectRepository(ProductEnhancement) private readonly enhancementRepo: Repository<ProductEnhancement>,
  ) { }

  async onModuleInit() {
    this.logger.log('Starting DB Seeding...');
    await this.cleanDatabase();
    await this.seedUsers();
    const categories = await this.seedCategories();
    await this.seedProducts(categories);
    await this.seedSizes();
    await this.seedOptions();
    await this.seedEnhancements();
    await this.seedRewards();
    this.logger.log('Seeding completed successfully');
  }

  private async cleanDatabase() {
    this.logger.log('Cleaning database...');
    await this.enhancementRepo.clear();
    await this.typeRepo.clear();
    await this.productSizeRepo.clear();
    await this.sizeRepo.clear();
    await this.seasonalRepo.clear();
    await this.transactionRepo.clear();
    await this.rewardRepo.clear();
    await this.productRepo.clear();
    await this.categoryRepo.clear();
    await this.userRepo.clear();
  }

  private async seedUsers() {
    this.logger.log('Seeding users...');
    const hashedPassword = await bcrypt.hash('admin', 10);
    await this.userRepo.save({
      id: '00000000-0000-0000-0000-000000000001',
      name: 'Elias Rodriguez',
      email: 'admin@admin.com',
      password: hashedPassword,
      stars: 85,
      photoUrl: 'https://i.pravatar.cc/150?u=elias',
      keepUpdated: true,
    });
  }

  private async seedCategories() {
    this.logger.log('Seeding categories...');
    const categories = [
      { name: 'Café', iconUrl: 'https://cdn-icons-png.flaticon.com/512/924/924514.png' },
      { name: 'Té', iconUrl: 'https://cdn-icons-png.flaticon.com/512/3504/3504827.png' },
      { name: 'Panadería', iconUrl: 'https://cdn-icons-png.flaticon.com/512/3014/3014534.png' },
      { name: 'Mercancía', iconUrl: 'https://cdn-icons-png.flaticon.com/512/3081/3081559.png' },
    ];
    return this.categoryRepo.save(categories);
  }

  private async seedProducts(categories: ProductCategory[]) {
    this.logger.log('Seeding products...');
    const getCat = (name: string) => categories.find(c => c.name.toLowerCase() === name.toLowerCase())?.id;
    const products = [
      { name: 'Cascara Cold Brew', description: 'Cherry-steeped morning bliss', categoryId: getCat('Café'), price: 6.50, rating: 4.9, numReviews: 124, starsReward: 15, imageUrl: 'https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=400&h=400' },
      { name: 'Spiced Autumn Latte', description: 'Warming seasonal notes', categoryId: getCat('Café'), price: 5.75, rating: 4.8, numReviews: 89, starsReward: 12, imageUrl: 'https://images.unsplash.com/photo-1570968915860-54d5c301fa9f?auto=format&fit=crop&w=400&h=400' },
      { name: 'Honey Almond Frappuccino', description: 'Sweet and nutty refreshment', categoryId: getCat('Café'), price: 6.25, rating: 4.7, numReviews: 45, starsReward: 14, imageUrl: 'https://images.unsplash.com/photo-1572490122747-3968b75cc699?auto=format&fit=crop&w=400&h=400' },
      { name: 'Matcha Mint Fizz', description: 'Sparkling green tea energy', categoryId: getCat('Té'), price: 5.50, rating: 4.6, numReviews: 32, starsReward: 10, imageUrl: 'https://images.unsplash.com/photo-1515823064-d6e0c04616a7?auto=format&fit=crop&w=400&h=400' },
      { name: 'Vanilla Sweet Cream', description: 'Smooth and creamy classic', categoryId: getCat('Café'), price: 5.25, rating: 4.7, numReviews: 210, starsReward: 10, imageUrl: 'https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=400&h=400' },
      { name: 'Caramel Macchiato', description: 'Espresso with silky caramel', categoryId: getCat('Café'), price: 5.50, rating: 4.8, numReviews: 180, starsReward: 12, imageUrl: 'https://images.unsplash.com/photo-1485808191679-5f86510681a2?auto=format&fit=crop&w=400&h=400' },
      { name: 'Butter Croissant', description: 'Flaky, buttery French specialty', categoryId: getCat('Panadería'), price: 3.50, rating: 4.9, numReviews: 340, starsReward: 8, imageUrl: 'https://images.unsplash.com/photo-1555507036-ab1f4038808a?auto=format&fit=crop&w=400&h=400' },
      { name: 'Blueberry Muffin', description: 'Bursting with fresh berries', categoryId: getCat('Panadería'), price: 3.25, rating: 4.6, numReviews: 156, starsReward: 8, imageUrl: 'https://images.unsplash.com/photo-1607958996333-41aef7caefaa?auto=format&fit=crop&w=400&h=400' },
      { name: 'Chocolate Chip Cookie', description: 'Warm with gooey chocolate', categoryId: getCat('Panadería'), price: 2.75, rating: 4.9, numReviews: 245, starsReward: 5, imageUrl: 'https://images.unsplash.com/photo-1499636136210-6f4ee915583e?auto=format&fit=crop&w=400&h=400' }
    ];
    const savedProducts = await this.productRepo.save(products);
    const seasonalNames = ['Spiced Autumn Latte', 'Honey Almond Frappuccino', 'Matcha Mint Fizz'];
    for (const prod of savedProducts) {
      if (seasonalNames.includes(prod.name)) await this.seasonalRepo.save({ productId: prod.id });
    }
  }

  private async seedRewards() {
    this.logger.log('Seeding rewards...');
    const rewards = [
      { title: 'Butter Croissant', description: 'Redeem for any fresh bakery item', starsRequired: 50, imageUrl: 'https://images.unsplash.com/photo-1555507036-ab1f4038808a?auto=format&fit=crop&w=400&h=400' },
      { title: 'Any Handcrafted Drink', description: 'Includes seasonal and limited editions', starsRequired: 150, imageUrl: 'https://images.unsplash.com/photo-1541167760496-162955ed8a9f?auto=format&fit=crop&w=400&h=400' },
      { title: 'Bag of Whole Bean Coffee', description: '250g of our signature roast', starsRequired: 400, imageUrl: 'https://images.unsplash.com/photo-1559056199-641a0ac8b55e?auto=format&fit=crop&w=400&h=400' }
    ];
    await this.rewardRepo.save(rewards);
  }

  private async seedSizes() {
    this.logger.log('Seeding product sizes...');
    const sizes = [{ name: 'Normal' }, { name: 'Grande' }, { name: 'Extragrande' }, { name: 'Expreso' }];
    const savedSizes = await this.sizeRepo.save(sizes);
    const products = await this.productRepo.find({ relations: ['category'] });
    for (const prod of products) {
      const catName = prod.category?.name || '';
      if (['Café', 'Té'].includes(catName)) {
        if (prod.name.toLowerCase().includes('espresso')) {
          await this.productSizeRepo.save({ productId: prod.id, sizeId: savedSizes.find(s => s.name === 'Expreso')?.id, price: Math.random() * 10 });
        } else {
          for (const size of savedSizes) {
            if (size.name !== 'Expreso') {
              await this.productSizeRepo.save({ productId: prod.id, sizeId: size.id, price: Math.random() * 10 });
            }
          }
        }
      }
    }
  }

  private async seedOptions() {
    this.logger.log('Seeding product types (choices)...');
    const products = await this.productRepo.find({ relations: ['category'] });
    for (const prod of products) {
      const catName = prod.category?.name || '';
      if (['Café', 'Té'].includes(catName)) {
        const milks = ['Original', 'Oat Milk', 'Almond Milk', 'Soy Milk'];
        for (const milk of milks) {
          await this.typeRepo.save({ name: milk, productId: prod.id, price: (milk === 'Original' ? 0 : 0.5) });
        }
      } else if (catName === 'Panadería') {
        await this.typeRepo.save({ name: 'Warmed', productId: prod.id, price: 0 });
        await this.typeRepo.save({ name: 'Not Warmed', productId: prod.id, price: 0 });
      }
    }
  }

  private async seedEnhancements() {
    this.logger.log('Seeding product enhancements (add-ons)...');
    const products = await this.productRepo.find({ relations: ['category'] });
    for (const prod of products) {
      const catName = prod.category?.name || '';
      if (['Café', 'Té'].includes(catName)) {
        await this.enhancementRepo.save({ name: 'Shot de espresso extra', price: 1.25, productId: prod.id });
        await this.enhancementRepo.save({ name: 'Endulzando', price: 3.00, productId: prod.id });
        await this.enhancementRepo.save({ name: 'Caramel Drizzle', price: 0.50, productId: prod.id });
      }
    }
  }
}
