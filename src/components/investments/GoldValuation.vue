<template>
 <section class="valuation panel">
  <header><div><h2>黄金参考估值</h2><p>持仓成本保留不变，估值只用于展示资产和浮动盈亏。</p></div><button :disabled="loading || disabled" @click="refresh">刷新报价</button></header>
  <p v-if="error" role="alert">{{ error }}</p>
  <template v-if="valid && result">
   <div class="estimate"><strong>¥{{ investmentMoney(result.marketValue!) }}</strong><span>{{ position.quantity }} {{ position.unitName || position.unit }} × ¥{{ result.unitPrice }} / {{ position.unitName || position.unit }}</span></div>
   <p>浮动盈亏 ¥{{ investmentMoney(result.unrealized || '0') }} · {{ result.sourceName }}</p>
   <p>报价采集 / 手动参考时间：{{ stamp(result.fetchedAt) }}。这是参考估值，不保证能按此价格成交。</p>
   <p v-if="result.quote?.source === 'cmb_spot'">采用招行客户卖出参考价。MarketDay={{ result.quote.marketDay }} · QuotedPrice={{ result.quote.quotedPrice }}。{{ result.quote.marketDay ? '' : '当前标记为非交易日，不能视为实时可成交价。' }}最终以招行交易界面为准。</p>
   <details v-if="result.quote?.source === 'cmb_spot'"><summary>报价原始时间说明</summary><p>银行 NowTime：{{ result.quote.apiNowTime }}；它不代表已经确认的价格最后更新时间。采集时间也不证明银行报价刚更新。</p></details>
  </template>
  <p v-else-if="result?.mode === 'cost'">当前按持仓成本 ¥{{ investmentMoney(position.cost) }} 展示。</p>
  <p v-else>当前没有有效参考估值（未配置、报价过期或读取失败），账户总览暂按持仓成本 ¥{{ investmentMoney(position.cost) }} 汇总。</p>
  <details :open="!!config && config.settings.mode === 'cost'"><summary>估值设置</summary>
   <form v-if="form && config" @submit.prevent="save">
    <fieldset :disabled="loading || disabled">
     <label>展示方式<select v-model="form.mode"><option value="cost">只显示持仓成本</option><option value="manual">手动参考价</option><option value="automatic" :disabled="!config.automaticAvailable">自动报价：{{ config.automaticName }}{{ config.automaticAvailable ? '' : '（部署者尚未配置）' }}</option></select></label>
     <template v-if="form.mode === 'manual'">
      <label>参考卖出价（人民币 / {{ position.unitName || position.unit }}）<input v-model="form.manualPrice" inputmode="decimal" required placeholder="输入每单位参考价" /></label>
      <label>参考时间<input v-model="manualDate" type="datetime-local" required /></label>
      <label>有效期（分钟，最长7天）<input v-model.number="form.maxAgeMinutes" type="number" min="1" max="10080" required /></label>
     </template>
     <p v-if="form.mode === 'automatic'">从服务端配置的 {{ config.automaticName }} 读取；超过 {{ config.automaticMaxAgeMinutes }} 分钟的报价不用作当前估值。</p>
     <button type="submit">保存估值设置</button>
    </fieldset>
   </form>
  </details><p v-if="message" role="status">{{ message }}</p>
 </section>
</template>
<script setup lang="ts">
import { computed,onMounted,onUnmounted,ref,watch } from 'vue';
import services from '@/lib/services';
import type { InvestmentPosition } from '@/models/investment';
import { investmentMoney } from '@/models/investment';
import type { InvestmentValuation,ValuationSettings,ValuationSettingsResponse } from '@/models/investmentValuation';
const props=defineProps<{position:InvestmentPosition;disabled?:boolean}>();
const config=ref<ValuationSettingsResponse>(),form=ref<ValuationSettings>(),result=ref<InvestmentValuation>(),manualDate=ref(''),loading=ref(false),error=ref(''),message=ref(''),now=ref(Date.now());
let generation=0,timer:ReturnType<typeof setInterval>|undefined;
const valid=computed(()=>result.value?.status==='reference' && result.value.marketValue!==null && result.value.positionVersion===props.position.version && result.value.validUntil*1000>now.value);
function stamp(value?:string){return value?new Date(value).toLocaleString():'—'}
function localDate(value:number){const d=new Date(value);return new Date(value-d.getTimezoneOffset()*60000).toISOString().slice(0,16)}
async function refresh(){if(loading.value)return;const id=props.position.id;const seq=++generation;loading.value=true;error.value='';try{const r=await services.getInvestmentValuation(id);if(seq===generation)result.value=r.data.result;}catch{if(seq===generation){result.value=undefined;error.value='报价读取失败，未沿用旧报价；请重试。';}}finally{if(seq===generation){loading.value=false;now.value=Date.now();}}}
async function load(){const seq=++generation;loading.value=true;result.value=undefined;config.value=undefined;form.value=undefined;error.value='';try{const r=await services.getInvestmentValuationSettings(props.position.id);if(seq!==generation)return;config.value=r.data.result;form.value={...r.data.result.settings};manualDate.value=localDate(form.value.manualAsOf?form.value.manualAsOf*1000:Date.now());}catch{if(seq===generation)error.value='无法读取估值设置，请刷新页面。';}finally{if(seq===generation)loading.value=false;}if(seq===generation)await refresh();}
async function save(){if(!form.value || loading.value)return;loading.value=true;error.value='';message.value='';try{const req={...form.value,manualAsOf:Math.floor(new Date(manualDate.value).getTime()/1000)};if(req.mode==='manual'&&!Number.isFinite(req.manualAsOf))throw Error();await services.saveInvestmentValuationSettings(req);message.value='估值设置已保存，持仓成本和交易记录不变。';}catch{error.value='设置未保存或结果尚未确认，请重新加载设置后核对（价格、时间或版本可能无效）。';return;}finally{loading.value=false;}await load();}
watch(()=>[props.position.id,props.position.version],()=>void load());
function visible(){now.value=Date.now();if(!document.hidden)void refresh();else result.value=undefined;}
onMounted(()=>{void load();timer=setInterval(()=>{now.value=Date.now();if(!document.hidden && !props.disabled)void refresh();},60000);document.addEventListener('visibilitychange',visible)});
onUnmounted(()=>{++generation;if(timer)clearInterval(timer);document.removeEventListener('visibilitychange',visible)});
</script>
<style scoped>
.valuation{border:1px solid #8884;border-radius:16px;padding:20px;margin:20px 0}.valuation header{display:flex;justify-content:space-between;align-items:start;gap:16px}.valuation h2{margin:0 0 8px}.valuation p{font-size:13px;line-height:1.7;opacity:.85}.estimate{display:flex;flex-direction:column;gap:8px;padding:14px 0}.estimate strong{font-size:30px;color:#a56d2d}.estimate span{font-size:14px}summary{cursor:pointer;padding:12px 0}fieldset{border:0;padding:0}label{display:grid;gap:8px;margin:12px 0;font-size:14px}input,select{width:100%;box-sizing:border-box;font:inherit;color:inherit;background:transparent;padding:12px;border:1px solid #8886;border-radius:8px}button{font:inherit;flex-shrink:0;border:1px solid #8886;border-radius:8px;padding:10px 14px;background:transparent;color:inherit;cursor:pointer}button:disabled{opacity:.5}form{max-width:580px}[role=alert]{color:#ae432d}@media(max-width:600px){.valuation{padding:16px}.estimate strong{font-size:26px}}
</style>
