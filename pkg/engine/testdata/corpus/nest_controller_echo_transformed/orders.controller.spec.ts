it('transforms stub result', async () => {
  const orders = [{ id: 1 }];
  service.findAll.mockResolvedValue(orders);
  const result = await controller.findAll();
  expect(result).toEqual([{ id: 1, processed: true }]);
});
