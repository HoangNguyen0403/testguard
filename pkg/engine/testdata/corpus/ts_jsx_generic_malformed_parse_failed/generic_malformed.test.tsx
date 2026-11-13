import React from 'react';
import { render } from '@testing-library/react';
import { Table, sum } from './sut';

it('malformed generic element', () => {
  render(<Table<{ n: number } rows={[{ n: 1 }]} />);
  expect(sum(1, 2)).toBe(3);
});
