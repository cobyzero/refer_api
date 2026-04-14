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
      sizeOptions: undefined, // Hide raw relations
      options: undefined,
      categoryId: undefined,
    };
  }

  async getSeasonalProducts() {
    const seasonalProducts = await this.seasonalProductRepository.find({
      relations: ['product', 'product.category', 'product.sizeOptions', 'product.sizeOptions.size', 'product.options', 'product.enhancements'],
    });

    return seasonalProducts.map(sp => this.mapProduct(sp.product));
  }

  async getTopRated() {
    const products = await this.productRepository.find({
      where: { isAvailable: true },
      order: { rating: 'DESC' },
      take: 5,
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements'],
    });
    return products.map(p => this.mapProduct(p));
  }

  async findAll() {
    const products = await this.productRepository.find({
      where: { isAvailable: true },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements'],
    });
    return products.map(p => this.mapProduct(p));
  }

  async findOne(id: string) {
    const product = await this.productRepository.findOne({
      where: { id, isAvailable: true },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements'],
    });

    return this.mapProduct(product);
  }

  async getCategories() {
    return this.categoryRepository.find();
  }
}
