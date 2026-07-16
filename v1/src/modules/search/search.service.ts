import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository, Like } from 'typeorm';
import { Product } from '../products/entities/product.entity';
import { ProductCategory } from '../products/entities/product-category.entity';

@Injectable()
export class SearchService {
  constructor(
    @InjectRepository(Product)
    private readonly productRepository: Repository<Product>,
    @InjectRepository(ProductCategory)
    private readonly categoryRepository: Repository<ProductCategory>,
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
      types: (product.options || []).map(opt => ({
        id: opt.id,
        name: opt.name,
        price: opt.price,
      })),
      enhancements: (product.enhancements || []).map(e => ({
        id: e.id,
        name: e.name,
        price: e.price,
      })),
      sizeOptions: undefined,
      options: undefined,
      categoryId: undefined,
    };
  }

  async searchProducts(query?: string, categoryIdOrName?: string) {
    const where: any = { isAvailable: true };

    if (query) {
      where.name = Like(`%${query}%`);
    }

    if (categoryIdOrName && categoryIdOrName !== 'All') {
      // Check if it looks like a UUID (categoryId) or just a name
      const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(categoryIdOrName);
      
      if (isUuid) {
        where.categoryId = categoryIdOrName;
      } else {
        where.category = { name: categoryIdOrName };
      }
    }

    const products = await this.productRepository.find({
      where,
      order: { name: 'ASC' },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements'],
    });

    return products.map(p => this.mapProduct(p));
  }

  async getCategories() {
    return this.categoryRepository.find();
  }
}
