import type { AccountInfoResponse } from './account.ts';
import { getCurrencyFraction } from '@/lib/currency.ts';
export interface AccountGroup { id: string; name: string; version: number }
export interface AccountGroupMember { accountId: string; groupId: string }
export interface AccountGroupsResponse { groups: AccountGroup[]; members: AccountGroupMember[] }
export interface GroupTotal { currency: string; assets: bigint; liabilities: bigint; net: bigint }
export function groupTotals(accounts: AccountInfoResponse[], includeHidden = false, excluded: Record<string, boolean> = {}, displayBalances: Record<string, number> = {}): GroupTotal[] {
    const totals = new Map<string, GroupTotal>();
    for (const root of accounts) {
        if ((!includeHidden && root.hidden) || excluded[root.id]) continue;
        // Count leaf balances once; a multi-account parent's balance is only a rollup.
        const leaves = root.type === 2 ? (root.subAccounts || []) : [root];
        for (const leaf of leaves) {
            if ((!includeHidden && leaf.hidden) || excluded[leaf.id]) continue;
            if (!Number.isSafeInteger(leaf.balance)) throw new Error('账户金额超出安全范围，暂不汇总');
            const total = totals.get(leaf.currency) || { currency: leaf.currency, assets: 0n, liabilities: 0n, net: 0n };
            const amount = BigInt(displayBalances[leaf.id] ?? leaf.balance);
            if (root.category === 3 || root.category === 5) total.liabilities -= amount;
            else total.assets += amount;
            total.net = total.assets - total.liabilities;
            totals.set(leaf.currency, total);
        }
    }
    return [...totals.values()].sort((a, b) => a.currency.localeCompare(b.currency));
}
export function groupMoney(amount: bigint, currency: string): string {
    const fraction = getCurrencyFraction(currency);
    if (fraction === undefined) return currency + '（未知币种，未换算）';
    const negative = amount < 0n;
    const digits = (negative ? -amount : amount).toString().padStart(fraction + 1, '0');
    const value = fraction ? digits.slice(0, -fraction) + '.' + digits.slice(-fraction) : digits;
    return `${currency} ${negative ? '-' : ''}${value}`;
}
