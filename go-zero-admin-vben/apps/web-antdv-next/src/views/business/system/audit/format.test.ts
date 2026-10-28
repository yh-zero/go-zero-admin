import { describe, expect, it } from 'vitest';

import { auditQuery, safeAuditParams } from './format';

describe('audit filters and safe detail fields', () => {
  it('serializes an explicit timezone to RFC3339 without leaking the form-only range', () => {
    expect(
      auditQuery({
        actorName: 'admin',
        pageNo: 2,
        timeRange: ['2026-10-02T08:00:00+08:00', '2026-10-02T09:00:00+08:00'],
      }),
    ).toEqual({
      actorName: 'admin',
      pageNo: 2,
      startTime: '2026-10-02T00:00:00.000Z',
      endTime: '2026-10-02T01:00:00.000Z',
    });
    expect(auditQuery({ pageNo: 1 })).toEqual({ pageNo: 1 });
    expect(() => auditQuery({ timeRange: ['invalid', 'invalid'] })).toThrow(
      '时间范围',
    );
    expect(() =>
      auditQuery({
        timeRange: ['2026-10-02T09:00:00Z', '2026-10-02T08:00:00Z'],
      }),
    ).toThrow('开始时间');
  });
  it('shows numeric identifiers but never passwords, tokens, OTPs or arbitrary strings', () => {
    const result = safeAuditParams(
      JSON.stringify({
        id: 7,
        authorityIds: [1, 2],
        menuIds: '3,4',
        method: 'PUT',
        password: 'credential',
        accessToken: 'jwt',
        captcha: 'otp',
        email: 'private',
        status: 'credential-hidden-under-allowed-key',
        ids: ['token'],
        description: 'sensitive',
      }),
    );
    expect(result.map((entry) => entry.value)).toEqual([
      '7',
      '1, 2',
      '3,4',
      'PUT',
    ]);
    expect(safeAuditParams('{}')).toEqual([]);
    expect(safeAuditParams('not-json')).toEqual([]);
    expect(safeAuditParams('[1,2]')).toEqual([]);
  });
});
