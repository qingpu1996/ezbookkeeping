import { useInvestmentValuations } from '@/composables/useInvestmentValuations';
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';

import type { HiddenAmount, NumberWithSuffix } from '@/core/numeral.ts';
import type { WeekDayValue } from '@/core/datetime.ts';
import { AccountCategory, AccountType } from '@/core/account.ts';
import type { Account, CategorizedAccount } from '@/models/account.ts';

import { isObject, isNumber, isString } from '@/lib/common.ts';

export function useAccountListPageBase() {
    const { formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

    const settingsStore = useSettingsStore();
    const userStore = useUserStore();
    const accountsStore = useAccountsStore();

    const loading = ref<boolean>(true);
    const showHidden = ref<boolean>(false);
    const displayOrderModified = ref<boolean>(false);

    const showAccountBalance = computed<boolean>({
        get: () => settingsStore.appSettings.showAccountBalance,
        set: (value) => settingsStore.setShowAccountBalance(value)
    });

    const customAccountCategoryOrder = computed<string>(() => settingsStore.appSettings.accountCategoryOrders);
    const defaultAccountCategory = computed<AccountCategory>(() => AccountCategory.values(customAccountCategoryOrder.value)[0] ?? AccountCategory.Default);

    const firstDayOfWeek = computed<WeekDayValue>(() => userStore.currentUserFirstDayOfWeek);
    const fiscalYearStart = computed<number>(() => userStore.currentUserFiscalYearStart);
    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
    const useLastReconciledTime = computed(() => userStore.currentUserUseLastReconciledTime);

    const allAccounts = computed<Account[]>(() => accountsStore.allAccounts);
    const { balances: displayBalances, note: valuationNote, refresh: refreshValuations } = useInvestmentValuations(allAccounts);
    const allCategorizedAccountsMap = computed<Record<number, CategorizedAccount>>(() => accountsStore.allCategorizedAccountsMap);
    const allAccountCount = computed<number>(() => accountsStore.allAvailableAccountsCount);
    const maxCategoryAccountCount = computed<number>(() => accountsStore.maxCategoryAccountCount);

    const netAssets = computed<string>(() => {
        const netAssets: number | HiddenAmount | NumberWithSuffix = accountsStore.getNetAssets(showAccountBalance.value, displayBalances.value);
        return formatAmountToLocalizedNumeralsWithCurrency(netAssets, defaultCurrency.value);
    });

    const totalAssets = computed<string>(() => {
        const totalAssets: number | HiddenAmount | NumberWithSuffix = accountsStore.getTotalAssets(showAccountBalance.value, displayBalances.value);
        return formatAmountToLocalizedNumeralsWithCurrency(totalAssets, defaultCurrency.value);
    });

    const cashAssets = computed(() => formatAmountToLocalizedNumeralsWithCurrency(accountsStore.getTotalAssets(showAccountBalance.value, displayBalances.value, 'cash'), defaultCurrency.value));
    const investmentAssets = computed(() => formatAmountToLocalizedNumeralsWithCurrency(accountsStore.getTotalAssets(showAccountBalance.value, displayBalances.value, 'investment'), defaultCurrency.value));

    const totalLiabilities = computed<string>(() => {
        const totalLiabilities: number | HiddenAmount | NumberWithSuffix = accountsStore.getTotalLiabilities(showAccountBalance.value, displayBalances.value);
        return formatAmountToLocalizedNumeralsWithCurrency(totalLiabilities, defaultCurrency.value);
    });

    function accountCategoryTotalBalance(accountCategory?: AccountCategory): string {
        if (!accountCategory) {
            return '';
        }

        const totalBalance: number | HiddenAmount | NumberWithSuffix = accountsStore.getAccountCategoryTotalBalance(showAccountBalance.value, accountCategory, displayBalances.value);
        return formatAmountToLocalizedNumeralsWithCurrency(totalBalance, defaultCurrency.value);
    }

    function accountBalance(account: Account, currentSubAccountId?: string): string | null {
        if (account.type === AccountType.SingleAccount.type) {
            const balance: number| HiddenAmount | null = accountsStore.getAccountBalance(showAccountBalance.value, account, displayBalances.value);

            if (!isNumber(balance) && !isString(balance)) {
                return '';
            }

            return formatAmountToLocalizedNumeralsWithCurrency(balance, account.currency) + (account.investmentPositionId ? (displayBalances.value[account.id] !== undefined ? "（参考估值）" : "（成本）") : "");
        } else if (account.type === AccountType.MultiSubAccounts.type) {
            const balanceResult = accountsStore.getAccountSubAccountBalance(showAccountBalance.value, showHidden.value, account, currentSubAccountId, displayBalances.value);

            if (!isObject(balanceResult)) {
                return '';
            }

            return formatAmountToLocalizedNumeralsWithCurrency(balanceResult.balance, balanceResult.currency);
        } else {
            return null;
        }
    }

    return {
        valuationNote, refreshValuations,
        // states
        loading,
        showHidden,
        displayOrderModified,
        // computed states
        showAccountBalance,
        customAccountCategoryOrder,
        defaultAccountCategory,
        firstDayOfWeek,
        fiscalYearStart,
        defaultCurrency,
        useLastReconciledTime,
        allAccounts,
        allCategorizedAccountsMap,
        allAccountCount,
        maxCategoryAccountCount,
        cashAssets, investmentAssets,
        netAssets,
        totalAssets,
        totalLiabilities,
        // functions
        accountCategoryTotalBalance,
        accountBalance
    };
}
