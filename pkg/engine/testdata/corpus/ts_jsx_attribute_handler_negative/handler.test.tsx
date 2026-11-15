import { render, screen, fireEvent } from '@testing-library/react';
import { Button, sum } from './sut';

describe('named handlers in a JSX file', () => {
  it('handler referenced only from a JSX attribute', () => {
    function handleClick(e: { type: string }) {
      expect(e.type).toBe('click');
    }
    render(<Button onClick={handleClick}>Go</Button>);
    fireEvent.click(screen.getByRole('button'));
  });

  it('handler referenced only from a JSX child expression', () => {
    function renderRow(n: number) {
      expect(n).toBeGreaterThan(0);
      return <li key={n}>{n}</li>;
    }
    render(<ul>{[1, 2].map(renderRow)}</ul>);
  });

  it('overloaded generic checker', () => {
    function check<T>(v: T, want: T) {
      expect(v).toEqual(want);
    }
    check(sum(1, 1), 2);
  });
});
