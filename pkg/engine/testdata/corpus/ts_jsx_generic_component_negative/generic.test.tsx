import React from 'react';
import { render } from '@testing-library/react';
import { Table, sum } from './sut';

describe('generic component', () => {
  it('handler on a generic component', () => {
    function checkRow(row: { n: number }) {
      expect(sum(row.n, 0)).toBe(row.n);
    }
    render(<Table<{ n: number }> rows={[{ n: 1 }]} onRow={checkRow} />);
  });

  it('a later test survives the generic element', () => {
    expect(sum(2, 2)).toBe(4);
  });
});
