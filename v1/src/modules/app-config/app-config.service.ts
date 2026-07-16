import { Injectable } from '@nestjs/common';

@Injectable()
export class AppConfigService {
  getAppConfig() {
    return {
      version: '1.0.0',
      minVersion: '1.0.0',
      maintenance: false,
      features: {
        starsEnabled: true,
        referralsEnabled: true,
        authEnabled: true,
      },
      endpoints: {
        termsUrl: 'https://example.com/terms',
        privacyUrl: 'https://example.com/privacy',
        supportUrl: 'https://example.com/support',
      }
    };
  }
}
