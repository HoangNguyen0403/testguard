import { render, screen } from '@testing-library/react';
import { add, login, Panel } from './sut';

describe('when admin', () => {
  beforeEach(() => login('admin'));
  it('shows the panel', () => {
    render(<Panel />);
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible();
  });
});

describe('when owner', () => {
  beforeEach(() => login('owner'));
  it('shows the panel', () => {
    render(<Panel />);
    expect(screen.getByRole('button', { name: 'Delete' })).toBeVisible();
  });
});

describe('add boundaries', () => {
  it('adds small numbers', () => {
    expect(add(1, 2)).toBe(3);
  });
  it('adds zero', () => {
    expect(add(0, 0)).toBe(0);
  });
  it('true copy-paste one', () => {
    expect(add(5, 5)).toBe(10);
  });
  it('true copy-paste two', () => {
    expect(add(5, 5)).toBe(10);
  });
});

describe('tautologies', () => {
  it('same identifier', () => {
    const x = add(1, 2);
    expect(x).toBe(x);
  });
  it('same literal', () => {
    expect(1).toBe(1);
  });
  it('negated literal', () => {
    expect(true).not.toBe(false);
  });
  it('constant arithmetic', () => {
    expect(2 + 2).toBe(4);
  });
});
