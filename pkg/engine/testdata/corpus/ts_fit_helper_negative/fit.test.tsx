import { fit } from './sut';

it('uses a helper named fit on an array', () => {
  const model = fit([1, 2, 3]);
  expect(model.length).toBe(3);
});
