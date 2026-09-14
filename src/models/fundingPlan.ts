import type { AccountInfoResponse } from './account';
import { AccountCategory } from '@/core/account';
export interface FundingPlan {
 configured: boolean; currency: string; reserve: string; accountIds: string[]; version: number;
}
export function fundingLeaves(accounts: AccountInfoResponse[]): AccountInfoResponse[] {
 return accounts.flatMap(a => a.type === 2 ? (a.subAccounts || []).map(c => ({ ...c, name: `${a.name} / ${c.name}` })) : [a]);
}
export function eligibleFundingAccount(a: AccountInfoResponse, currency: string): boolean {
 return a.type === 1 && !a.investmentPositionId && !!AccountCategory.valueOf(a.category)?.isAsset && a.currency === currency;
}
// Integer minor units throughout. No quote, cost, credit limit or FX fallback enters cash planning.
export function fundingSummary(accounts: AccountInfoResponse[], plan: FundingPlan) {
 let included = 0n, excluded = 0n;
 const leaves = fundingLeaves(accounts), selected = new Set(plan.accountIds);
 const invalid = plan.accountIds.some(id => !leaves.some(a => a.id === id && eligibleFundingAccount(a, plan.currency)));
 for (const a of leaves) {
  if (!eligibleFundingAccount(a, plan.currency)) continue;
  if (!Number.isSafeInteger(a.balance)) throw new Error('账户余额超出可计算范围');
  if (selected.has(a.id)) included += BigInt(a.balance); else excluded += BigInt(a.balance);
 }
 const reserve = plan.configured ? BigInt(plan.reserve) : 0n;
 const remainder = included - reserve;
 return { included, excluded, reserve, available: remainder > 0n ? remainder : 0n, shortfall: remainder < 0n ? -remainder : 0n, ready: plan.configured && !invalid, invalid };
}
