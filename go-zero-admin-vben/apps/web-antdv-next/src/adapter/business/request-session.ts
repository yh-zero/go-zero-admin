export function sessionRequestOptions(token?: string) {
  return token === undefined
    ? undefined
    : { headers: { Authorization: `Bearer ${token}` } };
}
