import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Button, List, svc, fit, load } from './sut';

jest.mock('./api');
import { fetchItems } from './api';

describe('suite A', () => {
  const check = (v: number) => {
    expect(v).toBeGreaterThan(0);
  };

  it('L13 uses asserting helper from its own scope', () => {
    check(svc.count());
  });

  it('L17 same body different suite is not a clone', () => {
    expect(svc.count()).toBe(2);
  });
});

describe('suite B', () => {
  it('L23 same body different suite is not a clone', () => {
    expect(svc.count()).toBe(2);
  });

  it('L27 click handler mock invoked by component', () => {
    const onClick = jest.fn();
    render(<Button onClick={onClick}>Save</Button>);
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('L34 generic arrow and comparison in jsx file', () => {
    const id = <T,>(v: T): T => v;
    const ok = 1 < 2 && 3 > 2;
    expect(id(ok)).toBe(true);
  });

  it('L40 waitFor with assertion', async () => {
    (fetchItems as jest.Mock).mockResolvedValue(['a']);
    render(<List />);
    await waitFor(() => expect(screen.getByText('a')).toBeInTheDocument());
  });

  it('L46 fake timers with setTimeout', () => {
    jest.useFakeTimers();
    const cb = jest.fn();
    svc.schedule(cb, 1000);
    jest.advanceTimersByTime(1000);
    expect(cb).toHaveBeenCalledTimes(1);
  });

  it('L54 local function named fit is not a focused test', () => {
    const model = fit([1, 2, 3]);
    expect(model.loss).toBeLessThan(1);
  });

  it('L59 word skip and only as identifiers', () => {
    const opts = { skip: 2, only: true };
    expect(svc.page(opts.skip).length).toBe(3);
  });

  it('L64 determinism compares two calls', async () => {
    const first = await load('x');
    const second = await load('x');
    expect(first).toEqual(second);
  });

  it('L70 conditional rendering asserted in both branches with else-if', () => {
    const n = svc.count();
    if (n === 0) {
      expect(screen.queryByText('empty')).not.toBeNull();
    } else if (n === 1) {
      expect(screen.queryByText('one')).not.toBeNull();
    } else {
      expect(screen.queryByText('many')).not.toBeNull();
    }
  });

  it.each([
    [1, 1],
    [2, 2],
  ])('L81 table %i', (input, want) => {
    expect(svc.echo(input)).toBe(want);
  });
});
