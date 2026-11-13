import { sum } from './sut';

const suffix = describeScenario();
const foo = expressionTitle();

it('foo' + suffix, () => {
  expect(sum(1, 2)).toBe(3);
});

it(`foo ${suffix}`, () => {
  expect(sum(2, 2)).toBe(4);
});

it(foo, () => {
  expect(sum(3, 3)).toBe(6);
});
