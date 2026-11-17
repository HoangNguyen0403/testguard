import { print } from 'graphql';
import { GET_USER_QUERY } from './user.graphql';

describe('user query', () => {
  it('contains expected fields', () => {
    const queryStr = print(GET_USER_QUERY);
    expect(queryStr).toContain('userId');
  });
});
