import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { createE2eApp } from './e2e-utils';

describe('Catalog and Search (e2e)', () => {
  let app: INestApplication;

  beforeAll(async () => {
    app = await createE2eApp();
  });

  afterAll(async () => {
    await app.close();
  });

  it('GET /api/v1/products returns list', async () => {
    const res = await request(app.getHttpServer()).get('/api/v1/products').expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/products/top-rated returns list', async () => {
    const res = await request(app.getHttpServer()).get('/api/v1/products/top-rated').expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/products/seasonal returns list', async () => {
    const res = await request(app.getHttpServer()).get('/api/v1/products/seasonal').expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/products/categories returns list', async () => {
    const res = await request(app.getHttpServer()).get('/api/v1/products/categories').expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/search/referrals returns list', async () => {
    const res = await request(app.getHttpServer())
      .get('/api/v1/search/referrals')
      .query({ q: 'a', category: 'All' })
      .expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/search/categories returns list', async () => {
    const res = await request(app.getHttpServer()).get('/api/v1/search/categories').expect(200);
    expect(Array.isArray(res.body)).toBe(true);
  });

  it('GET /api/v1/locations and /:id returns location', async () => {
    const all = await request(app.getHttpServer()).get('/api/v1/locations').expect(200);
    expect(Array.isArray(all.body)).toBe(true);
    expect(all.body.length).toBeGreaterThan(0);

    const one = await request(app.getHttpServer()).get(`/api/v1/locations/${all.body[0].id}`).expect(200);
    expect(one.body).toEqual(expect.objectContaining({ id: all.body[0].id }));
  });
});

