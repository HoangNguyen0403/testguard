import { Test } from '@nestjs/testing';
import { TypeOrmModule } from '@nestjs/typeorm';
import { PrismaModule } from './prisma.module';
import { PrismaService } from './prisma.service';
import { UsersService } from './users.service';

describe('UsersService integration', () => {
  it('persists a user against in-memory sqlite', async () => {
    const mod = await Test.createTestingModule({
      imports: [TypeOrmModule.forRoot({ type: 'sqlite', database: ':memory:', synchronize: true })],
      providers: [UsersService],
    }).compile();
    const saved = await mod.get(UsersService).create({ name: 'a' });
    expect(saved.id).toBeGreaterThan(0);
  });

  it('overrides the prisma provider with a fake', async () => {
    const fake = { user: { findMany: async () => [{ id: 1 }] } };
    const mod = await Test.createTestingModule({ imports: [PrismaModule], providers: [UsersService] })
      .overrideProvider(PrismaService)
      .useValue(fake)
      .compile();
    expect(await mod.get(UsersService).list()).toHaveLength(1);
  });
});
