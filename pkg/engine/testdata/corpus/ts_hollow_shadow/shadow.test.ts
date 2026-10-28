function check() {
  expect(1).toBe(1);
}

it('hollow shadow', () => {
  const check = () => {};
  check();
});
