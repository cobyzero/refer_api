import { IsString, IsNumber, IsOptional, IsArray, ValidateNested } from 'class-validator';
import { Type } from 'class-transformer';

export class CreateOrderItemDto {
  @IsString()
  productId: string;

  @IsOptional()
  @IsString()
  imageUrl?: string;

  @IsString()
  name: string;

  @IsNumber()
  quantity: number;

  @IsNumber()
  price: number;

  @IsOptional()
  @IsString()
  size?: string;

  @IsOptional()
  @IsString()
  type?: string;
}

export class CreateOrderDto {
  @IsString()
  locationId: string;

  @IsNumber()
  total: number;

  @IsNumber()
  serviceFee: number;

  @IsString()
  paymentMethod: string;

  @IsOptional()
  @IsString()
  pickupTime?: string;

  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => CreateOrderItemDto)
  items: CreateOrderItemDto[];

  @IsOptional()
  @IsString()
  redeemedRewardId?: string;
}
