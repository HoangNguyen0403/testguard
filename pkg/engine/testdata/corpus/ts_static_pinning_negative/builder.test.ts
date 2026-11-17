import { buildQuery } from './queryBuilder';

describe('query builder', () => {
  it('builds dynamic query with options', () => {
    const query = buildQuery({ includeRoles: true });
    expect(query).toContain('roles');
  });
});
