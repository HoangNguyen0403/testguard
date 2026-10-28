function expectCartEmpty() {
  expect(cart.items.length).toBe(0);
}

it('clears the cart', () => {
  expectCartEmpty();
});
