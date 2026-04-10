import { Entity, Column, PrimaryGeneratedColumn } from 'typeorm';

@Entity('product_sizes')
export class ProductSize {
  @PrimaryGeneratedColumn('uuid')
  id: string;

  @Column()
  name: string; // e.g., 'Pequeño', 'Normal', 'Grande', 'Expreso'
}
