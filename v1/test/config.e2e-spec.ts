import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { createE2eApp } from './e2e-utils';

describe('Config (e2e)', () => {
  let app: INestApplication;

  beforeAll(async () => {
    app = await createE2eApp();
  });

  afterAll(async () => {
    await app.close();
  });

  it('GET /api/v1/config returns app config', async () => {
    await request(app.getHttpServer())
      .get('/api/v1/config')
      .expect(200)
      .expect((res) => {
        expect(res.body).toEqual(
          expect.objectContaining({
            version: expect.any(String),
            minVersion: expect.any(String),
            maintenance: expect.any(Boolean),
            features: expect.any(Object),
            endpoints: expect.any(Object),
          }),
        );
      });
  });
});

