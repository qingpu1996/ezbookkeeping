import { computed, onMounted, onUnmounted, ref, watch, type Ref } from 'vue';
import services from '@/lib/services';
import { useUserStore } from '@/stores/user';
import type { AccountInfoResponse } from '@/models/account';
import { valuationBalances, type InvestmentValuation } from '@/models/investmentValuation';
export function useInvestmentValuations(accounts: Ref<AccountInfoResponse[]>) {
 const user=useUserStore();const rows=ref<InvestmentValuation[]>([]),now=ref(Date.now()),failed=ref(false),loading=ref(false);
 let sequence=0, timer: ReturnType<typeof setInterval> | undefined;
 const goldAccounts=computed(()=>accounts.value.flatMap(a=>a.type===2?a.subAccounts||[]:[a]).filter(a=>a.investmentPositionId));
 const balances=computed(()=>valuationBalances(accounts.value,rows.value,now.value));
 const note=computed(()=>{
  if (!goldAccounts.value.length) return '';
  if (loading.value && !rows.value.length) return '黄金估值读取中，当前暂按持仓成本汇总。';
  if (failed.value) return '黄金估值读取失败，当前明确按持仓成本汇总；请刷新重试。';
  const using=goldAccounts.value.filter(a=>balances.value[a.id]!==undefined).length;
  const unavailable=goldAccounts.value.some(a=>balances.value[a.id]===undefined && rows.value.find(r=>r.accountId===a.id)?.mode!=='cost');
  return (using?'总资产含黄金参考估值，非保证成交金额。':'黄金按持仓成本汇总。')+(unavailable?' 部分黄金报价未配置、过期或不可用，暂按成本汇总。':'');
 });
 async function refresh(){
  const seq=++sequence;const owner=user.currentUserBasicInfo;
  if (!owner || !goldAccounts.value.length){rows.value=[];return}
  loading.value=true;failed.value=false;
  try{const result=await services.getInvestmentValuations();if(seq===sequence && user.currentUserBasicInfo===owner)rows.value=result.data.result;}
  catch{if(seq===sequence){rows.value=[];failed.value=true;}}
  finally{if(seq===sequence){loading.value=false;now.value=Date.now();}}
 }
 function visible(){if(!document.hidden)void refresh();else {rows.value=[];now.value=Date.now();}}
 watch(()=>user.currentUserBasicInfo,()=>{++sequence;rows.value=[];failed.value=false;void refresh();});
 watch(()=>goldAccounts.value.map(a=>a.id+':'+a.balance).join(','),()=>void refresh());
 onMounted(()=>{void refresh();timer=setInterval(()=>{now.value=Date.now();if(!document.hidden)void refresh();},60000);document.addEventListener('visibilitychange',visible);});
 onUnmounted(()=>{++sequence;if(timer)clearInterval(timer);document.removeEventListener('visibilitychange',visible);});
 return {balances,note,refresh,rows,now};
}
