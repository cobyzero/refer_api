import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Product } from './entities/product.entity';
import { SeasonalProduct } from './entities/seasonal-product.entity';
import { ProductCategory } from './entities/product-category.entity';

@Injectable()
export class ProductsService {
  constructor(
    @InjectRepository(Product)
    private readonly productRepository: Repository<Product>,
    @InjectRepository(SeasonalProduct)
    private readonly seasonalProductRepository: Repository<SeasonalProduct>,
    @InjectRepository(ProductCategory)
    private readonly categoryRepository: Repository<ProductCategory>,
  ) { }

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
      sizeOptions: undefined, // Hide raw relations
      categoryId: undefined,
    };
  }

  async getSeasonalProducts() {
    const seasonalProducts = await this.seasonalProductRepository.find({
      relations: ['product', 'product.category', 'product.sizeOptions', 'product.sizeOptions.size'],
    });

    return seasonalProducts.map(sp => this.mapProduct(sp.product));
  }

  async getTopRated() {
    const products = await this.productRepository.find({
      where: { isAvailable: true },
      order: { rating: 'DESC' },
      take: 5,
      relations: ['category', 'sizeOptions', 'sizeOptions.size'],
    });
    return products.map(p => this.mapProduct(p));
  }

  async findAll() {
    const products = await this.productRepository.find({
      where: { isAvailable: true },
      relations: ['category', 'sizeOptions', 'sizeOptions.size'],
    });
    return products.map(p => this.mapProduct(p));
  }

  async findOne(id: string) {
    const product = await this.productRepository.findOne({
      where: { id, isAvailable: true },
      relations: ['category', 'sizeOptions', 'sizeOptions.size'],
    });

    return this.mapProduct(product);
  }

  async getCategories() {
    return this.categoryRepository.find();
  }
}
