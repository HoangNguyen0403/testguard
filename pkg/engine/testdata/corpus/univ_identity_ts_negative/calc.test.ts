import { calculateTotal } from './calculator';

describe('pricing', () => {
  it('calculates total with non-zero rebate', () => {
    const rebate = 20;
    const total = calculateTotal({ price: 100, rebate });
    expect(total).toBe(80);
  });

  it('handles zero rebate boundary', () => {
    const rebate = 0;
    const total = calculateTotal({ price: 100, rebate });
    expect(total).toBe(100);
  });
});
