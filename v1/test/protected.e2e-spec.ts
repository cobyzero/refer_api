import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { createE2eApp, loginAndGetToken } from './e2e-utils';

describe('Protected Endpoints (e2e)', () => {
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

  it('GET /api/v1/stars/balance requires auth', async () => {
    await request(app.getHttpServer()).get('/api/v1/stars/balance').expect(401);
  });

  it('GET /api/v1/stars/balance works with token', async () => {
    await request(app.getHttpServer())
      .get('/api/v1/stars/balance')
      .set('Authorization', `Bearer ${userToken}`)
      .expect(200);
  });

  it('GET /api/v1/orders requires auth', async () => {
    await request(app.getHttpServer()).get('/api/v1/orders').expect(401);
  });

  it('GET /api/v1/orders works with user token', async () => {
    const res = await request(app.getHttpServer())
      .get('/api/v1/orders')
      .set('Authorization', `Bearer ${userToken}`)
      .expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/dashboard/summary works with token', async () => {
    const res = await request(app.getHttpServer())
      .get('/api/v1/dashboard/summary')
      .set('Authorization', `Bearer ${userToken}`)
      .expect(200);
    expect(res.body).toEqual(
      expect.objectContaining({
        user: expect.any(Object),
        stars: expect.any(Number),
      }),
    );
  });

  it('GET /api/v1/dashboard/admin/summary forbids regular user', async () => {
    await request(app.getHttpServer())
      .get('/api/v1/dashboard/admin/summary')
      .set('Authorization', `Bearer ${userToken}`)
      .expect(403);
  });

  it('GET /api/v1/dashboard/admin/summary allows admin', async () => {
    const res = await request(app.getHttpServer())
      .get('/api/v1/dashboard/admin/summary')
      .set('Authorization', `Bearer ${adminToken}`)
      .expect(200);

    expect(res.body).toEqual(
      expect.objectContaining({
        stats: expect.any(Array),
        recentActivity: expect.any(Array),
      }),
    );
  });
});

