import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { createE2eApp, loginAndGetToken } from './e2e-utils';

describe('Users and Perks (e2e)', () => {
  let app: INestApplication;
  let userToken: string;
  let adminToken: string;

  beforeAll(async () => {
    app = await createE2eApp();
    userToken = await loginAndGetToken(app, 'user@user.com', 'admin');
    adminToken = await loginAndGetToken(app, 'admin@admin.com', 'admin');
  });

  afterAll(async () => {
    await app.close();
  });

  it('GET /api/v1/perks returns list', async () => {
    const res = await request(app.getHttpServer()).get('/api/v1/perks').expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/users/profile requires auth', async () => {
    await request(app.getHttpServer()).get('/api/v1/users/profile').expect(401);
  });

  it('GET /api/v1/users/profile returns current user with token', async () => {
    const res = await request(app.getHttpServer())
      .get('/api/v1/users/profile')
      .set('Authorization', `Bearer ${userToken}`)
      .expect(200);

    expect(res.body).toEqual(
      expect.objectContaining({
        email: 'user@user.com',
      }),
    );
  });

  it('GET /api/v1/users forbids regular user', async () => {
    await request(app.getHttpServer())
      .get('/api/v1/users')
      .set('Authorization', `Bearer ${userToken}`)
      .expect(403);
  });

  it('GET /api/v1/users allows admin', async () => {
    const res = await request(app.getHttpServer())
      .get('/api/v1/users')
      .set('Authorization', `Bearer ${adminToken}`)
      .expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });
});

