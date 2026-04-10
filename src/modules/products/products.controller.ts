import { Controller, Get, Param } from '@nestjs/common';
import { ProductsService } from './products.service';

@Controller('products')
export class ProductsController {
  constructor(private readonly productsService: ProductsService) { }

  @Get()
  async getAll() {
    return this.productsService.findAll();
  }

  @Get('top-rated')
  async getTopRated() {
    return this.productsService.getTopRated();
  }

  @Get('seasonal')
  async getSeasonal() {
    return this.productsService.getSeasonalProducts();
  }

  @Get('categories')
  async getCategories() {
    return this.productsService.getCategories();
  }

  @Get(':id')
  async getOne(@Param('id') id: string) {
    return this.productsService.findOne(id);
  }
}
