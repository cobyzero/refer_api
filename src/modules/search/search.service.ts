import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, Like } from 'typeorm';
import { Product } from '../products/entities/product.entity';

@Injectable()
export class SearchService {
  constructor(
    @InjectRepository(Product)
    private readonly productRepository: Repository<Product>,
  ) {}

  private mapProduct(product: Product | null) {
    if (!product) return null;
    return {
      ...product,
      category: product.category?.name || 'All',
      categoryIcon: product.category?.iconUrl,
      availableSizes: (product.sizeOptions || []).map(opt => ({
        id: opt.size?.id,
        name: opt.size?.name,
        price: opt.price,
      })),
      sizeOptions: undefined,
      categoryId: undefined,
    };
  }

  async searchProducts(query?: string, categoryName?: string) {
    const where: any = { isAvailable: true };

    if (query) {
      where.name = Like(`%${query}%`);
    }

    if (categoryName && categoryName !== 'All') {
      // We need to filter by category name
      // find options don't support deep relations easily in 'where' without QueryBuilder or nested objects but let's try
      where.category = { name: categoryName };
    }

    const products = await this.productRepository.find({
      where,
      order: { name: 'ASC' },
      relations: ['category', 'sizeOptions', 'sizeOptions.size'],
    });

    return products.map(p => this.mapProduct(p));
  }

  async getCategories() {
    // This now returns category objects or just names? The user wanted a table.
    // Let's return the full category info for the dashboard if needed.
    // But for search filter, names are fine.
    const products = await this.productRepository.find({ 
      relations: ['category'],
      select: ['category'] 
    });
    const categories = Array.from(new Set(products.map(p => p.category?.name).filter(Boolean)));
    return ['All', ...categories];
  }
}
