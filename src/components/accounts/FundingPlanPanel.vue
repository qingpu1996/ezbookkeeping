<template>
 <section class="funding-panel" aria-label="资金规划">
  <header><div><strong>资金规划</strong><small>只影响投资预算，不改变账本余额</small></div><button type="button" :disabled="busy || accountsLoading" @click="openEditor">设置保留资金与账户</button><button type="button" :disabled="busy" @click="load">刷新设置</button></header>
  <p v-if="error" role="alert">{{ error }} <button type="button" @click="load">重新读取</button></p>
  <p v-else-if="!plan || accountsLoading">正在读取资金规划…</p>
  <template v-else>
   <p v-if="!plan.configured" class="notice">待设置：请明确选择可动用账户和保留规则，再计算投资可用资金。新账户默认不计入。</p>
   <p v-else-if="summary?.invalid" class="notice">已选账户发生变化，请重新确认设置；暂不提供投资可用金额。</p>
   <div v-if="summary" class="figures">
    <div><small>可调配资金 · {{ currency }}</small><strong>{{ money(summary.included) }}</strong></div>
    <div><small>实际保留资金</small><strong>{{ plan.configured ? money(summary.reserve) : '待设置' }}</strong><small v-if="plan.configured">保底 {{ money(summary.minimum) }} 与 {{ (plan.reserveBasisPoints / 100).toFixed(2) }}% 取高</small></div>
    <div class="available"><small>投资可用资金</small><strong>{{ summary.ready ? money(summary.available) : '待确认' }}</strong></div>
    <div><small>受限或排除资金 · {{ currency }}</small><strong>{{ money(summary.excluded) }}</strong></div>
   </div>
   <p v-if="summary?.ready && summary.shortfall > 0n" class="notice">保留资金缺口：{{ money(summary.shortfall) }} {{ currency }}。当前投资可用资金为零。</p>
   <small class="explanation">实际保留资金取「固定保底」与「可调配资金 × 保留比例」的较高值；剩余为投资可用资金，最低为零。仅统计所设币种的资金余额；投资持仓、负债和其他币种不计入。请把近期生活费、还款等预留在固定保底中，未自动扣除全部负债。账户是否隐藏或从总额排除，不改变这里的明确选择。</small>
  </template>
  <dialog ref="dialog" @cancel="cancelEditor">
   <form @submit.prevent="save">
    <h3>保留资金与可动用账户</h3>
    <p>勾选允许用于投资的资金账户。公积金、专用应急账户等可以不勾选；这些钱仍保留在总资产中。</p>
    <fieldset :disabled="busy">
     <label class="reserve-label">固定保底金额 · {{ currency }}<input v-model="reserveInput" inputmode="decimal" placeholder="请输入金额，允许明确填 0" required /></label>
     <label class="reserve-label">保留比例 · %<input v-model="percentInput" inputmode="decimal" placeholder="0–100，最多两位小数" required /></label>
     <small>两者取高：比例只按下方已勾选账户的资金计算。比例设为 0 可只用固定金额；保底设为 0 可只用比例。未勾选账户不重复扣减。</small>
     <div class="account-choices">
      <label v-for="a in leaves" :key="a.id" :class="{disabled: !eligibleFundingAccount(a, currency)}">
       <input type="checkbox" v-model="selected" :value="a.id" :disabled="!eligibleFundingAccount(a, currency)" />
       <span>{{ a.name }}<small>{{ reason(a) }}</small></span><span class="account-amount">{{ showBalances ? (a.balance / 100).toFixed(2) : '••••' }} {{ a.currency }}</span>
      </label>
     </div>
    </fieldset>
    <p v-if="editError" role="alert">{{ editError }}</p>
    <footer><button type="button" :disabled="busy" @click="closeEditor">取消</button><button type="submit" :disabled="busy">{{ busy ? '保存中…' : '确认并保存设置' }}</button></footer>
   </form>
  </dialog>
 </section>
</template>
<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue';
import services from '@/lib/services';
import { useUserStore } from '@/stores/user';
import type { AccountInfoResponse } from '@/models/account';
import { fundingLeaves, eligibleFundingAccount, fundingSummary, type FundingPlan } from '@/models/fundingPlan';
import { investmentMinorUnits, investmentMoney } from '@/models/investment';
const props = defineProps<{ accounts: AccountInfoResponse[]; accountsLoading: boolean; showBalances: boolean }>();
const user = useUserStore();
const plan = ref<FundingPlan>(); const error = ref(''); const editError = ref(''); const busy = ref(false);
const dialog = ref<HTMLDialogElement>(); const selected = ref<string[]>([]); const reserveInput = ref(''); const percentInput = ref('0');
const currency = computed(() => plan.value?.configured ? plan.value.currency : user.currentUserDefaultCurrency);
const leaves = computed(() => fundingLeaves(props.accounts));
const summary = computed(() => plan.value ? fundingSummary(props.accounts, { ...plan.value, currency: currency.value }) : undefined);
let sequence = 0;
function money(value: bigint) { return props.showBalances ? investmentMoney(value.toString()) : '••••'; }
function reason(a: AccountInfoResponse) {
 if (a.investmentPositionId) return '投资持仓：不计入可用现金';
 if (a.currency !== currency.value) return '其他币种：本次不计入';
 if (!eligibleFundingAccount(a, currency.value)) return '负债或非资金账户：不计入';
 return selected.value.includes(a.id) ? '计入投资可用资金' : '受限或不用于投资';
}
async function load() {
 const seq = ++sequence; const owner = user.currentUserBasicInfo;
 error.value = ''; plan.value = undefined;
 try { const r = await services.getFundingPlan(); if (seq === sequence && user.currentUserBasicInfo === owner) plan.value = r.data.result; }
 catch { if (seq === sequence) error.value = '资金规划读取失败，暂不计算投资可用金额。'; }
}
function openEditor() {
 if (!plan.value) return;
 selected.value = plan.value.accountIds.filter(id => leaves.value.some(a => a.id === id && eligibleFundingAccount(a, currency.value)));
 reserveInput.value = plan.value.configured ? (Number(plan.value.reserve) / 100).toFixed(2) : '';
 percentInput.value = ((plan.value.reserveBasisPoints ?? 0) / 100).toFixed(2);
 editError.value = ''; dialog.value?.showModal();
}
function cancelEditor(e: Event) { if (busy.value) e.preventDefault(); }
function closeEditor() { dialog.value?.close(); }
async function save() {
 if (!plan.value || busy.value) return;
 editError.value = '';
 let reserve: string; let reserveBasisPoints: number;
 try { reserve = investmentMinorUnits(reserveInput.value); reserveBasisPoints = Number(investmentMinorUnits(percentInput.value)); if (reserveBasisPoints > 10000) throw new Error('range'); } catch { editError.value = '请输入非负保底金额和 0–100% 的比例，均最多两位小数。'; return; }
 const owner = user.currentUserBasicInfo;
 busy.value = true;
 try {
  const r = await services.saveFundingPlan({ currency: currency.value, reserve, reserveBasisPoints, accountIds: [...selected.value], version: plan.value.version });
  if (owner === user.currentUserBasicInfo) { plan.value = r.data.result; closeEditor(); }
 } catch { editError.value = '保存未确认成功。请取消后重新读取设置，再核对重试（可能是其他设备已修改或账户已变化）。'; }
 finally { busy.value = false; }
}
watch(() => user.currentUserBasicInfo, () => { closeEditor(); plan.value = undefined; void load(); });
function onVisible() { if (!document.hidden && !dialog.value?.open && !busy.value) void load(); }
onMounted(() => { void load(); document.addEventListener('visibilitychange', onVisible); });
onUnmounted(() => { ++sequence; document.removeEventListener('visibilitychange', onVisible); });
</script>
<style scoped>
.funding-panel{margin:16px 0;padding:20px;border:1px solid rgba(128,128,128,.22);border-radius:10px;background:var(--f7-card-bg-color, rgb(var(--v-theme-surface,255,255,255)));color:inherit}
header,footer{display:flex;align-items:center;gap:12px}header>div{margin-inline-end:auto}small{display:block;opacity:.75;font-size:12px}header strong{font-size:16px}.figures{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:16px;margin:20px 0}.figures strong{display:block;font-size:20px;margin-top:5px;overflow-wrap:anywhere}.available strong{color:#b87940}.notice{padding:10px;background:rgba(200,135,68,.1);border-radius:6px}button{padding:8px 12px;border:1px solid #9996;border-radius:6px;color:inherit;cursor:pointer;background:transparent}button:disabled{opacity:.5;cursor:default}.explanation{line-height:1.7}
dialog{box-sizing:border-box;width:min(640px,94vw);max-height:85vh;overflow:auto;margin:auto;padding:24px;border:1px solid #9996;border-radius:12px;background:var(--f7-card-bg-color,rgb(var(--v-theme-surface,255,255,255)));color:inherit}dialog::backdrop{background:#0006}dialog p{font-size:14px;line-height:1.7}fieldset{border:0;padding:0;margin:0}.reserve-label{display:block}.reserve-label input{display:block;width:100%;box-sizing:border-box;padding:10px;border:1px solid #9997;border-radius:6px;margin:8px 0 5px;color:inherit}.account-choices{margin:18px 0}.account-choices label{display:flex;align-items:center;gap:12px;padding:12px 0;border-bottom:1px solid #9993}.account-choices input{appearance:auto;width:18px;height:18px;flex-shrink:0;accent-color:#bd804b}.account-amount{margin-inline-start:auto;font-size:12px;white-space:nowrap}.disabled{opacity:.6}footer{justify-content:flex-end;margin-top:18px}button[type=submit]{background:#bd804b;color:white}h3{margin-top:0}p[role=alert]{color:#be3535}@media(max-width:650px){.figures{grid-template-columns:repeat(2,minmax(0,1fr))}header{align-items:flex-start;flex-direction:column}.funding-panel{margin:12px;padding:16px}.account-choices label{flex-wrap:wrap}.account-amount{padding-inline-start:30px}.figures strong{font-size:18px}}
</style>
