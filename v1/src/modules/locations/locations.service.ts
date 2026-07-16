import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Location } from './entities/location.entity';

@Injectable()
export class LocationsService {
  constructor(
    @InjectRepository(Location)
    private readonly locationRepository: Repository<Location>,
  ) { }

  async findAll(): Promise<Location[]> {
    return this.locationRepository.find({ where: { isActive: true } });
  }

  async findOne(id: string): Promise<Location> {
    const location = await this.locationRepository.findOne({ where: { id } });
    if (!location) {
      throw new Error('Location not found');
    }
    return location;
  }
}
