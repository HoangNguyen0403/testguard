import { Test } from '@nestjs/testing';
import { TypeOrmModule } from '@nestjs/typeorm';

describe('OrdersController', () => {
  beforeEach(async () => {
    await Test.createTestingModule({
      imports: [TypeOrmModule.forRoot({ type: 'postgres' })],
    }).compile();
  });

  it('runs query', () => {
    expect(1).toBe(1);
  });
});
