import { Controller, Get, Param, Patch, Body, Post, UseInterceptors, UploadedFile, Delete } from '@nestjs/common';
import { FileInterceptor } from '@nestjs/platform-express';
import { diskStorage } from 'multer';
import { extname } from 'path';
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

  @Patch(':id')
  async update(@Param('id') id: string, @Body() body: any) {
    return this.productsService.update(id, body);
  }

  @Post()
  async create(@Body() body: any) {
    return this.productsService.create(body);
  }

  @Post('upload')
  @UseInterceptors(FileInterceptor('file', {
    storage: diskStorage({
      destination: './uploads',
      filename: (req, file, cb) => {
        const randomName = Array(32).fill(null).map(() => (Math.round(Math.random() * 16)).toString(16)).join('');
        return cb(null, `${randomName}${extname(file.originalname)}`);
      }
    })
  }))
  async uploadFile(@UploadedFile() file: any) {
    const port = process.env.PORT || 3001;
    const imageUrl = `http://localhost:${port}/uploads/${file.filename}`;
    return { imageUrl };
  }

  @Post(':id/enhancements')
  async addEnhancement(@Param('id') productId: string, @Body() body: any) {
    return this.productsService.addEnhancement(productId, body);
  }

  @Delete(':id/enhancements/:enhancementId')
  async deleteEnhancement(@Param('enhancementId') enhancementId: string) {
    return this.productsService.deleteEnhancement(enhancementId);
  }
}
