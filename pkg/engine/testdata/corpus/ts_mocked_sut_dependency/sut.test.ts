import { Sut } from './sut';
jest.mock('./dependency');

it('tests sut with mocked dependency', () => {
  expect(1).toBe(1);
});
