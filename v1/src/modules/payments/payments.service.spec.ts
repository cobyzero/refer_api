import { PaymentsService } from './payments.service';

describe('PaymentsService', () => {
  it('creates payment intent in cents and returns client secret', async () => {
    const configService = { get: jest.fn().mockReturnValue('sk_test_key') };
    const service = new PaymentsService(configService as any);
    
    const createMock = jest.fn().mockResolvedValue({ client_secret: 'pi_secret' });
    (service as any).stripe = {
      paymentIntents: {
        create: createMock,
      },
    };

    const result = await service.createPaymentIntent(10.25, 'usd');
    expect(createMock).toHaveBeenCalledWith(
      expect.objectContaining({
        amount: 1025,
        currency: 'usd',
      }),
    );
    expect(result).toEqual({ clientSecret: 'pi_secret' });
  });
});

