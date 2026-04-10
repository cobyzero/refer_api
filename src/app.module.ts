import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ConfigModule } from '@nestjs/config';
import { UsersModule } from './modules/users/users.module';
import { AuthModule } from './modules/auth/auth.module';
import { AppConfigModule } from './modules/app-config/app-config.module';
import { DashboardModule } from './modules/dashboard/dashboard.module';
import { StarsModule } from './modules/stars/stars.module';
import { SearchModule } from './modules/search/search.module';
import { SeedModule } from './modules/seed/seed.module';
import { ProductsModule } from './modules/products/products.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
    }),
    TypeOrmModule.forRoot({
      type: 'sqlite',
      database: process.env.DATABASE_NAME || 'database.sqlite',
      entities: [__dirname + '/**/*.entity{.ts,.js}'],
      synchronize: true, // Auto-create tables (use only in dev)
    }),
    UsersModule,
    AuthModule,
    AppConfigModule,
    DashboardModule,
    StarsModule,
    SearchModule,
    SeedModule,
    ProductsModule,
  ],
})
export class AppModule { }
