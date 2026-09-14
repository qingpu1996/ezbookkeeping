<template>
    <section class="institution-groups" :aria-busy="busy">
        <header><div><p class="eyebrow">账户整理</p><h1>按机构看清你的钱</h1></div><button :disabled="busy" @click="reload">刷新</button></header>
        <p class="intro">把同一家银行或平台的账户放在一起。分组不记账，也不会改变余额、分类或黄金持仓。</p>
        <p v-if="valuationNote" role="status">{{ valuationNote }} <button @click="refreshValuations">刷新估值</button></p><p v-if="error" role="alert" class="error">{{ error }}</p><p v-if="message" role="status" class="message">{{ message }}</p>
        <form class="create-group" @submit.prevent="create"><label>新分组名称<input v-model="newName" maxlength="64" placeholder="例如：招商银行、支付宝" required :disabled="busy" /></label><button :disabled="busy || !newName.trim() || !loaded">创建分组</button></form>
        <label class="group-visibility"><input v-model="showHidden" type="checkbox" :disabled="busy" />显示隐藏账户并纳入本页汇总</label>
        <p class="help">不同币种分别统计，不自动换汇。黄金依据账户估值设置显示参考估值或成本；净额不等于可投资资金。已设置“不计入总金额”的账户不计入本页汇总。</p>
        <p v-if="!loaded && busy" role="status">正在读取账户与分组…</p>
        <template v-if="loaded">
            <article v-for="section in sections" :key="section.id" class="group-card">
                <header><div><h2>{{ section.name }}</h2><small>{{ section.accounts.length }} 个顶层账户</small></div><details v-if="section.group"><summary>管理分组</summary><form @submit.prevent="rename(section.group)"><label>分组名称<input v-model="names[section.id]" maxlength="64" required :disabled="busy" /></label><button :disabled="busy">保存名称</button></form><button class="danger" :disabled="busy" @click="deleting = section.id">删除分组</button></details></header>
                <div v-if="section.group && deleting === section.id" class="delete-confirm" role="alert"><p>只删除“{{ section.name }}”这个分组。组内账户回到“未分组”，所有余额和交易记录保留。</p><button :disabled="busy" @click="remove(section.group!)">确认删除分组</button><button :disabled="busy" @click="deleting = ''">取消</button></div>
                <div v-for="total in section.totals" :key="total.currency" class="totals"><div><small>资产</small><strong>{{ groupMoney(total.assets, total.currency) }}</strong></div><div><small>负债</small><strong>{{ groupMoney(total.liabilities, total.currency) }}</strong></div><div><small>净额</small><strong>{{ groupMoney(total.net, total.currency) }}</strong></div></div>
                <p v-if="!section.accounts.length" class="empty">{{ section.id ? '还没有账户。从下方“未分组”选择账户归属即可。' : '没有未分组账户。新增账户后会出现在这里。' }}</p>
                <div v-for="account in section.accounts" :key="account.id" class="account-row">
                    <div><h3><button class="account-open" @click="emit('openAccount', accountDestination(account))">{{ account.name }}</button><small v-if="account.hidden"> · 已隐藏</small></h3><p>{{ tt(AccountCategory.valueOf(account.category)?.name || '') }}<span v-if="account.investmentPositionId"> · 黄金账户</span><span v-if="excluded[account.id]"> · 不计入汇总</span></p>
                        <strong v-if="account.type === 1">{{ accountMoney(account) }}<small v-if="account.investmentPositionId"> · {{ displayBalances[account.id] !== undefined ? "参考估值" : "持仓成本" }}</small></strong>
                        <ul v-else><li v-for="child in visibleChildren(account)" :key="child.id"><button class="account-open" @click="emit('openAccount', accountDestination(child))">{{ child.name }}</button> · {{ accountMoney(child) }}<span v-if="child.hidden"> · 已隐藏</span><span v-if="excluded[child.id]"> · 不计入汇总</span></li></ul>
                    </div>
                    <form @submit.prevent="assign(account)"><label :for="'institution-' + account.id">机构归属<select :id="'institution-' + account.id" v-model="selected[account.id]" :disabled="busy"><option value="">未分组</option><option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option></select></label><button :disabled="busy || selected[account.id] === membership[account.id]">保存归属</button><small v-if="account.type === 2">子账户随父账户一起移动</small></form>
                </div>
            </article>
        </template>
    </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useInvestmentValuations } from '@/composables/useInvestmentValuations';
import { accountDestination } from '@/lib/account_navigation.ts';
const emit = defineEmits<{ openAccount: [path: string] }>();
import services from '@/lib/services.ts';
import type { AccountInfoResponse } from '@/models/account.ts';
import { groupMoney, groupTotals, type AccountGroup } from '@/models/accountGroup.ts';
import { AccountCategory } from '@/core/account.ts';
import { useI18n } from '@/locales/helpers.ts';
import { useSettingsStore } from '@/stores/setting.ts';
const { tt } = useI18n();
const settings = useSettingsStore();
const excluded = computed(() => settings.appSettings.totalAmountExcludeAccountIds);
const groups = ref<AccountGroup[]>([]), accounts = ref<AccountInfoResponse[]>([]);
const { balances: displayBalances, note: valuationNote, refresh: refreshValuations } = useInvestmentValuations(accounts);
const membership = ref<Record<string, string>>({}), selected = ref<Record<string, string>>({}), names = ref<Record<string, string>>({});
const busy = ref(false), loaded = ref(false), showHidden = ref(false), newName = ref(''), deleting = ref(''), error = ref(''), message = ref('');
const sections = computed(() => [...groups.value.map(group => ({ id: group.id, name: group.name, group })), { id: '', name: '未分组', group: undefined }].map(section => {
    const list = accounts.value.filter(account => membership.value[account.id] === section.id && (showHidden.value || !account.hidden));
    return { ...section, accounts: list, totals: groupTotals(list, showHidden.value, excluded.value, displayBalances.value) };
}));
function visibleChildren(account: AccountInfoResponse): AccountInfoResponse[] { return (account.subAccounts || []).filter(child => showHidden.value || !child.hidden); }
function accountMoney(account: AccountInfoResponse): string { return groupMoney(BigInt(displayBalances.value[account.id] ?? account.balance) * ((account.isLiability ?? AccountCategory.valueOf(account.category)?.isLiability) ? -1n : 1n), account.currency); }
async function fetchData(): Promise<void> {
    const [groupResponse, accountResponse] = await Promise.all([services.getAccountGroups(), services.getAllAccounts({ visibleOnly: false })]);
    const nextGroups = groupResponse.data.result.groups;
    const ids = new Set(nextGroups.map(group => group.id));
    const nextAccounts = accountResponse.data.result.filter(account => !account.parentId || account.parentId === '0');
    // Validate before replacing the page snapshot; keep loading failures explicit.
    groupTotals(nextAccounts, true);
    const members: Record<string, string> = Object.fromEntries(nextAccounts.map(account => [account.id, '']));
    for (const member of groupResponse.data.result.members) if (member.accountId in members && ids.has(member.groupId)) members[member.accountId] = member.groupId;
    groups.value = nextGroups; accounts.value = nextAccounts; membership.value = members; selected.value = { ...members };
    names.value = Object.fromEntries(nextGroups.map(group => [group.id, group.name])); loaded.value = true;
}
async function run(action: () => Promise<void>): Promise<void> {
    if (busy.value) return;
    busy.value = true; error.value = ''; message.value = '';
    try { await action(); } catch { loaded.value = false; error.value = '操作未完成或结果尚未确认，请刷新后核对。账户余额不会因分组而改变。'; } finally { busy.value = false; }
}
async function reload(): Promise<void> { await run(fetchData); }
async function create(): Promise<void> { await run(async () => { await services.saveAccountGroup({ name: newName.value.trim() }); await fetchData(); newName.value = ''; message.value = '分组已创建。'; }); }
async function rename(group: AccountGroup): Promise<void> { await run(async () => { await services.saveAccountGroup({ id: group.id, version: group.version, name: names.value[group.id] || '' }); await fetchData(); message.value = '名称已保存。'; }); }
async function remove(group: AccountGroup): Promise<void> { await run(async () => { await services.deleteAccountGroup({ id: group.id, version: group.version }); await fetchData(); deleting.value = ''; message.value = '分组已删除，账户和账目保留。'; }); }
async function assign(account: AccountInfoResponse): Promise<void> { await run(async () => { await services.assignAccountGroup({ accountId: account.id, groupId: selected.value[account.id] || '', expectedGroupId: membership.value[account.id] || '' }); await fetchData(); message.value = '机构归属已保存。'; }); }
onMounted(reload);
</script>
<style scoped>
.account-open{padding:0;border:0;background:transparent;color:inherit;text-align:left;min-height:32px;text-decoration:underline;text-underline-offset:3px}.institution-groups{max-width:1020px;margin:auto;padding:24px;color:var(--f7-text-color,inherit)}header{display:flex;justify-content:space-between;align-items:start;gap:16px}h1{font-size:26px;margin:4px 0 12px;line-height:1.3}h2{font-size:20px;margin:0 0 6px}h3{font-size:16px;margin:0 0 6px}.eyebrow{font-size:12px;color:#58746b;letter-spacing:2px}.intro,.help,.empty,small{opacity:.75}.help{font-size:12px;line-height:1.7}.create-group{display:flex;align-items:end;gap:12px;margin:24px 0}label{display:grid;gap:6px;font-size:13px}input:not([type=checkbox]),select{font:inherit;color:inherit;background:transparent;border:1px solid #84978c80;border-radius:8px;padding:10px;box-sizing:border-box;width:100%;min-height:42px}button{width:auto;flex-shrink:0;font:inherit;border:1px solid #829a8d80;background:#edf4ee;color:#203d31;border-radius:8px;padding:10px 14px;cursor:pointer;min-height:42px}button:disabled{opacity:.5;cursor:default}button:focus-visible,input:focus-visible,select:focus-visible,summary:focus-visible{outline:2px solid #36806b;outline-offset:3px}.group-visibility{display:flex;align-items:center;gap:8px}.group-visibility input{appearance:auto;width:18px;height:18px;flex:0 0 18px;margin:0}.group-card{border:1px solid #89998c55;border-radius:16px;padding:20px;margin-top:20px}.totals{display:grid;grid-template-columns:repeat(3,1fr);gap:14px;padding:18px 0;border-bottom:1px solid #89998c35}.totals div{display:grid;gap:6px}.totals strong{font-size:17px;overflow-wrap:anywhere}.account-row{display:flex;justify-content:space-between;gap:20px;padding:18px 0;border-bottom:1px solid #89998c25}.account-row:last-child{border:0}.account-row form{min-width:170px;display:grid;gap:8px}.account-row p{font-size:12px;opacity:.7;margin:0 0 8px}.account-row li{font-size:13px;margin:8px 0}.error,.delete-confirm{padding:12px;border:1px solid #c28171;border-radius:8px}.message{padding:12px;background:#7cb99220;border-radius:8px}.delete-confirm button{margin:5px}.danger{color:#924a39;margin-top:10px}summary{cursor:pointer}details form{display:grid;gap:8px;margin-top:10px}@media(max-width:600px){.account-open{padding:0;border:0;background:transparent;color:inherit;text-align:left;min-height:32px;text-decoration:underline;text-underline-offset:3px}.institution-groups{padding:16px}h1{font-size:23px}.group-card{padding:16px}.totals{grid-template-columns:1fr;gap:10px}.totals div{display:flex;justify-content:space-between;gap:10px}.account-row{flex-direction:column;gap:12px}.account-row form{grid-template-columns:minmax(0,1fr) auto;align-items:end}.account-row form small{grid-column:1/-1}.create-group label{flex:1;min-width:0}.create-group button{padding:10px}.group-card header{flex-wrap:wrap}}
</style>
