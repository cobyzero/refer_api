import { AppConfigService } from './app-config.service';

describe('AppConfigService', () => {
  it('returns app configuration with expected top-level keys', () => {
    const service = new AppConfigService();
    const config = service.getAppConfig();

    expect(config).toHaveProperty('version');
    expect(config).toHaveProperty('minVersion');
    expect(config).toHaveProperty('maintenance');
    expect(config).toHaveProperty('features');
    expect(config).toHaveProperty('endpoints');
  });
});

