import { DataSource } from 'typeorm';
import { User } from './src/modules/users/entities/user.entity';
import { Role } from './src/modules/users/entities/role.entity';

const dataSource = new DataSource({
  type: 'sqlite',
  database: 'database.sqlite',
  entities: [User, Role],
  synchronize: false,
});

async function main() {
  await dataSource.initialize();
  const repo = dataSource.getRepository(User);
  const user = await repo.findOne({
    where: { id: '00000000-0000-0000-0000-000000000001' },
    relations: ['roles'],
  });
  console.log('Query result:', user);
  await dataSource.destroy();
}

main().catch(console.error);
