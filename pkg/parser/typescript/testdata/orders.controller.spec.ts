import { Test } from '@nestjs/testing';
import { TypeOrmModule } from '@nestjs/typeorm';
import { OrdersController } from './orders.controller';
import { OrdersService } from './orders.service';

describe('OrdersController', () => {
  let controller: OrdersController;
  const service = { findAll: jest.fn() };

  beforeEach(async () => {
    const mod = await Test.createTestingModule({
      imports: [TypeOrmModule.forRoot({ type: 'postgres', host: 'db.internal', database: 'prod' })],
      controllers: [OrdersController],
      providers: [{ provide: OrdersService, useValue: service }],
    }).compile();
    controller = mod.get(OrdersController);
  });

  it('echoes the stub through an assigned result', async () => {
    const orders = [{ id: 1 }];
    service.findAll.mockResolvedValue(orders);
    const result = await controller.findAll();
    expect(result).toEqual(orders);
  });

  it('echoes the stub inline', async () => {
    const orders = [{ id: 1 }];
    service.findAll.mockResolvedValue(orders);
    expect(await controller.findAll()).toEqual(orders);
  });
});
