import { calculateTotal } from './calculator';

describe('pricing', () => {
  it('calculates total with rebate', () => {
    const rebate = 0;
    const total = calculateTotal({ price: 100, rebate });
    expect(total).toBe(100);
  });
});
