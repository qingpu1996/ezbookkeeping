import type { AccountInfoResponse } from './account';
export interface ValuationSettings { positionId: string; version: number; mode: 'cost' | 'manual' | 'automatic'; manualPrice: string; manualAsOf: number; maxAgeMinutes: number }
export interface ValuationSettingsResponse { settings: ValuationSettings; automaticAvailable: boolean; automaticName: string; automaticMaxAgeMinutes: number }
export interface InvestmentValuation {
 positionId: string; accountId: string; positionVersion: number; cost: string; quantity: string; mode: string; status: string;
 marketValue: string | null; unrealized?: string; unitPrice?: string; sourceName: string; fetchedAt?: string; validUntil: number; reason?: string;
 quote?: { source: string; unit: string; fetchedAt: string; apiNowTime: string; marketDay: boolean; quotedPrice: boolean };
}
export function valuationBalances(accounts: AccountInfoResponse[], rows: InvestmentValuation[], now: number): Record<string, number> {
 const leaves = accounts.flatMap(a => a.type === 2 ? a.subAccounts || [] : [a]);
 const map: Record<string, number> = {};
 for (const row of rows) {
  const a=leaves.find(a=>a.id===row.accountId && a.investmentPositionId===row.positionId);
  if (!a || a.currency !== 'CNY' || row.status !== 'reference' || row.marketValue === null || row.validUntil * 1000 <= now || String(a.balance)!==row.cost) continue;
  if (!/^\d+$/.test(row.marketValue)) continue;
  const amount=Number(row.marketValue);
  if (Number.isSafeInteger(amount) && amount<=9999999999999) map[a.id]=amount;
 }
 return map;
}
