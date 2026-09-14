import { describe, it, expect } from 'vitest';
import { groupTotals, groupMoney } from './accountGroup.ts';
import type { AccountInfoResponse } from './account.ts';
const account = (id: string, balance: number, more: Partial<AccountInfoResponse> = {}): AccountInfoResponse => ({ id, name: id, parentId: '0', category: 2, type: 1, icon: '1', color: '112233', currency: 'CNY', balance, comment: '', displayOrder: 0, hidden: false, ...more });
describe('institution group totals', () => {
    it('combines asset types and debts without mixing currencies or double counting parents', () => {
        const list = [account('cash', 800000), account('gold', 900000, { category: 7 }), account('credit', -300000, { category: 3 }), account('parent', 999999, { type: 2, subAccounts: [account('child', 12345, { parentId: 'parent', currency: 'USD' })] })];
        expect(groupTotals(list)).toEqual([{ currency: 'CNY', assets: 1700000n, liabilities: 300000n, net: 1400000n }, { currency: 'USD', assets: 12345n, liabilities: 0n, net: 12345n }]);
    });
    it('respects hidden parents/children and total exclusions', () => {
        const list = [account('hidden', 100, { hidden: true }), account('exclude', 200), account('p', 500, { type: 2, subAccounts: [account('c', 500, { hidden: true })] })];
        expect(groupTotals(list, false, { exclude: true })).toEqual([]);
        expect(groupTotals(list, true, { exclude: true })[0]?.assets).toBe(600n);
        expect(groupTotals([account('p', 0, { hidden: true, type: 2, subAccounts: [account('c', 1)] })])).toEqual([]);
    });
    it('handles credit card overpayments using existing liability convention', () => {
        expect(groupTotals([account('card', 100, { category: 3 })])[0]).toEqual({ currency: 'CNY', assets: 0n, liabilities: -100n, net: 100n });
    });
    it('keeps sums exact past Number precision and supports currency decimal places', () => {
        const total = groupTotals([account('a', Number.MAX_SAFE_INTEGER), account('b', 1)])[0]!;
        expect(total.assets).toBe(9007199254740992n);
        expect(groupMoney(total.assets, 'CNY')).toBe('CNY 90071992547409.92');
        expect(groupMoney(123n, 'JPY')).toBe('JPY 123');
        expect(groupMoney(-1234n, 'KWD')).toBe('KWD -1.234');
        expect(() => groupTotals([account('bad', Number.MAX_SAFE_INTEGER + 1)])).toThrow();
    });
});
