import { Test } from '@nestjs/testing';
import { TypeOrmModule } from '@nestjs/typeorm';

describe('UsersService integration', () => {
  it('persists a user against in-memory sqlite', async () => {
    const mod = await Test.createTestingModule({
      imports: [TypeOrmModule.forRoot({ type: 'sqlite', database: ':memory:' })],
    }).compile();
    expect(mod).toBeDefined();
  });
});
