it('echoes assigned stub result', async () => {
  const orders = [{ id: 1 }];
  service.findAll.mockResolvedValue(orders);
  const result = await controller.findAll();
  expect(result).toEqual(orders);
});
