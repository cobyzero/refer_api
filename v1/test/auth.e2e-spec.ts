import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { createE2eApp, loginAndGetToken } from './e2e-utils';

describe('Auth (e2e)', () => {
  let app: INestApplication;

  beforeAll(async () => {
    app = await createE2eApp();
  });

  afterAll(async () => {
    await app.close();
  });

  it('POST /api/v1/auth/login authenticates seeded admin', async () => {
    const res = await request(app.getHttpServer())
      .post('/api/v1/auth/login')
      .send({ email: 'admin@admin.com', password: 'admin' })
      .expect(200);

    expect(res.body).toEqual(
      expect.objectContaining({
        accessToken: expect.any(String),
        refreshToken: expect.any(String),
        user: expect.objectContaining({
          email: 'admin@admin.com',
        }),
      }),
    );
  });

  it('POST /api/v1/auth/login rejects invalid credentials', async () => {
    await request(app.getHttpServer())
      .post('/api/v1/auth/login')
      .send({ email: 'admin@admin.com', password: 'wrong-password' })
      .expect(401);
  });

  it('GET /api/v1/auth/validate returns 401 without token', async () => {
    await request(app.getHttpServer()).get('/api/v1/auth/validate').expect(401);
  });

  it('GET /api/v1/auth/validate returns payload with token', async () => {
    const token = await loginAndGetToken(app, 'admin@admin.com', 'admin');

    const res = await request(app.getHttpServer())
      .get('/api/v1/auth/validate')
      .set('Authorization', `Bearer ${token}`)
      .expect(200);

    expect(res.body).toEqual(
      expect.objectContaining({
        valid: true,
        user: expect.any(Object),
      }),
    );
  });
});

