import { Controller, Get, Query } from '@nestjs/common';
import { SearchService } from './search.service';

@Controller('search')
export class SearchController {
  constructor(private readonly searchService: SearchService) {}

  @Get('referrals')
  async search(
    @Query('q') q: string,
    @Query('category') category: string,
  ) {
    return this.searchService.searchProducts(q, category);
  }

  @Get('categories')
  async getCategories() {
    return this.searchService.getCategories();
  }
}
