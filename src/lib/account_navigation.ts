import { isInvestmentAssetsEnabled } from './server_settings.ts';
export function accountDestination(account: { id: string; investmentPositionId?: string } | null | undefined): string {
    if (!account) return '/account/list';
    return isInvestmentAssetsEnabled() && account.investmentPositionId ? '/account/gold?accountId=' + encodeURIComponent(account.id) : '/transaction/list?accountIds=' + encodeURIComponent(account.id);
}
export function canSetUpGold(account: { type: number; category: number; currency: string; balance: number; investmentPositionId?: string } | null | undefined): boolean {
    return !!account && isInvestmentAssetsEnabled() && account.type === 1 && account.category === 7 && account.currency === 'CNY' && account.balance === 0 && !account.investmentPositionId;
}
