import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Product } from './entities/product.entity';
import { SeasonalProduct } from './entities/seasonal-product.entity';
import { ProductCategory } from './entities/product-category.entity';
import { ProductEnhancement } from './entities/product-enhancement.entity';

@Injectable()
export class ProductsService {
  constructor(
    @InjectRepository(Product)
    private readonly productRepository: Repository<Product>,
    @InjectRepository(SeasonalProduct)
    private readonly seasonalProductRepository: Repository<SeasonalProduct>,
    @InjectRepository(ProductCategory)
    private readonly categoryRepository: Repository<ProductCategory>,
    @InjectRepository(ProductEnhancement)
    private readonly enhancementRepository: Repository<ProductEnhancement>,
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
      order: { createdAt: 'DESC' },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements'],
    });
    return products.map(p => this.mapProduct(p));
  }

  async findOne(id: string) {
    const product = await this.productRepository.findOne({
      where: { id },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements'],
    });

    return this.mapProduct(product);
  }

  async getCategories() {
    return this.categoryRepository.find();
  }

  async update(id: string, body: any) {
    const product = await this.productRepository.findOne({
      where: { id },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements']
    });
    if (!product) {
      throw new Error('Product not found');
    }
    
    if (body.name !== undefined) product.name = body.name;
    if (body.description !== undefined) product.description = body.description;
    if (body.price !== undefined) product.price = Number(body.price);
    if (body.isAvailable !== undefined) product.isAvailable = body.isAvailable;
    if (body.starsReward !== undefined) product.starsReward = Number(body.starsReward);

    await this.productRepository.save(product);

    const updated = await this.productRepository.findOne({
      where: { id },
      relations: ['category', 'sizeOptions', 'sizeOptions.size', 'options', 'enhancements']
    });
    return this.mapProduct(updated);
  }

  async create(body: any) {
    const product = new Product();
    product.name = body.name || 'New Product';
    product.description = body.description || '';
    product.price = Number(body.price) || 0;
    product.starsReward = Number(body.starsReward) || 0;
    product.isAvailable = body.isAvailable !== undefined ? body.isAvailable : true;
    product.rating = 5.0;
    product.numReviews = 1;

    if (body.categoryId) {
      product.categoryId = body.categoryId;
    } else {
      const categories = await this.getCategories();
      if (categories.length > 0) {
        product.categoryId = categories[0].id;
      }
    }

    if (body.imageUrl) {
      product.imageUrl = body.imageUrl;
    }

    const saved = await this.productRepository.save(product);
    return this.findOne(saved.id);
  }

  async addEnhancement(productId: string, body: any) {
    const enhancement = new ProductEnhancement();
    enhancement.name = body.name || 'Extra';
    enhancement.price = Number(body.price) || 0;
    enhancement.productId = productId;
    
    await this.enhancementRepository.save(enhancement);
    return this.findOne(productId);
  }

  async deleteEnhancement(enhancementId: string) {
    const enh = await this.enhancementRepository.findOne({ where: { id: enhancementId } });
    if (enh) {
      const productId = enh.productId;
      await this.enhancementRepository.delete(enhancementId);
      return this.findOne(productId);
    }
    return null;
  }
}
