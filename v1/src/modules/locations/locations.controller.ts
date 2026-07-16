import { Controller, Get, Param } from '@nestjs/common';
import { LocationsService } from './locations.service';
import { Location } from './entities/location.entity';

@Controller('locations')
export class LocationsController {
  constructor(private readonly locationsService: LocationsService) {}

  @Get()
  async findAll(): Promise<Location[]> {
    return this.locationsService.findAll();
  }

  @Get(':id')
  async findOne(@Param('id') id: string): Promise<Location> {
    return this.locationsService.findOne(id);
  }
}
