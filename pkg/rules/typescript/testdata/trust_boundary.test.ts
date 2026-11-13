import { expect, it, vi } from 'vitest';
it('asserts on both exhaustive branches', () => {
  const result = parse(input);
  if (result.ok) {
    expect(result.value).toEqual(42);
  } else {
    expect(result.error.code).toBe('INVALID');
  }
});
it('commented invocation is not self mocking', () => {
  const callback = vi.fn();
  // callback();
  emitter.subscribe(callback);
  emitter.emit('ready');
  expect(callback).toHaveBeenCalledWith('ready');
});
it('quoted invocation is not self mocking', () => {
  const callback = vi.fn();
  const documentation = 'callback()';
  emitter.subscribe(callback);
  emitter.emit(documentation);
  expect(callback).toHaveBeenCalledWith(documentation);
});
