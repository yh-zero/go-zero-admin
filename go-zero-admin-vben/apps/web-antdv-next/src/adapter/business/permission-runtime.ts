export interface EditTicket {
  token: string;
  objectId: number;
  epoch: number;
}
export function createEditSession(token: () => null | string) {
  let epoch = 0;
  let objectId = 0;
  const capture = (): EditTicket => ({ token: token() ?? '', objectId, epoch });
  return {
    capture,
    begin(id: number) {
      objectId = id;
      epoch++;
      return capture();
    },
    invalidate() {
      epoch++;
    },
    valid(ticket: EditTicket) {
      return (
        !!ticket.token &&
        ticket.token === token() &&
        ticket.epoch === epoch &&
        ticket.objectId === objectId
      );
    },
  };
}
export function compareRevision(a: string, b: string) {
  if (!/^\d+$/.test(a) || !/^\d+$/.test(b)) throw new Error('权限版本格式错误');
  const left = a.replace(/^0+(?=\d)/, '');
  const right = b.replace(/^0+(?=\d)/, '');
  if (left.length !== right.length) return left.length > right.length ? 1 : -1;
  return left === right ? 0 : left > right ? 1 : -1;
}
export function createPermissionRefresh<
  S extends { revision: string; fingerprint: string },
  C,
>(options: {
  token: () => null | string;
  load: () => Promise<S>;
  build: (snapshot: S) => Promise<C>;
  commit: (candidate: C, snapshot: S) => void;
}) {
  let epoch = 0;
  let session = '';
  let revision = '0';
  let fingerprint: string | undefined;
  return {
    reset() {
      epoch++;
      session = '';
      revision = '0';
      fingerprint = undefined;
    },
    async refresh() {
      const token = options.token() ?? '';
      if (!token) return;
      if (token !== session) {
        session = token;
        revision = '0';
        fingerprint = undefined;
      }
      const request = ++epoch;
      const valid = () =>
        token === options.token() && request === epoch && token === session;
      const snapshot = await options.load();
      if (!valid() || compareRevision(snapshot.revision, revision) < 0) return;
      if (
        fingerprint !== undefined &&
        compareRevision(snapshot.revision, revision) === 0
      )
        return;
      if (snapshot.fingerprint === fingerprint) {
        revision = snapshot.revision;
        return;
      }
      const candidate = await options.build(snapshot);
      if (!valid() || compareRevision(snapshot.revision, revision) < 0) return;
      options.commit(candidate, snapshot);
      revision = snapshot.revision;
      fingerprint = snapshot.fingerprint;
    },
    get revision() {
      return revision;
    },
  };
}
