void verifyCartNotEmpty(Cart cart) {
  expect(cart.items.length, greaterThan(0));
}

void main() {
  test('cart contains items', () {
    verifyCartNotEmpty(myCart);
  });
}
