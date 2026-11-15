const { sum } = require('./sut');

function test(a, b) {
  return a + b;
}

const matcher = {
  test(value) {
    return value > 0;
  },
};

const rules = [{ test: /\.css$/, use: 'css-loader' }];

describe('declaration shapes', () => {
  it('L16 uses local function named test', () => {
    expect(test(1, 2)).toBe(3);
  });

  it('L20 comparison chain in js file', () => {
    const a = 1, b = 2, c = 3;
    const ok = a < b && b > a || c <b;
    expect(ok).toBe(true);
  });

  it('L26 object matcher', () => {
    expect(matcher.test(1)).toBe(true);
    expect(rules).toHaveLength(1);
  });
});
