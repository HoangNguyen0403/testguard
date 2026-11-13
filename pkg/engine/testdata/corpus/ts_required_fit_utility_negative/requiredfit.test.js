const { fit } = require('./chart-layout');
const { sum, bounds } = require('./sut');

it('required fit utility with a callback is not a focused test', () => {
  const scale = fit(bounds, () => 2);
  expect(scale).toBe(2);
});

it('a genuine declaration still analyses normally', () => {
  expect(sum(2, 2)).toBe(4);
});
